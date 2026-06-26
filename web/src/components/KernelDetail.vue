<script setup>
import { ref, computed, onMounted } from 'vue'

const activeTab = ref('syscall')

const diffData = ref(null)
const loading = ref(true)
const error = ref(null)
const filterType = ref('modified')

// 页面加载时获取数据
onMounted(() => {
  loadData()
})

function loadData() {
  try {
    const scripts = document.querySelectorAll('script[type="application/json"]')
    for (const script of scripts) {
      if (script.textContent.trim()) {
        const rawData = JSON.parse(script.textContent)
        diffData.value = rawData.diffResult || rawData
        break
      }
    }
    if (!diffData.value) {
      if (window.__INITIAL_STATE__) {
        diffData.value = window.__INITIAL_STATE__.diffResult || window.__INITIAL_STATE__
      }
    }
    if (!diffData.value) {
      error.value = '未找到差异数据'
    }
  } catch (e) {
    error.value = '解析数据失败: ' + e.message
    console.error(e)
  } finally {
    loading.value = false
  }
}

// 系统调用数据
const syscallData = computed(() => {
  if (!diffData.value?.syscallsDiff) return { onlyInA: [], onlyInB: [] }
  return diffData.value.syscallsDiff
})

// 内核符号数据
const kernelSymbolData = computed(() => {
  if (!diffData.value?.kernelSymbolsDiff) {
    return { onlyInA: [], onlyInB: [], modified: [] }
  }
  return diffData.value.kernelSymbolsDiff
})

// 根据筛选类型获取内核符号
const filteredKernelSymbols = computed(() => {
  const data = kernelSymbolData.value
  if (!data) return []

  switch (filterType.value) {
    case 'onlyInA':
      return data.onlyInA || []
    case 'onlyInB':
      return data.onlyInB || []
    case 'modified':
      return data.modified || []
    default:
      return data.modified || []
  }
})

// 内核符号表格列定义
const kernelSymbolColumns = [
  { key: 'name', label: '符号名', width: '200' },
  { key: 'module', label: '所属模块', width: '250' },
  { key: 'crcInA', label: 'OS A CRC', width: '150' },
  { key: 'crcInB', label: 'OS B CRC', width: '150' },
  { key: 'status', label: '状态', width: '120' },
]

// 行样式函数
function getRowClass({ row }) {
  if (row.crcInA && row.crcInB && row.crcInA !== row.crcInB) {
    return 'crc-conflict-row'
  }
  return ''
}

// 筛选选项
const filterOptions = [
  { label: 'CRC 冲突', value: 'modified' },
  { label: '仅 A 有', value: 'onlyInA' },
  { label: '仅 B 有', value: 'onlyInB' },
]

// 跳转到指定标签页
function goToTab(tab) {
  activeTab.value = tab
}

defineExpose({
  activeTab
})
</script>

<template>
  <div class="kernel-page">
    <!-- Tab 切换 -->
    <el-tabs v-model="activeTab" type="card">
      <el-tab-pane label="Syscall 列表" name="syscall">
        <div class="tab-content">
          <el-row :gutter="20">
            <el-col :span="12">
              <el-card>
                <template #header>
                  <span>仅在 OS A 中存在 ({{ syscallData.onlyInA?.length || 0 }})</span>
                </template>
                <el-table
                  :data="syscallData.onlyInA || []"
                  :virtual-scrollbar="true"
                >
                  <el-table-column prop="number" label="编号" width="100" />
                  <el-table-column prop="name" label="名称" />
                </el-table>
              </el-card>
            </el-col>
            <el-col :span="12">
              <el-card>
                <template #header>
                  <span>仅在 OS B 中存在 ({{ syscallData.onlyInB?.length || 0 }})</span>
                </template>
                <el-table
                  :data="syscallData.onlyInB || []"
                  :virtual-scrollbar="true"
                >
                  <el-table-column prop="number" label="编号" width="100" />
                  <el-table-column prop="name" label="名称" />
                </el-table>
              </el-card>
            </el-col>
          </el-row>
        </div>
      </el-tab-pane>

      <el-tab-pane label="Kernel Symbols" name="kernel">
        <div class="tab-content kernel-content">
          <!-- 筛选器 -->
          <div class="filter-bar">
            <span>筛选: </span>
            <el-select v-model="filterType" placeholder="请选择" style="width: 200px;">
              <el-option
                v-for="item in filterOptions"
                :key="item.value"
                :label="item.label"
                :value="item.value"
              />
            </el-select>
            <span style="margin-left: 20px;">当前显示: {{ filteredKernelSymbols.length }} 条</span>
          </div>

          <!-- 内核符号表格 - 使用 el-table-v2 虚拟化 -->
          <div class="table-wrapper">
            <el-table-v2
              :columns="kernelSymbolColumns"
              :data="filteredKernelSymbols"
              :width="1200"
              height="100%"
              :row-height="40"
              :row-class-name="getRowClass"
              class="symbol-table"
            >
            <template #cell="{ column, rowData }">
              <template v-if="column.key === 'name'">
                {{ rowData.name }}
              </template>
              <template v-else-if="column.key === 'module'">
                {{ rowData.module }}
              </template>
              <template v-else-if="column.key === 'crcInA'">
                {{ rowData.crcInA || rowData.crc || '-' }}
              </template>
              <template v-else-if="column.key === 'crcInB'">
                {{ rowData.crcInB || '-' }}
              </template>
              <template v-else-if="column.key === 'status'">
                <el-tag v-if="rowData.crcInA !== rowData.crcInB" type="danger" size="small">
                  CRC 冲突
                </el-tag>
                <el-tag v-else type="info" size="small">
                  正常
                </el-tag>
              </template>
            </template>
            </el-table-v2>
          </div>
        </div>
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<style scoped>
.kernel-page {
  margin-top: 10px;
  height: 100%;
  min-height: 100%;
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
}

.kernel-page > .el-tabs {
  flex: 1;
  display: flex;
  flex-direction: column;
}

.kernel-page > .el-tabs :deep(.el-tabs__content) {
  flex: 1;
  overflow: hidden;
}

.kernel-page > .el-tabs :deep(.el-tab-pane) {
  height: 100%;
}

.tab-content {
  padding: 10px 0;
  height: 100%;
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
}

.tab-content > .el-row {
  flex: 1;
  min-height: 0;
}

.tab-content > .el-row > .el-col {
  height: 100%;
  min-height: 0;
}

.tab-content .el-card {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.tab-content :deep(.el-table),
.tab-content :deep(.el-table-v2) {
  flex: 1;
}

.kernel-content {
  flex: 1;
  min-height: 0;
}

.kernel-content .table-wrapper {
  flex: 1;
  min-height: 0;
  overflow: hidden;
}

.kernel-content .symbol-table {
  height: 100%;
}

.filter-bar {
  margin-bottom: 15px;
  padding: 10px;
  background: #f5f7fa;
  border-radius: 4px;
}

:deep(.crc-conflict-row) {
  background-color: #fef0f0;
}

:deep(.el-table-v2__row) {
  cursor: pointer;
}

:deep(.el-table-v2__empty) {
  background-color: #ffffff;
}
</style>
