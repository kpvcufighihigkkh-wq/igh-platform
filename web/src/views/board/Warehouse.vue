<template>
  <div class="board">
    <header class="board-header"><h1>仓库实时状态</h1><span class="time">{{ clock }}</span></header>
    <div class="wh-tabs">
      <button v-for="w in warehouses" :key="w.id" class="wh-tab" :class="{active: selectedId === w.id}" @click="selectWarehouse(w)">{{ w.name }}</button>
    </div>
    <template v-if="warehouse">
      <div class="wh-info">
        <span>{{ warehouse.name }} ({{ warehouse.rows }}行 × {{ warehouse.cols }}列 × {{ warehouse.layers }}层)</span>
        <span class="stat">空闲: <b style="color:#63e2b7">{{ stats.empty || 0 }}</b> | 占用: <b style="color:#e88080">{{ stats.occupied || 0 }}</b> | 预留: <b style="color:#f2c97d">{{ stats.reserved || 0 }}</b> | 占用率: <b>{{ usage }}%</b></span>
      </div>
      <WarehouseGrid
        :rows="warehouse.rows"
        :cols="warehouse.cols"
        :layers="warehouse.layers"
        :slots="slots"
      />
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue'
import http from '../../api/http'
import WarehouseGrid from '../../components/warehouse/WarehouseGrid.vue'

const clock = ref('')
const warehouses = ref<any[]>([])
const warehouse = ref<any>(null)
const selectedId = ref<number | null>(null)
const slots = ref<any[]>([])
const stats = ref<Record<string, number>>({ empty: 0, occupied: 0 })
let timer: any

const usage = computed(() => {
  const t = (stats.value.empty || 0) + (stats.value.occupied || 0) + (stats.value.reserved || 0)
  return t > 0 ? Math.round(((stats.value.occupied || 0) / t) * 100) : 0
})

async function selectWarehouse(w: any) {
  warehouse.value = w
  selectedId.value = w.id
  await loadSlots(w.id)
}

async function loadSlots(id: number) {
  const [slotsRes, statsRes]: any[] = await Promise.all([
    http.get(`/warehouses/${id}/slots`),
    http.get(`/warehouses/${id}/stats`),
  ])
  slots.value = slotsRes.data || []
  stats.value = statsRes.data || { empty: 0, occupied: 0 }
}

async function load() {
  const whRes: any = await http.get('/warehouses')
  warehouses.value = whRes.data || []
  if (warehouses.value.length && !selectedId.value) {
    await selectWarehouse(warehouses.value[0])
  } else if (selectedId.value) {
    await loadSlots(selectedId.value)
  }
}

onMounted(() => {
  timer = setInterval(() => { clock.value = new Date().toLocaleTimeString('zh-CN', { hour12: false }) }, 1000)
  clock.value = new Date().toLocaleTimeString('zh-CN', { hour12: false })
  load()
})
onUnmounted(() => clearInterval(timer))
</script>

<style scoped>
.board { height:100vh; background:#0d1117; padding:16px; overflow:auto; }
.board-header { display:flex; justify-content:space-between; align-items:center; margin-bottom:12px; }
.board-header h1 { font-size:20px; color:#00d4ff; margin:0; }
.time { font-size:16px; color:#8b949e; font-family:monospace; }
.wh-tabs { display:flex; gap:8px; margin-bottom:12px; }
.wh-tab { background:#161b22; border:1px solid #30363d; color:#c9d1d9; padding:6px 16px; border-radius:6px; cursor:pointer; font-size:13px; transition:all .2s; }
.wh-tab:hover { border-color:#58a6ff; }
.wh-tab.active { background:#1f6feb; border-color:#1f6feb; color:#fff; }
.wh-info { display:flex; justify-content:space-between; margin-bottom:12px; font-size:14px; color:#c9d1d9; }
.stat b { font-size:16px; }
</style>
