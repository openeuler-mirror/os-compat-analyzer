<script setup>
import { ref, computed, onMounted, onUnmounted, defineComponent, h } from 'vue'
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

const diffData = ref(null)
const loading = ref(true)
const searchText = ref('')
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
  } catch (e) {
    console.error('加载数据失败:', e)
  } finally {
    loading.value = false
  }
}

// 合并所有动态库的符号数据（含相同）
const allSymbols = computed(() => {
  if (!diffData.value?.userspaceSymbolsDiff?.bySoPath) return []

  const bySoPath = diffData.value.userspaceSymbolsDiff.bySoPath
  const rows = []

  for (const path of Object.keys(bySoPath)) {
    const group = bySoPath[path]
    const libName = path.split('/').pop() || path

    // 相同（两个 OS 中版本一致）
    ;(group.common || []).forEach(s => {
      rows.push({
        libName,
        symbolName: s.symbolName || s.SymbolName,
        versionInA: s.symbolVersion || s.SymbolVersion || '-',
        versionInB: s.symbolVersion || s.SymbolVersion || '-',
        status: '相同'
      })
    })
    // 仅 A 有
    ;(group.onlyInA || []).forEach(s => {
      rows.push({
        libName,
        symbolName: s.symbolName || s.SymbolName,
        versionInA: s.symbolVersion || s.SymbolVersion || '-',
        versionInB: '-',
        status: '仅A有'
      })
    })
    // 仅 B 有
    ;(group.onlyInB || []).forEach(s => {
      rows.push({
        libName,
        symbolName: s.symbolName || s.SymbolName,
        versionInA: '-',
        versionInB: s.symbolVersion || s.SymbolVersion || '-',
        status: '仅B有'
      })
    })
    // 版本变化
    ;(group.modified || []).forEach(s => {
      rows.push({
        libName,
        symbolName: s.symbolName,
        versionInA: s.versionInA,
        versionInB: s.versionInB,
        status: isVersionDowngrade(s.versionInA, s.versionInB) ? '版本降级' : '版本升级'
      })
    })
  }

  // 按动态库名称、符号名排序，保证输出顺序一致
  rows.sort((a, b) => {
    const cmp = (a.libName || '').localeCompare(b.libName || '')
    if (cmp !== 0) return cmp
    return (a.symbolName || '').localeCompare(b.symbolName || '')
  })
  return rows
})

// 筛选后的数据
const filteredSymbols = computed(() => {
  let data = allSymbols.value

  // 按动态库名称搜索
  if (searchText.value) {
    const search = searchText.value.toLowerCase()
    data = data.filter(row => row.libName.toLowerCase().includes(search))
  }

  // 按状态筛选
  if (statusFilter.value) {
    data = data.filter(row => row.status === statusFilter.value)
  }

  return data
})

function isVersionDowngrade(verA, verB) {
  // 简单的版本比较：GLIBC_2.34 > GLIBC_2.17
  if (!verA || !verB) return false
  const numA = parseFloat(verA.replace(/[^0-9.]/g, ''))
  const numB = parseFloat(verB.replace(/[^0-9.]/g, ''))
  return numA > numB
}

// 表格列定义
const columns = [
  { key: 'libName', dataKey: 'libName', title: '动态库', width: 240 },
  { key: 'symbolName', dataKey: 'symbolName', title: '符号名', width: 440 },
  { key: 'versionInA', dataKey: 'versionInA', title: 'OS A 版本', width: 140 },
  { key: 'versionInB', dataKey: 'versionInB', title: 'OS B 版本', width: 140 },
  { key: 'status', dataKey: 'status', title: '状态', width: 100 }
]

// 状态筛选选项
const statusOptions = [
  { label: '全部', value: '' },
  { label: '相同', value: '相同' },
  { label: '仅A有', value: '仅A有' },
  { label: '仅B有', value: '仅B有' },
  { label: '版本升级', value: '版本升级' },
  { label: '版本降级', value: '版本降级' },
]
</script>

<template>
  <div class="userspace-page">
    <el-card>
      <template #header>
        <div class="header-row">
          <span>动态库符号差异详情</span>
          <span class="stats">共 {{ filteredSymbols.length }} 条</span>
        </div>
      </template>

      <!-- 搜索和筛选 -->
      <div class="filter-bar">
        <el-input
          v-model="searchText"
          placeholder="搜索动态库名称..."
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

      <!-- 符号表格 -->
      <div ref="tableWrapperRef" class="table-wrapper">
        <el-table-v2
          :columns="columns"
          :data="filteredSymbols"
          :width="tableWidth"
          :height="tableHeight"
          :row-height="40"
          class="symbol-table"
        >
          <template #cell="{ column, rowData }">
            <template v-if="column.key === 'status'">
              <el-tag
                v-if="rowData.status === '相同'"
                type="success"
                size="small"
              >
                {{ rowData.status }}
              </el-tag>
              <el-tag
                v-else-if="rowData.status === '版本降级'"
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
              <el-tag v-else type="primary" size="small">
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
</template>

<style scoped>
.userspace-page {
  margin-top: 10px;
  height: 100%;
  box-sizing: border-box;
}

.userspace-page > .el-card {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.userspace-page > .el-card :deep(.el-card__body) {
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
</style>
