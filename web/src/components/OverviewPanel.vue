<script setup>
import { computed, onMounted, ref } from 'vue'
import * as echarts from 'echarts'

const props = defineProps({
  diffData: Object,
  osA: Object,
  osB: Object
})

const stats = computed(() => {
  if (!props.diffData) return {}
  const d = props.diffData
  return {
    kernelCrcConflict: d.kernelSymbolsDiff?.modified?.length || 0,
    userspaceApiMissing: calculateUserspaceMissing(d),
    rpmUpgrade: countRPMUpgrade(d),
    rpmDowngrade: countRPMPackageDowngrade(d),
    syscallOnlyInA: d.syscallsDiff?.onlyInA?.length || 0,
    syscallOnlyInB: d.syscallsDiff?.onlyInB?.length || 0,
  }
})

function calculateUserspaceMissing(d) {
  if (!d.userspaceSymbolsDiff?.bySoPath) return 0
  let count = 0
  for (const path in d.userspaceSymbolsDiff.bySoPath) {
    const group = d.userspaceSymbolsDiff.bySoPath[path]
    count += (group.onlyInA?.length || 0) + (group.onlyInB?.length || 0)
  }
  return count
}

function countRPMUpgrade(d) {
  if (!d.rpmPackagesDiff?.modified) return 0
  return d.rpmPackagesDiff.modified.filter(p => p.upgrade).length
}

function countRPMPackageDowngrade(d) {
  if (!d.rpmPackagesDiff?.modified) return 0
  return d.rpmPackagesDiff.modified.filter(p => !p.upgrade).length
}

const radarData = computed(() => {
  if (!props.diffData) return null
  const d = props.diffData
  const totalA = d.syscallsDiff?.totalInA || 1
  const totalB = d.syscallsDiff?.totalInB || 1
  return {
    indicator: [
      { name: 'Syscall 差异度', max: 100 },
      { name: '内核CRC冲突度', max: 100 },
      { name: '用户态API差异度', max: 100 },
      { name: 'RPM包差异度', max: 100 },
    ],
    values: [
      Math.min(100, Math.round(((d.syscallsDiff?.onlyInA?.length || 0) + (d.syscallsDiff?.onlyInB?.length || 0)) / Math.max(totalA, totalB) * 100)),
      Math.min(100, Math.round((d.kernelSymbolsDiff?.modified?.length || 0) / Math.max(d.kernelSymbolsDiff?.totalInA || 1, d.kernelSymbolsDiff?.totalInB || 1) * 1000)),
      Math.min(100, Math.round(calculateUserspaceMissing(d) / 50)),
      Math.min(100, Math.round(((d.rpmPackagesDiff?.onlyInA?.length || 0) + (d.rpmPackagesDiff?.onlyInB?.length || 0)) / 20)),
    ]
  }
})

function initCharts() {
  if (!radarData.value) return
  const chartDom = document.getElementById('radarChart')
  if (!chartDom) return

  const myChart = echarts.init(chartDom)
  const option = {
    title: { text: '兼容性评分', left: 'center' },
    tooltip: {},
    radar: { indicator: radarData.value.indicator, radius: '60%' },
    series: [{
      name: '兼容性评分',
      type: 'radar',
      data: [{ value: radarData.value.values, name: 'OS B 相对 OS A' }]
    }]
  }
  myChart.setOption(option)
  window.addEventListener('resize', () => myChart.resize())
}

function formatDate(dateStr) {
  if (!dateStr) return 'N/A'
  try {
    return new Date(dateStr).toLocaleString('zh-CN')
  } catch {
    return dateStr
  }
}

onMounted(() => {
  if (props.diffData) {
    setTimeout(initCharts, 100)
  }
})
</script>

<template>
  <div class="overview">
    <el-row :gutter="20" class="metadata-row">
      <el-col :span="12">
        <el-card>
          <template #header><span>OS A</span></template>
          <p><strong>名称:</strong> {{ osA?.metadata?.name || 'N/A' }}</p>
          <p><strong>版本:</strong> {{ osA?.metadata?.version || 'N/A' }}</p>
          <p><strong>架构:</strong> {{ osA?.metadata?.architecture || 'N/A' }}</p>
          <p><strong>采集时间:</strong> {{ formatDate(osA?.metadata?.collectedAt) }}</p>
        </el-card>
      </el-col>
      <el-col :span="12">
        <el-card>
          <template #header><span>OS B</span></template>
          <p><strong>名称:</strong> {{ osB?.metadata?.name || 'N/A' }}</p>
          <p><strong>版本:</strong> {{ osB?.metadata?.version || 'N/A' }}</p>
          <p><strong>架构:</strong> {{ osB?.metadata?.architecture || 'N/A' }}</p>
          <p><strong>采集时间:</strong> {{ formatDate(osB?.metadata?.collectedAt) }}</p>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="20" class="stats-row">
      <el-col :span="6">
        <el-card class="stats-card conflict">
          <div class="stats-number">{{ stats.kernelCrcConflict }}</div>
          <div class="stats-label">内核 CRC 冲突数</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card class="stats-card warning">
          <div class="stats-number">{{ stats.userspaceApiMissing }}</div>
          <div class="stats-label">用户态 API 缺失数</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card class="stats-card info">
          <div class="stats-number">{{ stats.rpmDowngrade }}</div>
          <div class="stats-label">RPM 包降级数</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card class="stats-card success">
          <div class="stats-number">{{ stats.rpmUpgrade }}</div>
          <div class="stats-label">RPM 包升级数</div>
        </el-card>
      </el-col>
    </el-row>

    <el-row>
      <el-col :span="24">
        <el-card>
          <div id="radarChart" style="width: 100%; height: 400px;"></div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<style scoped>
.overview {
  padding: 20px;
}

.metadata-row,
.stats-row {
  margin-bottom: 20px;
}

.stats-card {
  text-align: center;
}

.stats-card.conflict {
  border-top: 3px solid #f56c6c;
}

.stats-card.warning {
  border-top: 3px solid #e6a23c;
}

.stats-card.info {
  border-top: 3px solid #409eff;
}

.stats-card.success {
  border-top: 3px solid #67c23a;
}

.stats-number {
  font-size: 36px;
  font-weight: bold;
  color: #333;
  margin: 10px 0;
}

.stats-label {
  font-size: 14px;
  color: #666;
}
</style>
