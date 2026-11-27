<template>
  <n-layout has-sider style="height: 100vh">
    <n-layout-sider bordered collapse-mode="width" :collapsed-width="64" :width="220" :collapsed="collapsed" show-trigger @collapse="collapsed = true" @expand="collapsed = false">
      <!-- 品牌区：白色铭牌承载 IGH 标识，折叠时退化为红字缩写 -->
      <div class="brand" :class="{ mini: collapsed }">
        <img v-if="!collapsed" class="brand-logo" src="/igh-logo.jpg" alt="IGH S.r.l." />
        <span v-else class="brand-mini">IGH</span>
      </div>
      <n-menu :collapsed="collapsed" :options="menuOptions" :value="activeKey" @update:value="(k: string) => router.push({ name: k })" />
    </n-layout-sider>
    <n-layout>
      <n-layout-header bordered style="height:52px;display:flex;align-items:center;padding:0 20px;justify-content:space-between">
        <div class="page-title">
          <span class="title-bar" aria-hidden="true"></span>
          <span>{{ currentTitle }}</span>
        </div>
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
import { GridOutline, BarChartOutline, CubeOutline, HardwareChipOutline, AlertCircleOutline, CartOutline, BuildOutline, StatsChartOutline, SettingsOutline, SyncOutline, CarOutline, OptionsOutline, ConstructOutline, SearchOutline } from '@vicons/ionicons5'
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
  AdminERP: 'ERP对接', AdminVehicles: '载具管理', AdminLineConfig: '产线配置', AdminSiteConfig: '现场配置',
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
  { label: '现场配置', key: 'AdminSiteConfig', icon: icon(ConstructOutline) },
  { label: '异常管理', key: 'AdminAlarms', icon: icon(AlertCircleOutline) },
  // 追溯不挂模块开关：查货的来历不该因为某个模块被关掉而查不了
  { label: '全链追溯', key: 'AdminTrace', icon: icon(SearchOutline) },
  { label: '统计分析', key: 'AdminStatistics', icon: icon(StatsChartOutline) },
  { label: '系统设置', key: 'AdminSettings', icon: icon(SettingsOutline) },
])

function handleLogout() { auth.logout(); router.push('/login') }

onMounted(async () => {
  try { modules.value = ((await http.get('/admin/modules')) as any).data || {} } catch {}
})
</script>

<style scoped>
/* 白色铭牌，与登录页、工位页使用同一品牌呈现方式 */
.brand {
  margin: 14px 12px 10px;
  padding: 12px 14px;
  background: var(--plate);
  border-radius: 2px;
  display: flex;
  align-items: center;
  justify-content: center;
}
.brand.mini {
  margin: 14px 10px 10px;
  padding: 10px 4px;
}
.brand-logo {
  width: 100%;
  height: auto;
  display: block;
}
.brand-mini {
  font-family: var(--font-display);
  font-size: 18px;
  font-weight: 700;
  letter-spacing: 0.06em;
  color: var(--igh-red);
}
.page-title {
  display: flex;
  align-items: center;
  gap: 10px;
  font-family: var(--font-display);
  font-size: 18px;
  font-weight: 600;
  letter-spacing: 0.02em;
}
.title-bar {
  width: 3px;
  height: 18px;
  background: var(--igh-red);
}
</style>
