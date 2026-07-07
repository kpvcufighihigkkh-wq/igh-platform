<template>
  <n-layout has-sider style="height: 100vh">
    <n-layout-sider
      bordered
      collapse-mode="width"
      :collapsed-width="64"
      :width="220"
      :collapsed="collapsed"
      show-trigger
      @collapse="collapsed = true"
      @expand="collapsed = false"
    >
      <div class="logo">IGH</div>
      <n-menu :collapsed="collapsed" :options="menuOptions" :value="activeKey" @update:value="handleMenuSelect" />
    </n-layout-sider>
    <n-layout>
      <n-layout-header bordered style="height: 56px; display: flex; align-items: center; padding: 0 24px; justify-content: space-between">
        <span style="font-size: 18px">{{ currentTitle }}</span>
        <n-button text @click="handleLogout">退出</n-button>
      </n-layout-header>
      <n-layout-content style="padding: 16px; overflow: auto; height: calc(100vh - 56px)">
        <router-view />
      </n-layout-content>
    </n-layout>
  </n-layout>
</template>

<script setup lang="ts">
import { ref, computed, h } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { NIcon } from 'naive-ui'
import { GridOutline, BarChartOutline, CubeOutline, HardwareChipOutline, AlertCircleOutline, CartOutline, BuildOutline, StatsChartOutline } from '@vicons/ionicons5'
import type { MenuOption } from 'naive-ui'

const router = useRouter()
const route = useRoute()
const auth = useAuthStore()
const collapsed = ref(false)

const activeKey = computed(() => route.name as string)
const currentTitle = computed(() => {
  const item = menuOptions.find(m => m.key === activeKey.value)
  return (item?.label as string) || 'IGH 智能制造平台'
})

function renderIcon(icon: any) {
  return () => h(NIcon, null, { default: () => h(icon) })
}

const menuOptions: MenuOption[] = [
  { label: '生产大屏', key: 'Dashboard', icon: renderIcon(GridOutline) },
  { label: '生产管理', key: 'Production', icon: renderIcon(BarChartOutline) },
  { label: '质量管理', key: 'Quality', icon: renderIcon(BarChartOutline) },
  { label: '订单管理', key: 'Orders', icon: renderIcon(CartOutline) },
  { label: '仓库管理', key: 'Warehouse', icon: renderIcon(CubeOutline) },
  { label: '设备管理', key: 'Devices', icon: renderIcon(HardwareChipOutline) },
  { label: '维保管理', key: 'Maintenance', icon: renderIcon(BuildOutline) },
  { label: '异常管理', key: 'Alarms', icon: renderIcon(AlertCircleOutline) },
  { label: '统计分析', key: 'Statistics', icon: renderIcon(StatsChartOutline) },
]

function handleMenuSelect(key: string) {
  router.push({ name: key })
}

function handleLogout() {
  auth.logout()
  router.push('/login')
}
</script>

<style scoped>
.logo {
  height: 56px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 24px;
  font-weight: bold;
  color: var(--color-primary);
}
</style>
