<script setup>
import { ref, onMounted, computed } from 'vue'
import OverviewPanel from './components/OverviewPanel.vue'
import KernelDetail from './components/KernelDetail.vue'
import UserspaceDetail from './components/UserspaceDetail.vue'
import RPMDetail from './components/RPMDetail.vue'

// 从 DOM 中读取 JSON 数据
const diffData = ref(null)
const osA = ref(null)
const osB = ref(null)
const loading = ref(true)
const error = ref(null)
const activeTab = ref('overview')

onMounted(() => {
  try {
    // 尝试从多个可能的位置获取数据
    let dataElement = document.getElementById('data')
    if (!dataElement) {
      const scripts = document.querySelectorAll('script[type="application/json"]')
      for (const script of scripts) {
        if (script.textContent.trim()) {
          dataElement = script
          break
        }
      }
    }

    if (dataElement && dataElement.textContent) {
      const rawData = JSON.parse(dataElement.textContent)
      diffData.value = rawData.diffResult || rawData
      osA.value = rawData.OS_A || {}
      osB.value = rawData.OS_B || {}
    } else if (window.__INITIAL_STATE__) {
      diffData.value = window.__INITIAL_STATE__.diffResult || window.__INITIAL_STATE__
      osA.value = window.__INITIAL_STATE__.OS_A || {}
      osB.value = window.__INITIAL_STATE__.OS_B || {}
    } else {
      error.value = '未找到差异数据'
    }
  } catch (e) {
    error.value = '解析数据失败: ' + e.message
    console.error(e)
  } finally {
    loading.value = false
  }
})


</script>

<template>
  <div v-if="loading">兼容性报告加载中...</div>
  <div v-else-if="error" class="error">{{ error }}</div>
  <div v-else>
    <!-- 顶部导航 -->
    <el-menu mode="horizontal" :default-active="activeTab" @select="(key) => activeTab = key">
      <el-menu-item index="overview">全局概览</el-menu-item>
      <el-menu-item index="kernel">Kernel 兼容性</el-menu-item>
      <el-menu-item index="userspace">Userspace 动态库</el-menu-item>
      <el-menu-item index="rpm">RPM 软件包</el-menu-item>
    </el-menu>

    <!-- 全局概览面板 -->
    <div v-show="activeTab === 'overview'" class="detail-page">
      <OverviewPanel :diff-data="diffData" :os-a="osA" :os-b="osB" />
    </div>

    <!-- Kernel 兼容性详情页 -->
    <div v-show="activeTab === 'kernel'" class="detail-page">
      <KernelDetail />
    </div>

    <!-- Userspace 动态库详情页 -->
    <div v-show="activeTab === 'userspace'" class="detail-page">
      <UserspaceDetail />
    </div>

    <!-- RPM 软件包详情页 -->
    <div v-show="activeTab === 'rpm'" class="detail-page">
      <RPMDetail />
    </div>
  </div>
</template>

<style>
#app {
  max-width: 1400px;
  margin: 0 auto;
}

.error {
  color: red;
  padding: 20px;
}

.detail-page {
  height: calc(100vh - 120px);
  min-height: 500px;
}
</style>
