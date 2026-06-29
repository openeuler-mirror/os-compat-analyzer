<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'

const diffData = ref(null)
const loading = ref(true)
const selectedNode = ref(null)
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
    // 初始化选中第一个节点
    if (diffData.value?.userspaceSymbolsDiff?.bySoPath) {
      const paths = Object.keys(diffData.value.userspaceSymbolsDiff.bySoPath)
      if (paths.length > 0) {
        selectedNode.value = paths[0]
      }
    }
  } catch (e) {
    console.error('加载数据失败:', e)
  } finally {
    loading.value = false
  }
}

// 树形数据
const treeData = computed(() => {
  if (!diffData.value?.userspaceSymbolsDiff?.bySoPath) return []

  const bySoPath = diffData.value.userspaceSymbolsDiff.bySoPath
  return Object.keys(bySoPath).map(path => {
    const group = bySoPath[path]
    const diffCount = (group.onlyInA?.length || 0) +
                      (group.onlyInB?.length || 0) +
                      (group.modified?.length || 0)
    // 提取文件名
    const fileName = path.split('/').pop() || path
    return {
      id: path,
      label: fileName,
      diffCount,
      path: path
    }
  })
})

// 当前选中的节点数据
const currentSymbolData = computed(() => {
  if (!selectedNode.value || !diffData.value?.userspaceSymbolsDiff?.bySoPath) {
    return { onlyInA: [], onlyInB: [], modified: [] }
  }
  return diffData.value.userspaceSymbolsDiff.bySoPath[selectedNode.value] || { onlyInA: [], onlyInB: [], modified: [] }
})

// 合并所有差异符号，按状态分组后合并排序
const filteredSymbols = computed(() => {
  const data = currentSymbolData.value
  const rows = []

  // 仅 A 有
  ;(data.onlyInA || []).forEach(s => {
    rows.push({
      symbolName: s.symbolName || s.SymbolName,
      versionInA: s.symbolVersion || s.SymbolVersion || '-',
      versionInB: '-',
      status: '仅A有'
    })
  })
  // 仅 B 有
  ;(data.onlyInB || []).forEach(s => {
    rows.push({
      symbolName: s.symbolName || s.SymbolName,
      versionInA: '-',
      versionInB: s.symbolVersion || s.SymbolVersion || '-',
      status: '仅B有'
    })
  })
  // 版本变化
  ;(data.modified || []).forEach(s => {
    rows.push({
      symbolName: s.symbolName,
      versionInA: s.versionInA,
      versionInB: s.versionInB,
      status: isVersionDowngrade(s.versionInA, s.versionInB) ? '版本降级' : '版本变化'
    })
  })

  // 按符号名排序，保证输出顺序一致
  rows.sort((a, b) => (a.symbolName || '').localeCompare(b.symbolName || ''))
  return rows
})

function isVersionDowngrade(verA, verB) {
  // 简单的版本比较：GLIBC_2.34 > GLIBC_2.17
  if (!verA || !verB) return false
  const numA = parseFloat(verA.replace(/[^0-9.]/g, ''))
  const numB = parseFloat(verB.replace(/[^0-9.]/g, ''))
  return numA > numB
}

// 树节点点击
function handleNodeClick(node) {
  selectedNode.value = node.path
}
</script>

<template>
  <div class="userspace-page">
    <el-row :gutter="20">
      <!-- 左侧树形控件 -->
      <el-col :span="6">
        <el-card class="tree-card">
          <template #header>
            <span>动态库列表</span>
          </template>
          <el-tree
            :data="treeData"
            :props="{
              children: 'children',
              label: 'label'
            }"
            @node-click="handleNodeClick"
            default-expand-all
          >
            <template #default="{ node, data }">
              <span class="tree-node">
                <span>{{ node.label }}</span>
                <el-badge
                  v-if="data.diffCount > 0"
                  :value="data.diffCount"
                  type="danger"
                  class="tree-badge"
                />
              </span>
            </template>
          </el-tree>
        </el-card>
      </el-col>

      <!-- 右侧表格 -->
      <el-col :span="18">
        <el-card>
          <template #header>
            <span>符号差异 - {{ selectedNode?.split('/').pop() }}</span>
          </template>

          <!-- 符号表格 -->
          <div ref="tableWrapperRef" class="table-wrapper">
            <el-table-v2
              :columns="[
                { key: 'symbolName', dataKey: 'symbolName', title: '符号名', width: 250 },
                { key: 'versionInA', dataKey: 'versionInA', title: 'OS A 版本', width: 200 },
                { key: 'versionInB', dataKey: 'versionInB', title: 'OS B 版本', width: 200 },
                { key: 'status', dataKey: 'status', title: '状态', width: 150 }
              ]"
              :data="filteredSymbols"
              :width="tableWidth"
              :height="tableHeight"
              :row-height="40"
              class="symbol-table"
            >
            <template #cell="{ column, rowData }">
              <template v-if="column.key === 'symbolName'">
                {{ rowData.symbolName || rowData.SymbolName }}
              </template>
              <template v-else-if="column.key === 'versionInA'">
                {{ rowData.versionInA || rowData.SymbolVersion || '-' }}
              </template>
              <template v-else-if="column.key === 'versionInB'">
                {{ rowData.versionInB || '-' }}
              </template>
              <template v-else-if="column.key === 'status'">
                <el-tag
                  v-if="rowData.status === '版本降级'"
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
            </template>
            </el-table-v2>
          </div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<style scoped>
.userspace-page {
  margin-top: 10px;
  height: 100%;
  box-sizing: border-box;
}

.userspace-page > .el-row {
  height: 100%;
  min-height: 0;
}

.userspace-page > .el-row > .el-col {
  height: 100%;
  min-height: 0;
}

.tree-card,
.el-col:nth-child(2) > .el-card {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.tree-card :deep(.el-card__body),
.el-col:nth-child(2) > .el-card :deep(.el-card__body) {
  flex: 1;
  overflow: auto;
  display: flex;
  flex-direction: column;
}

.tree-card :deep(.el-tree) {
  flex: 1;
  overflow: auto;
}

.table-wrapper {
  flex: 1;
  min-width: 0;
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

.tree-node {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
}

.tree-badge {
  margin-left: 10px;
}
</style>
