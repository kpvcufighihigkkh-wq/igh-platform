<template>
  <div class="dashboard">
    <n-grid :cols="4" :x-gap="16" :y-gap="16">
      <n-gi>
        <n-card class="stat-card">
          <div class="stat-label">今日落纱</div>
          <div class="stat-value">{{ stats.doffings }}</div>
        </n-card>
      </n-gi>
      <n-gi>
        <n-card class="stat-card">
          <div class="stat-label">今日丝饼</div>
          <div class="stat-value">{{ stats.bobbins }}</div>
        </n-card>
      </n-gi>
      <n-gi>
        <n-card class="stat-card">
          <div class="stat-label">活跃异常</div>
          <div class="stat-value" :class="{ danger: stats.alarms > 0 }">{{ stats.alarms }}</div>
        </n-card>
      </n-gi>
      <n-gi>
        <n-card class="stat-card">
          <div class="stat-label">仓库利用率</div>
          <div class="stat-value">{{ stats.warehouseUsage }}%</div>
        </n-card>
      </n-gi>
    </n-grid>

    <n-grid :cols="2" :x-gap="16" :y-gap="16" style="margin-top: 16px">
      <n-gi>
        <n-card title="设备状态">
          <DevicePanel />
        </n-card>
      </n-gi>
      <n-gi>
        <n-card title="仓库概览">
          <WarehouseGrid
            v-if="warehouse"
            :rows="warehouse.rows"
            :cols="warehouse.cols"
            :layers="warehouse.layers"
            :slots="warehouseSlots"
          />
          <n-empty v-else description="暂无仓库数据" />
        </n-card>
      </n-gi>
    </n-grid>

    <n-card title="实时事件" style="margin-top: 16px">
      <div class="event-list">
        <div v-for="(evt, i) in recentEvents" :key="i" class="event-item">
          <span class="event-time">{{ formatTime(evt.timestamp) }}</span>
          <n-tag :type="eventTagType(evt.topic)" size="small">{{ evt.topic }}</n-tag>
          <span class="event-detail">{{ eventSummary(evt) }}</span>
        </div>
        <n-empty v-if="recentEvents.length === 0" description="等待事件..." />
      </div>
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import http from '../../api/http'
import { useWebSocket } from '../../composables/useWebSocket'
import DevicePanel from '../../components/device/DevicePanel.vue'
import WarehouseGrid from '../../components/warehouse/WarehouseGrid.vue'

const stats = ref({ doffings: 0, bobbins: 0, alarms: 0, warehouseUsage: 0 })
const warehouse = ref<any>(null)
const warehouseSlots = ref<any[]>([])
const recentEvents = ref<any[]>([])

const { lastMessage } = useWebSocket(['production', 'alarm', 'device', 'warehouse'])

watch(lastMessage, (msg) => {
  if (!msg) return
  recentEvents.value.unshift(msg)
  if (recentEvents.value.length > 20) recentEvents.value.pop()

  if (msg.topic?.startsWith('production.doffing')) {
    stats.value.doffings++
    stats.value.bobbins += msg.data?.bobbin_count || 0
  }
})

onMounted(async () => {
  try {
    const whRes: any = await http.get('/warehouses')
    if (whRes.data?.length > 0) {
      warehouse.value = whRes.data[0]
      const slotsRes: any = await http.get(`/warehouses/${warehouse.value.id}/slots`)
      warehouseSlots.value = slotsRes.data || []
      const statsRes: any = await http.get(`/warehouses/${warehouse.value.id}/stats`)
      const s = statsRes.data || {}
      const total = (s.empty || 0) + (s.occupied || 0) + (s.reserved || 0)
      stats.value.warehouseUsage = total > 0 ? Math.round(((s.occupied || 0) / total) * 100) : 0
    }
  } catch { /* initial load may fail without data */ }
})

function formatTime(ts: string) {
  return new Date(ts).toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit', second: '2-digit' })
}

function eventTagType(topic: string) {
  if (topic?.includes('alarm')) return 'error'
  if (topic?.includes('production')) return 'success'
  if (topic?.includes('warehouse')) return 'info'
  return 'default'
}

function eventSummary(evt: any) {
  const d = evt.data
  if (!d) return ''
  if (d.line_code) return `${d.line_code} ${d.winder_name || ''} ×${d.bobbin_count || ''}`
  if (d.edge_id) return d.edge_id
  return JSON.stringify(d).slice(0, 60)
}
</script>

<style scoped>
.dashboard { height: 100%; overflow-y: auto; }
.stat-card { text-align: center; }
.stat-label { font-size: 14px; color: var(--text-secondary); margin-bottom: 4px; }
.stat-value { font-size: 32px; font-weight: bold; color: var(--color-primary); }
.stat-value.danger { color: var(--color-danger); }
.event-list { max-height: 240px; overflow-y: auto; }
.event-item { display: flex; align-items: center; gap: 8px; padding: 4px 0; border-bottom: 1px solid rgba(255,255,255,0.05); }
.event-time { font-size: 12px; color: var(--text-secondary); font-family: monospace; min-width: 70px; }
.event-detail { font-size: 13px; color: var(--text-primary); }
</style>
