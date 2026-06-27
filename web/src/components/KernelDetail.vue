<script setup>
import { ref, computed, onMounted, onUnmounted, defineComponent, h, watch, nextTick } from 'vue'
import { Search } from '@element-plus/icons-vue'
import { ElTooltip } from 'element-plus'

// 文本溢出时悬浮显示完整内容
const OverflowCell = defineComponent({
  name: 'OverflowCell',
  props: {
    content: { type: String, default: '' }
  },
  setup(props) {
    const overflow = ref(false)
    const handleEnter = (e) => {
      const el = e.currentTarget
      overflow.value = el.scrollWidth > el.clientWidth
    }
    return () => h(ElTooltip, {
      content: String(props.content ?? ''),
      disabled: !overflow.value,
      placement: 'top',
      showAfter: 300
    }, {
      default: () => h('div', { class: 'ellipsis-cell', onMouseenter: handleEnter }, props.content)
    })
  }
})

const activeTab = ref('syscall')

const diffData = ref(null)
const loading = ref(true)
const error = ref(null)
const symbolSearch = ref('')
const statusFilter = ref('')
const tableWidth = ref(800)
const tableHeight = ref(400)
const tableWrapperRef = ref(null)

function updateTableSize() {
  if (tableWrapperRef.value) {
    tableWidth.value = tableWrapperRef.value.clientWidth
    tableHeight.value = tableWrapperRef.value.clientHeight
  }
}

let resizeObserver = null

// 页面加载时获取数据
onMounted(() => {
  loadData()
  if (typeof ResizeObserver !== 'undefined') {
    resizeObserver = new ResizeObserver(() => {
      updateTableSize()
    })
    if (tableWrapperRef.value) {
      resizeObserver.observe(tableWrapperRef.value)
    }
  } else {
    updateTableSize()
    window.addEventListener('resize', updateTableSize)
  }
})

onUnmounted(() => {
  if (resizeObserver) {
    resizeObserver.disconnect()
  } else {
    window.removeEventListener('resize', updateTableSize)
  }
})

// 切换到 kernel tab 时重新计算表格尺寸
watch(activeTab, (val) => {
  if (val === 'kernel') {
    nextTick(updateTableSize)
  }
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

// 合并所有内核符号（仅A有、仅B有、CRC冲突）
const allKernelSymbols = computed(() => {
  const data = kernelSymbolData.value
  if (!data) return []
  const rows = []

  // 仅 A 有
  ;(data.onlyInA || []).forEach(s => {
    rows.push({
      name: s.name,
      module: s.module,
      crcInA: s.crc || '-',
      crcInB: '-',
      status: '仅A有'
    })
  })
  // 仅 B 有
  ;(data.onlyInB || []).forEach(s => {
    rows.push({
      name: s.name,
      module: s.module,
      crcInA: '-',
      crcInB: s.crc || '-',
      status: '仅B有'
    })
  })
  // CRC 冲突
  ;(data.modified || []).forEach(s => {
    rows.push({
      name: s.name,
      module: s.module,
      crcInA: s.crcInA || '-',
      crcInB: s.crcInB || '-',
      status: 'CRC冲突'
    })
  })

  // 按符号名排序
  rows.sort((a, b) => (a.name || '').localeCompare(b.name || ''))
  return rows
})

// 筛选后的内核符号
const filteredKernelSymbols = computed(() => {
  let data = allKernelSymbols.value

  // 按符号名搜索
  if (symbolSearch.value) {
    const search = symbolSearch.value.toLowerCase()
    data = data.filter(row => row.name.toLowerCase().includes(search))
  }

  // 按状态筛选
  if (statusFilter.value) {
    data = data.filter(row => row.status === statusFilter.value)
  }

  return data
})

// 表格列定义
const kernelSymbolColumns = [
  { key: 'name', dataKey: 'name', title: '符号名', width: 250 },
  { key: 'module', dataKey: 'module', title: '所属模块', width: 300 },
  { key: 'crcInA', dataKey: 'crcInA', title: 'OS A CRC', width: 150 },
  { key: 'crcInB', dataKey: 'crcInB', title: 'OS B CRC', width: 150 },
  { key: 'status', dataKey: 'status', title: '状态', width: 120 }
]

// 状态筛选选项
const statusOptions = [
  { label: '全部', value: '' },
  { label: 'CRC冲突', value: 'CRC冲突' },
  { label: '仅A有', value: '仅A有' },
  { label: '仅B有', value: '仅B有' },
]

// 行样式函数
function getRowClass({ row }) {
  if (row.status === 'CRC冲突') {
    return 'crc-conflict-row'
  }
  return ''
}

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
          <el-card>
            <template #header>
              <div class="header-row">
                <span>内核符号差异详情</span>
                <span class="stats">共 {{ filteredKernelSymbols.length }} 条</span>
              </div>
            </template>

            <!-- 搜索和筛选 -->
            <div class="filter-bar">
              <el-input
                v-model="symbolSearch"
                placeholder="搜索符号名..."
                clearable
                style="width: 300px;"
              >
                <template #prefix>
                  <el-icon><Search /></el-icon>
                </template>
              </el-input>

              <el-select
                v-model="statusFilter"
                placeholder="按状态筛选"
                clearable
                style="width: 150px; margin-left: 10px;"
              >
                <el-option
                  v-for="item in statusOptions"
                  :key="item.value"
                  :label="item.label"
                  :value="item.value"
                />
              </el-select>
            </div>

            <!-- 内核符号表格 -->
            <div ref="tableWrapperRef" class="table-wrapper">
              <el-table-v2
                :columns="kernelSymbolColumns"
                :data="filteredKernelSymbols"
                :width="tableWidth"
                :height="tableHeight"
                :row-height="40"
                :row-class-name="getRowClass"
                class="symbol-table"
              >
                <template #cell="{ column, rowData }">
                  <template v-if="column.key === 'status'">
                    <el-tag
                      v-if="rowData.status === 'CRC冲突'"
                      type="danger"
                      size="small"
                    >
                      {{ rowData.status }}
                    </el-tag>
                    <el-tag
                      v-else-if="rowData.status === '仅A有' || rowData.status === '仅B有'"
                      type="warning"
                      size="small"
                    >
                      {{ rowData.status }}
                    </el-tag>
                    <el-tag v-else type="info" size="small">
                      {{ rowData.status }}
                    </el-tag>
                  </template>
                  <template v-else>
                    <OverflowCell :content="String(rowData[column.dataKey] ?? '')" />
                  </template>
                </template>
              </el-table-v2>
            </div>
          </el-card>
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

.kernel-content > .el-card {
  height: 100%;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.kernel-content > .el-card :deep(.el-card__body) {
  flex: 1;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.header-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.stats {
  font-size: 14px;
  color: #666;
}

.filter-bar {
  margin-bottom: 15px;
  display: flex;
  align-items: center;
}

.table-wrapper {
  flex: 1;
  min-height: 0;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  position: relative;
}

.symbol-table {
  flex: 1;
}

.table-wrapper :deep(.el-table-v2__empty) {
  width: 100% !important;
  background-color: #ffffff;
}

.table-wrapper :deep(.el-empty) {
  width: 100%;
}

.table-wrapper :deep(.ellipsis-cell) {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  width: 100%;
}

:deep(.crc-conflict-row) {
  background-color: #fef0f0;
}

:deep(.el-table-v2__row) {
  cursor: pointer;
}
</style>
