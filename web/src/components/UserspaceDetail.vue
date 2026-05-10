<script setup>
import { ref, computed, onMounted } from 'vue'

const diffData = ref(null)
const loading = ref(true)
const selectedNode = ref(null)
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

// 根据筛选类型获取符号
const filteredSymbols = computed(() => {
  const data = currentSymbolData.value
  switch (filterType.value) {
    case 'onlyInA':
      return (data.onlyInA || []).map(s => ({ ...s, status: '仅A有' }))
    case 'onlyInB':
      return (data.onlyInB || []).map(s => ({ ...s, status: '仅B有' }))
    case 'modified':
      return (data.modified || []).map(s => ({
        symbolName: s.symbolName,
        versionInA: s.versionInA,
        versionInB: s.versionInB,
        status: isVersionDowngrade(s.versionInA, s.versionInB) ? '版本降级' : '版本变化'
      }))
    default:
      return []
  }
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

// 筛选选项
const filterOptions = [
  { label: '版本变化', value: 'modified' },
  { label: '仅 A 有', value: 'onlyInA' },
  { label: '仅 B 有', value: 'onlyInB' },
]
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
          </div>

          <!-- 符号表格 -->
          <el-table-v2
            :columns="[
              { key: 'symbolName', label: '符号名', width: 250 },
              { key: 'versionInA', label: 'OS A 版本', width: 200 },
              { key: 'versionInB', label: 'OS B 版本', width: 200 },
              { key: 'status', label: '状态', width: 150 }
            ]"
            :data="filteredSymbols"
            :width="800"
            :height="500"
            :row-height="40"
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
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<style scoped>
.userspace-page {
  padding: 20px;
}

.tree-card {
  height: 600px;
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

.filter-bar {
  margin-bottom: 15px;
  padding: 10px;
  background: #f5f7fa;
  border-radius: 4px;
}
</style>
