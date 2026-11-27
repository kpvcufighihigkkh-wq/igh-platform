<template>
  <div>
    <!-- 计数牌：与生产看板同一套读数样式，横向排列、数字用窄体 -->
    <section class="kpi-row">
      <div class="kpi">
        <span class="kpi-label">今日落纱</span>
        <span class="kpi-value">{{ stats.doffCount }}</span>
      </div>
      <div class="kpi">
        <span class="kpi-label">今日丝饼</span>
        <span class="kpi-value">{{ stats.bobbinCount }}</span>
      </div>
      <div class="kpi">
        <span class="kpi-label">活跃告警</span>
        <span class="kpi-value" :class="{ alarm: stats.alarmCount > 0 }">{{ stats.alarmCount }}</span>
      </div>
      <div class="kpi">
        <span class="kpi-label">仓库占用率</span>
        <span class="kpi-value">{{ stats.whUsage }}%</span>
      </div>
    </section>

    <n-card size="small" class="alarm-card">
      <template #header><span class="panel-title">最近告警</span></template>
      <n-data-table :columns="alarmCols" :data="alarms" :bordered="false" size="small" :max-height="200" />
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, h } from 'vue'
import { NTag } from 'naive-ui'
import http from '../../api/http'
import { tStatus, tLevel, formatTime } from '../../utils/status'

const stats = ref({ doffCount: 0, bobbinCount: 0, alarmCount: 0, whUsage: 0 })
const alarms = ref<any[]>([])

const alarmCols = [
  { title: '时间', key: 'created_at', width: 160, render: (r: any) => formatTime(r.created_at) },
  { title: '级别', key: 'level', width: 60, render: (r: any) => h(NTag, { type: r.level === 'urgent' ? 'error' : 'warning', size: 'small' }, () => tLevel(r.level)) },
  { title: '信息', key: 'message' },
  { title: '状态', key: 'status', width: 80, render: (r: any) => tStatus(r.status) },
]

onMounted(async () => {
  const [alarmRes, countRes, outputRes, whRes]: any[] = await Promise.all([
    http.get('/alarms?page=1&page_size=5'),
    http.get('/alarms/active-count'),
    http.get('/stats/daily-output?days=1'),
    http.get('/warehouses'),
  ])
  alarms.value = alarmRes.data?.items || []
  stats.value.alarmCount = countRes.data?.count || 0
  // days=1 是"最近 24 小时"，跨零点时数组首条可能是昨天。
  // 按日期精确匹配今天，取不到就保持 0，避免把昨天的产量算进"今日"。
  const todayStr = new Date().toLocaleDateString('sv-SE')
  const today = (outputRes.data || []).find((d: any) => (d.date || '').slice(0, 10) === todayStr)
  if (today) { stats.value.doffCount = today.doff_count; stats.value.bobbinCount = today.bobbin_count }
  if (whRes.data?.length) {
    const wh = whRes.data[0]
    const sRes: any = await http.get(`/warehouses/${wh.id}/stats`)
    const s = sRes.data || {}
    const total = (s.empty || 0) + (s.occupied || 0)
    stats.value.whUsage = total > 0 ? Math.round((s.occupied / total) * 100) : 0
  }
})
</script>

<style scoped>
.kpi-row {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  background: var(--plate);
  border: 1px solid var(--line);
  border-radius: 2px;
}
.kpi {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 16px 22px;
  position: relative;
}
.kpi + .kpi::before {
  content: '';
  position: absolute;
  left: 0;
  top: 16px;
  bottom: 16px;
  width: 1px;
  background: var(--line-soft);
}
.kpi-label { font-size: 13px; color: var(--steel); }
.kpi-value {
  font-family: var(--font-display);
  font-size: 40px;
  font-weight: 600;
  line-height: 1;
  font-variant-numeric: tabular-nums;
  color: var(--graphite);
}
.kpi-value.alarm { color: var(--alarm); }
.alarm-card { margin-top: 16px; }
.panel-title {
  font-family: var(--font-display);
  font-size: 15px;
  font-weight: 600;
  color: var(--steel);
}
@media (max-width: 900px) {
  .kpi-row { grid-template-columns: repeat(2, 1fr); }
}
</style>
