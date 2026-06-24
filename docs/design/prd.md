# OS 兼容性检查工具 需求规格说明书

## 一、 产品概述
*   **产品定位**：面向 OS 开发工程师的命令行诊断与可视化对比工具。
*   **运行环境**：目标机（RedHat/RPM 系 Linux OS）。
*   **核心价值**：以“快照”形式提取 OS 的内核态与用户态接口特征，生成标准化 JSON 数据；支持任意两个 OS 报告的精准 Diff，并**通过嵌入式前端生成直观的可视化离线 HTML 报告**，辅助评估迁移成本与升级风险。

## 二、 命令行交互设计 (CLI) 与工作流
工具由 Golang 开发，设计为单一静态链接二进制文件。

```bash
# 1. 在目标机上采集数据，生成标准数据集
os-compat-analyzer collect -o ./os_a.json
os-compat-analyzer collect -o ./os_b.json

# 2. 对比两个数据集，生成可视化前端报告（核心功能）
os-compat-analyzer report ./os_a.json ./os_b.json -o ./compat_report.html
```

## 三、 后端核心功能需求 (Golang)

### 模块 1：本地 OS 特征采集
*   **Kernel Syscall 采集**：获取当前架构的所有系统调用编号与名称。
*   **Kernel Symbols 采集**：解析内核导出符号表（如 `Module.symvers`），**必须提取符号的 CRC 校验值**。
*   **Userspace 动态库符号采集**：扫描系统级 `.so` 文件，提取导出函数及**符号版本**（如 `@@GLIBC_2.17`），过滤掉非 GLOBAL 符号。
*   **RPM 包版本采集**：调用 `rpm` 命令获取已安装包的 Name、Version、Release、Arch。
*   **元数据**：OS 名称、内核版本、架构、采集时间。

### 模块 2：标准化数据格式
输出结构化 JSON，作为前后端的数据契约，需具备良好的扩展性。

### 模块 3：差异比对引擎
*   **Syscall Diff**：集合对比，输出 A 特有、B 特有、共有。
*   **Kernel Symbols Diff**：按符号名对比，**精准识别同名符号的 CRC 不一致**（标记为 High Risk，会导致 .ko 加载失败）。
*   **Userspace Symbols Diff**：按 `.so` 文件名聚合，识别符号新增、删除，以及**符号版本的降级/升级**。
*   **RPM Diff**：包级别的增删，同名包的版本号大小比较。

---

## 四、 前端可视化需求

### 4.1 整体形态与技术要求
*   **交付形态**：单个无外部依赖的 `.html` 文件（内嵌 CSS、JS，甚至可将 VFS 图标转 Base64）。
*   **数据加载**：由 Golang 后端在执行 `report` 命令时，将计算好的 Diff JSON 结果直接注入到 HTML 的 `<script>` 标签中（`window.__INITIAL_STATE__ = {...}`），实现双击即可看，无需起 Web Server。
*   **前端技术栈建议**：Vue 3 + Vite（构建配置开启 `build.inlineStyles` 和脚本内联） + Element Plus + ECharts。

### 4.2 页面布局与 UI 设计
采用经典的“左导航 + 右内容区”的后台管理类布局，顶部固定展示两个 OS 的元数据对比卡片（OS A vs OS B）。

#### 4.2.1 全局概览面板
*   **兼容性雷达图/评分**：根据四个维度的差异率，计算一个粗略的“兼容性评分”（例如：Kernel ABI影响度、用户态API影响度等）。
*   **核心风险数字看板**：直观展示四个关键数字卡片：
    *   🔴 Kernel CRC 冲突数（最严重）
    *   🟠 用户态 API 缺失数
    *   🟡 RPM 包降级数
    *   🟢 RPM 包升级数

#### 4.2.2 Kernel 兼容性详情页
*   **Syscall 列表**：表格展示，支持筛选（A有B无 / B有A无）。
*   **导出符号列表 (重点)**：
    *   表格列：符号名、所属模块、A的CRC、B的CRC、状态。
    *   **特殊交互**：默认只显示“有差异”的符号。对于 CRC 不一致的符号，行背景标红，并提供悬浮提示：“该符号结构体可能发生变更，会导致内核模块加载失败”。

#### 4.2.3 Userspace 动态库详情页
*   **左侧树形控件**：按动态库文件名（如 `libc.so.6`, `libz.so.1`）进行分类聚合，显示每个库的差异数量红点。
*   **右侧详情表格**：选中某个 `.so` 后，展示其导出符号的差异。
    *   重点高亮：**版本降级**（如 A 是 `GLIBC_2.34`，B 是 `GLIBC_2.17`，可能导致高版本编译的程序在 B 上跑不起来）。
    *   支持按符号名搜索。

#### 4.2.4 RPM 软件包详情页
*   表格列：包名、架构、OS A 版本、OS B 版本、差异状态（新增/删除/升级/降级）。
*   交互：支持按包名模糊搜索；支持按“状态”列进行筛选过滤；支持点击表头按版本号排序。

---

## 五、 非功能需求

1.  **后端性能**：全量扫描 `/usr/lib64` 等目录时，Golang 需使用协程池并发解析 ELF 文件，单机采集时间控制在 30 秒内。
2.  **前端渲染性能**：Kernel Symbols 和 Userspace Symbols 可能有数万条差异。前端表格**必须使用虚拟滚动**，禁止一次性渲染上万行 DOM 导致浏览器卡死。
3.  **权限降级提示**：若非 root 运行 `collect`，导致 `/proc/kallsyms` 等读不到，CLI 需打 Warning，且前端报告顶部需展示醒目的黄色警告条：“⚠️ 本报告内核数据采集于非 Root 环境，可能缺失部分符号”。
4.  **安全性**：由于 HTML 报告可能通过邮件/网盘流转，前端严禁使用 `v-html` 直接渲染未经转义的用户输入（虽然当前数据全是系统提取的，但要养成安全习惯），防止 XSS。

---

## 六、 技术架构 (Golang + Vue3)

*   **后端**：
    *   CLI 框架：`github.com/spf13/cobra`
    *   ELF 解析：`github.com/spf13/afero` + 标准库 `debug/elf`
    *   前端资源嵌入：使用 `embed.FS` 将 Vue 构建出的 `dist/index.html` 嵌入到 Go Binary 中。
*   **前端工程化**：
    *   由于需要内嵌到 Go 中，前端项目单独建一个目录（如 `web/`）。
    *   修改 Vite 配置：`build.cssCodeSplit: false`, `build.assetsInlineLimit: 100000000`（强制所有资源转 base64 内联），最终只输出一个 `index.html`。
    *   Go 程序在执行 `report` 命令时，读取嵌入的 `index.html` 模板，使用 `html/template` 将 Diff JSON 序列化替换进模板中，写入目标文件。