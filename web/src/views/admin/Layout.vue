<template>
  <n-layout has-sider style="height: 100vh">
    <n-layout-sider bordered collapse-mode="width" :collapsed-width="64" :width="200" :collapsed="collapsed" show-trigger @collapse="collapsed = true" @expand="collapsed = false">
      <div class="logo">IGH</div>
      <n-menu :collapsed="collapsed" :options="menuOptions" :value="activeKey" @update:value="(k: string) => router.push({ name: k })" />
    </n-layout-sider>
    <n-layout>
      <n-layout-header bordered style="height:52px;display:flex;align-items:center;padding:0 20px;justify-content:space-between">
        <span style="font-size:16px;font-weight:bold">{{ currentTitle }}</span>
        <n-space>
          <n-button text size="small" @click="router.push('/board/overview')">看板</n-button>
          <n-button text size="small" @click="handleLogout">退出</n-button>
        </n-space>
      </n-layout-header>
      <n-layout-content style="padding:16px;overflow:auto;height:calc(100vh - 52px)">
        <ModuleAlert />
        <router-view />
      </n-layout-content>
    </n-layout>
  </n-layout>
</template>

<script setup lang="ts">
import { ref, computed, h, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '../../stores/auth'
import { NIcon } from 'naive-ui'
import ModuleAlert from '../../components/ModuleAlert.vue'
import http from '../../api/http'
import { GridOutline, BarChartOutline, CubeOutline, HardwareChipOutline, AlertCircleOutline, CartOutline, BuildOutline, StatsChartOutline, SettingsOutline, SyncOutline, CarOutline, OptionsOutline } from '@vicons/ionicons5'
import type { MenuOption } from 'naive-ui'

const router = useRouter()
const route = useRoute()
const auth = useAuthStore()
const collapsed = ref(false)
const modules = ref<any>({})

const activeKey = computed(() => route.name as string)
const titles: Record<string, string> = {
  AdminDashboard: '管理总览', AdminProduction: '生产管理', AdminQuality: '质量管理',
  AdminOrders: '订单管理', AdminWarehouse: '仓库管理', AdminMaintenance: '维保管理',
  AdminAlarms: '异常管理', AdminStatistics: '统计分析', AdminDevices: '设备管理', AdminSettings: '系统设置',
  AdminERP: 'ERP对接', AdminVehicles: '载具管理', AdminLineConfig: '产线配置',
}
const currentTitle = computed(() => titles[activeKey.value] || 'IGH 管理后台')

function icon(i: any) { return () => h(NIcon, null, { default: () => h(i) }) }

const menuOptions = computed<MenuOption[]>(() => [
  { label: '管理总览', key: 'AdminDashboard', icon: icon(GridOutline) },
  { label: '生产管理', key: 'AdminProduction', icon: icon(BarChartOutline), disabled: modules.value.Winding === false },
  { label: '质量管理', key: 'AdminQuality', icon: icon(BarChartOutline), disabled: modules.value.Quality === false },
  { label: '订单管理', key: 'AdminOrders', icon: icon(CartOutline), disabled: modules.value.Packaging === false },
  { label: '仓库管理', key: 'AdminWarehouse', icon: icon(CubeOutline), disabled: modules.value.Warehouse === false },
  { label: '维保管理', key: 'AdminMaintenance', icon: icon(BuildOutline), disabled: modules.value.Maintenance === false },
  { label: 'ERP对接', key: 'AdminERP', icon: icon(SyncOutline), disabled: modules.value.ERP === false },
  { label: '载具管理', key: 'AdminVehicles', icon: icon(CarOutline), disabled: modules.value.Vehicle === false },
  { label: '设备管理', key: 'AdminDevices', icon: icon(HardwareChipOutline) },
  { label: '产线配置', key: 'AdminLineConfig', icon: icon(OptionsOutline) },
  { label: '异常管理', key: 'AdminAlarms', icon: icon(AlertCircleOutline) },
  { label: '统计分析', key: 'AdminStatistics', icon: icon(StatsChartOutline) },
  { label: '系统设置', key: 'AdminSettings', icon: icon(SettingsOutline) },
])

function handleLogout() { auth.logout(); router.push('/login') }

onMounted(async () => {
  try { modules.value = ((await http.get('/admin/modules')) as any).data || {} } catch {}
})
</script>

<style scoped>
.logo { height:52px;display:flex;align-items:center;justify-content:center;font-size:22px;font-weight:bold;color:#00d4ff }
</style>
