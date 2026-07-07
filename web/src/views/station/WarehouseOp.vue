<template>
  <div class="station">
    <header class="station-header">
      <h1>仓库工位</h1>
      <div class="header-info">
        <span class="user" v-if="isLoggedIn()">{{ userName }} <n-button text size="tiny" @click="logout">退出</n-button></span>
        <span class="clock">{{ clock }}</span>
      </div>
    </header>
    <ModuleAlert />
    <n-grid :cols="3" :x-gap="12">
      <n-gi :span="2">
        <n-card title="库位状态" size="small">
          <template #header-extra>
            <n-select v-model:value="selectedId" :options="whOpts" size="small" style="width:160px" @update:value="onWhChange" />
          </template>
          <WarehouseGrid
            v-if="warehouse"
            :rows="warehouse.rows"
            :cols="warehouse.cols"
            :layers="warehouse.layers"
            :slots="slots"
          />
        </n-card>
      </n-gi>
      <n-gi>
        <n-card title="统计" size="small">
          <n-descriptions :column="1" size="small">
            <n-descriptions-item label="空闲">{{ wStats.empty || 0 }}</n-descriptions-item>
            <n-descriptions-item label="占用">{{ wStats.occupied || 0 }}</n-descriptions-item>
            <n-descriptions-item label="预留">{{ wStats.reserved || 0 }}</n-descriptions-item>
            <n-descriptions-item label="占用率">{{ usage }}%</n-descriptions-item>
          </n-descriptions>
        </n-card>
        <n-card title="操作" size="small" style="margin-top:12px">
          <n-space vertical>
            <n-button size="small" type="primary" block @click="requireAuth(() => {})">手动入库</n-button>
            <n-button size="small" block @click="requireAuth(() => {})">手动出库</n-button>
          </n-space>
        </n-card>
        <n-card title="待入库料车" size="small" style="margin-top:12px">
          <n-data-table :columns="trolleyCols" :data="trolleys" :bordered="false" size="small" :max-height="200" />
        </n-card>
      </n-gi>
    </n-grid>
    <LoginModal />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue'
import http from '../../api/http'
import { tStatus } from '../../utils/status'
import { useAuth } from '../../composables/useAuth'
import LoginModal from '../../components/LoginModal.vue'
import ModuleAlert from '../../components/ModuleAlert.vue'
import WarehouseGrid from '../../components/warehouse/WarehouseGrid.vue'

const { isLoggedIn, userName, requireAuth, logout } = useAuth()
const clock = ref('')
const warehouses = ref<any[]>([])
const warehouse = ref<any>(null)
const selectedId = ref<number | null>(null)
const slots = ref<any[]>([])
const wStats = ref<Record<string, number>>({ empty: 0, occupied: 0 })
const trolleys = ref<any[]>([])
let timer: any

const whOpts = computed(() => warehouses.value.map(w => ({ label: w.name, value: w.id })))
const usage = computed(() => { const t = (wStats.value.empty || 0) + (wStats.value.occupied || 0) + (wStats.value.reserved || 0); return t > 0 ? Math.round(((wStats.value.occupied || 0) / t) * 100) : 0 })

const trolleyCols = [
  { title: '编号', key: 'number', width: 60 },
  { title: '状态', key: 'status', width: 80, render: (r: any) => tStatus(r.status) },
]

async function onWhChange(id: number) {
  warehouse.value = warehouses.value.find(w => w.id === id)
  await loadSlots(id)
}

async function loadSlots(id: number) {
  const [sRes, stRes]: any[] = await Promise.all([
    http.get(`/warehouses/${id}/slots`),
    http.get(`/warehouses/${id}/stats`),
  ])
  slots.value = sRes.data || []
  wStats.value = stRes.data || { empty: 0, occupied: 0 }
}

async function load() {
  const [whRes, tRes]: any[] = await Promise.all([http.get('/warehouses'), http.get('/trolleys?page=1&page_size=10')])
  trolleys.value = tRes.data?.items || []
  warehouses.value = whRes.data || []
  if (warehouses.value.length && !selectedId.value) {
    selectedId.value = warehouses.value[0].id
    warehouse.value = warehouses.value[0]
    await loadSlots(warehouses.value[0].id)
  } else if (selectedId.value) {
    await loadSlots(selectedId.value)
  }
}

onMounted(() => {
  timer = setInterval(() => { clock.value = new Date().toLocaleTimeString('zh-CN', { hour12: false }); load() }, 20000)
  clock.value = new Date().toLocaleTimeString('zh-CN', { hour12: false })
  load()
})
onUnmounted(() => clearInterval(timer))
</script>

<style scoped>
.station { height:100vh; background:#0d1117; padding:16px; overflow:auto; }
.station-header { display:flex; justify-content:space-between; align-items:center; margin-bottom:12px; }
.station-header h1 { font-size:18px; color:#00d4ff; margin:0; }
.header-info { display:flex; align-items:center; gap:12px; }
.clock { font-family:monospace; color:#8b949e; }
.user { font-size:12px; color:#63e2b7; }
</style>
