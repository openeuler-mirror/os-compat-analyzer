// Package collector 负责从本地系统采集各种 OS 特征数据。
package collector

import (
	"debug/elf"
	"errors"
	"io/fs"
	"log"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"

	"atomgit.com/openeuler/os-compat-analyzer/internal/model"
)

// CollectUserspaceSymbols 采集用户态动态库的符号信息。
// libDirs 指定要扫描的目录列表，默认为 /lib64, /usr/lib64, /lib, /usr/lib。
//
// 返回值：
//   - []model.UserspaceSymbol: 用户态符号列表
//   - error: 扫描或解析失败时返回错误
func CollectUserspaceSymbols(libDirs []string) ([]model.UserspaceSymbol, error) {
	if len(libDirs) == 0 {
		libDirs = []string{"/lib64", "/usr/lib64", "/lib", "/usr/lib"}
	}

	// 第一步：收集所有 .so 文件路径
	soFiles, err := findSOFiles(libDirs)
	if err != nil {
		return nil, err
	}

	if len(soFiles) == 0 {
		log.Printf("INFO: no .so files found in directories: %v", libDirs)
		return []model.UserspaceSymbol{}, nil
	}

	log.Printf("INFO: found %d .so files, starting concurrent parsing...", len(soFiles))

	// 第二步：并发解析 ELF 文件
	symbols, err := parseSOFilesConcurrently(soFiles)
	if err != nil {
		return nil, err
	}

	// 按 SoPath 和 SymbolName 排序，保证输出顺序一致
	sort.Slice(symbols, func(i, j int) bool {
		if symbols[i].SoPath != symbols[j].SoPath {
			return symbols[i].SoPath < symbols[j].SoPath
		}
		return symbols[i].SymbolName < symbols[j].SymbolName
	})

	log.Printf("INFO: collected %d userspace symbols", len(symbols))
	return symbols, nil
}

// isSOFile 检查路径是否为 .so 文件（包括 .so, .so.x, .so.x.y.z 等形式）
func isSOFile(path string) bool {
	// 检查是否包含 .so
	idx := strings.Index(path, ".so")
	if idx == -1 {
		return false
	}

	// 检查 .so 后面是否为空（直接是 .so 文件）
	// 或者是 .so.数字（版本号）
	rest := path[idx+3:]
	if len(rest) == 0 {
		return true
	}

	// 如果后面是 . 开头，后面应该是数字版本号
	// 必须以 .数字 开头，且不能以 . 结尾
	if len(rest) > 0 && rest[0] == '.' && len(rest) > 1 {
		// 检查剩余部分是否全是数字和点号
		for _, c := range rest[1:] {
			if c != '.' && (c < '0' || c > '9') {
				return false
			}
		}
		// 不能以 . 结尾
		if rest[len(rest)-1] == '.' {
			return false
		}
		return true
	}

	return false
}

// findSOFiles 查找指定目录下的所有 .so 文件
func findSOFiles(libDirs []string) ([]string, error) {
	soFiles := make([]string, 0)
	seen := make(map[string]bool)

	for _, dir := range libDirs {
		err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				// 忽略权限错误
				if errors.Is(err, fs.ErrPermission) {
					log.Printf("WARN: permission denied: %s", path)
					return nil
				}
				return err
			}

			// 只处理 .so 文件（包括 .so, .so.x, .so.x.y.z 等形式）
			// 匹配: libfoo.so, libfoo.so.1, libfoo.so.1.2
			if !d.IsDir() && isSOFile(path) {
				if !seen[path] {
					seen[path] = true
					soFiles = append(soFiles, path)
				}
			}
			return nil
		})
		if err != nil {
			log.Printf("WARN: failed to walk directory %s: %v", dir, err)
		}
	}

	return soFiles, nil
}

