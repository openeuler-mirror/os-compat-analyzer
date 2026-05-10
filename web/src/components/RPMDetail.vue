<script setup>
import { ref, computed, onMounted } from 'vue'

const diffData = ref(null)
const loading = ref(true)
const searchText = ref('')
const statusFilter = ref('')

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
  } catch (e) {
    console.error('加载数据失败:', e)
  } finally {
    loading.value = false
  }
}

// 合并所有 RPM 包差异数据
const allPackages = computed(() => {
  if (!diffData.value?.rpmPackagesDiff) return []

  const data = diffData.value.rpmPackagesDiff
  const result = []

  // 添加仅 A 有的包 (新增)
  for (const pkg of (data.onlyInA || [])) {
    result.push({
      name: pkg.name,
      arch: pkg.arch,
      versionA: `${pkg.version}-${pkg.release}`,
      versionB: '-',
      status: '删除',
      statusType: 'danger'
    })
  }

  // 添加仅 B 有的包 (新增)
  for (const pkg of (data.onlyInB || [])) {
    result.push({
      name: pkg.name,
      arch: pkg.arch,
      versionA: '-',
      versionB: `${pkg.version}-${pkg.release}`,
      status: '新增',
      statusType: 'success'
    })
  }

  // 添加版本变化的包
  for (const pkg of (data.modified || [])) {
    result.push({
      name: pkg.name,
      arch: pkg.arch,
      versionA: pkg.versionInA,
      versionB: pkg.versionInB,
      status: pkg.upgrade ? '升级' : '降级',
      statusType: pkg.upgrade ? 'primary' : 'warning'
    })
  }

  return result
})

// 筛选后的数据
const filteredPackages = computed(() => {
  let data = allPackages.value

  // 按包名搜索
  if (searchText.value) {
    const search = searchText.value.toLowerCase()
    data = data.filter(pkg => pkg.name.toLowerCase().includes(search))
  }

  // 按状态筛选
  if (statusFilter.value) {
    data = data.filter(pkg => pkg.status === statusFilter.value)
  }

  return data
})

// 表格列定义
const columns = [
  { key: 'name', label: '包名', sortable: true, width: 250 },
  { key: 'arch', label: '架构', width: 100 },
  { key: 'versionA', label: 'OS A 版本', sortable: true, width: 200 },
  { key: 'versionB', label: 'OS B 版本', sortable: true, width: 200 },
  { key: 'status', label: '状态', width: 100 },
]

// 状态筛选选项
const statusOptions = [
  { label: '全部', value: '' },
  { label: '新增', value: '新增' },
  { label: '删除', value: '删除' },
  { label: '升级', value: '升级' },
  { label: '降级', value: '降级' },
]

// 自定义排序函数
function sortMethod(a, b, col) {
  // 版本号排序：按 . 分割成数字数组，逐个比较
  const versionA = col === 'versionA' ? a.versionA : a.name
  const versionB = col === 'versionB' ? b.versionB : b.name

  const partsA = versionA.replace(/-.*$/, '').split('.').map(Number)
  const partsB = versionB.replace(/-.*$/, '').split('.').map(Number)

  const maxLen = Math.max(partsA.length, partsB.length)
  for (let i = 0; i < maxLen; i++) {
    const numA = partsA[i] || 0
    const numB = partsB[i] || 0
    if (numA !== numB) {
      return numA - numB
    }
  }
  return 0
}
</script>

<template>
  <div class="rpm-page">
    <el-card>
      <template #header>
        <div class="header-row">
          <span>RPM 软件包差异详情</span>
          <span class="stats">共 {{ filteredPackages.length }} 条差异</span>
        </div>
      </template>

      <!-- 搜索和筛选 -->
      <div class="filter-bar">
        <el-input
          v-model="searchText"
          placeholder="搜索包名..."
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

      <!-- 表格 -->
      <el-table
        :data="filteredPackages"
        height="550"
        :default-sort="{ prop: 'name', order: 'ascending' }"
      >
        <el-table-column prop="name" label="包名" width="250" sortable />
        <el-table-column prop="arch" label="架构" width="100" />
        <el-table-column
          prop="versionA"
          label="OS A 版本"
          width="200"
          :sort-method="(a, b) => sortMethod(a, b, 'versionA')"
          sortable
        />
        <el-table-column
          prop="versionB"
          label="OS B 版本"
          width="200"
          :sort-method="(a, b) => sortMethod(a, b, 'versionB')"
          sortable
        />
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.statusType" size="small">
              {{ row.status }}
            </el-tag>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<style scoped>
.rpm-page {
  padding: 20px;
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
</style>
