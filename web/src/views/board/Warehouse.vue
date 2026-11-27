<template>
  <div class="board">
    <header class="board-header">
      <div class="brand">
        <span class="title-bar" aria-hidden="true"></span>
        <h1>仓库实时状态</h1>
      </div>
      <span class="time">{{ clock }}</span>
    </header>

    <div class="wh-tabs">
      <button
        v-for="w in warehouses"
        :key="w.id"
        class="wh-tab"
        :class="{ active: selectedId === w.id }"
        @click="selectWarehouse(w)"
      >
        {{ w.name }}
      </button>
    </div>

    <template v-if="warehouse">
      <div class="wh-info">
        <span class="wh-spec">
          {{ warehouse.name }}（{{ warehouse.rows }} 行 × {{ warehouse.cols }} 列 × {{ warehouse.layers }} 层）
        </span>
        <span class="stat">
          <span class="stat-item"><span class="lbl">空闲</span><b class="ok">{{ stats.empty || 0 }}</b></span>
          <span class="stat-item"><span class="lbl">占用</span><b class="alarm">{{ stats.occupied || 0 }}</b></span>
          <span class="stat-item"><span class="lbl">预留</span><b class="warn">{{ stats.reserved || 0 }}</b></span>
          <span class="stat-item"><span class="lbl">占用率</span><b>{{ usage }}%</b></span>
        </span>
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
.board {
  height: 100vh;
  padding: 16px;
  background: var(--paper);
  overflow: auto;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.board-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-bottom: 12px;
  border-bottom: 2px solid var(--graphite);
  flex-shrink: 0;
}
.brand {
  display: flex;
  align-items: center;
  gap: 12px;
}
.title-bar {
  width: 4px;
  height: 22px;
  background: var(--igh-red);
}
.brand h1 {
  margin: 0;
  font-family: var(--font-display);
  font-size: 26px;
  font-weight: 600;
  color: var(--graphite);
}
.time {
  font-family: var(--font-mono);
  font-size: 22px;
  font-variant-numeric: tabular-nums;
  color: var(--graphite);
}

.wh-tabs {
  display: flex;
  gap: 8px;
  flex-shrink: 0;
}
.wh-tab {
  background: var(--plate);
  border: 1px solid var(--line);
  color: var(--steel);
  padding: 7px 18px;
  border-radius: 2px;
  cursor: pointer;
  font-size: 14px;
  transition: border-color 0.15s, color 0.15s;
}
.wh-tab:hover {
  border-color: var(--steel-light);
  color: var(--graphite);
}
.wh-tab.active {
  background: var(--graphite);
  border-color: var(--graphite);
  color: #fff;
}

.wh-info {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  padding: 10px 14px;
  background: var(--plate);
  border: 1px solid var(--line);
  border-radius: 2px;
  font-size: 14px;
  color: var(--graphite);
  flex-shrink: 0;
}
.wh-spec {
  font-family: var(--font-display);
  font-size: 16px;
  font-weight: 600;
}
.stat {
  display: flex;
  gap: 22px;
}
.stat-item {
  display: flex;
  align-items: baseline;
  gap: 6px;
}
.stat-item .lbl {
  font-size: 12px;
  color: var(--steel);
}
.stat b {
  font-family: var(--font-display);
  font-size: 20px;
  font-variant-numeric: tabular-nums;
}
.stat .ok { color: var(--ok); }
.stat .alarm { color: var(--alarm); }
.stat .warn { color: var(--warn); }
</style>