// parseSOFilesConcurrently 并发解析多个 .so 文件
func parseSOFilesConcurrently(soFiles []string) ([]model.UserspaceSymbol, error) {
	// 计算并发数：CPU 核心数 * 2
	maxConcurrency := runtime.NumCPU() * 2
	if maxConcurrency < 4 {
		maxConcurrency = 4
	}

	// 使用带缓冲的 channel 收集结果
	resultChan := make(chan []model.UserspaceSymbol, len(soFiles))

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, maxConcurrency)

	for _, soFile := range soFiles {
		wg.Add(1)
		go func(path string) {
			defer wg.Done()

			// 信号量控制并发
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			symbols, err := parseSOFile(path)
			if err != nil {
				log.Printf("WARN: failed to parse %s: %v", path, err)
				return
			}

			if len(symbols) > 0 {
				resultChan <- symbols
			}
		}(soFile)
	}

	// 等待所有 goroutine 完成
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// 收集结果
	var allSymbols []model.UserspaceSymbol
	for symbols := range resultChan {
		allSymbols = append(allSymbols, symbols...)
	}

	return allSymbols, nil
}

// 从 Info 字段提取符号类型
func getSymbolType(info byte) uint8 {
	return info & 0x0F
}

// 从 Info 字段提取符号绑定信息
func getSymbolBind(info byte) uint8 {
	return info >> 4
}

// parseSOFile 解析单个 .so 文件，提取导出符号
func parseSOFile(soPath string) ([]model.UserspaceSymbol, error) {
	// 打开 ELF 文件
	f, err := elf.Open(soPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// 获取动态符号表
	dynsyms, err := f.DynamicSymbols()
	if err != nil {
		return nil, err
	}

	// 如果没有动态符号表，跳过
	if len(dynsyms) == 0 {
		return nil, nil
	}

	symbols := make([]model.UserspaceSymbol, 0, len(dynsyms))
	for _, sym := range dynsyms {
		// 从 Info 字段提取绑定信息
		bind := getSymbolBind(sym.Info)

		// 只提取 STB_GLOBAL (2) 或 STB_WEAK (3) 类型
		if bind != 2 && bind != 3 {
			continue
		}

		// 从 Info 字段提取类型
		symType := getSymbolType(sym.Info)

		// 只提取函数 (2) 或对象 (1) 类型
		if symType != 2 && symType != 1 {
			continue
		}

		// 忽略空符号名
		if sym.Name == "" {
			continue
		}

		// 优先使用符号自带的版本信息（来自 DynamicSymbols）
		version := sym.Version
		if version == "" {
			// 尝试从符号名推断版本（如 GLIBC_2.17）
			version = extractVersionFromName(sym.Name)
		}

		userspaceSym := model.UserspaceSymbol{
			SoPath:        soPath,
			SymbolName:    sym.Name,
			SymbolVersion: version,
		}
		symbols = append(symbols, userspaceSym)
	}

	return symbols, nil
}

// extractVersionFromName 从符号名中提取版本信息
// 例如：__pthread_create_0 -> GLIBC_2.17
func extractVersionFromName(name string) string {
	// 尝试匹配常见的版本后缀模式，如 _2.17, _2.14 等
	idx := strings.Index(name, "_2.")
	if idx > 0 {
		// 向前查找可能的版本前缀
		start := idx
		for start > 0 {
			if name[start-1] == '_' || name[start-1] == '$' {
				start--
				break
			}
			start--
			if start < idx-20 {
				break
			}
		}

		prefix := name[start:idx]
		// 检查是否包含已知的前缀
		knownPrefixes := []string{"GLIBC", "UCLIBC", "CURL", "OPENSSL", "LLVM"}
		for _, p := range knownPrefixes {
			if strings.Contains(prefix, p) {
				// 提取完整版本
				rest := name[idx+1:]
				end := 0
				for end < len(rest) && (rest[end] == '.' || (rest[end] >= '0' && rest[end] <= '9')) {
					end++
				}
				if end > 0 {
					return p + "_" + rest[:end]
				}
			}
		}
	}
	return ""
}
