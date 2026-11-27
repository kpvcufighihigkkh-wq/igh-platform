<template>
  <div class="board">
    <header class="board-header">
      <div class="brand">
        <span class="title-bar" aria-hidden="true"></span>
        <h1>产线实时状态</h1>
      </div>
      <span class="time">{{ clock }}</span>
    </header>

    <div class="lines-grid">
      <div class="line-card" v-for="line in lines" :key="line.code">
        <div class="line-title">
          <span class="line-name">{{ line.name }}</span>
          <span class="line-code">{{ line.code }}</span>
          <n-tag :type="line.status === 'running' ? 'success' : 'warning'" size="small">
            {{ line.status === 'running' ? '运行中' : '停机' }}
          </n-tag>
        </div>
        <div class="line-stats">
          <div class="stat">
            <span class="num">{{ line.doffCount }}</span>
            <span class="lbl">落纱</span>
          </div>
          <div class="stat">
            <span class="num">{{ line.bobbinCount }}</span>
            <span class="lbl">丝饼</span>
          </div>
        </div>
      </div>
    </div>

    <div class="alarm-scroll" v-if="alarms.length">
      <div class="alarm-title">实时告警</div>
      <div class="alarm-item" v-for="a in alarms" :key="a.id">
        <n-tag :type="a.level === 'urgent' ? 'error' : 'warning'" size="small">
          {{ a.level === 'urgent' ? '紧急' : '警告' }}
        </n-tag>
        <span class="alarm-msg">{{ a.message }}</span>
        <span class="alarm-time">{{ new Date(a.created_at).toLocaleTimeString('zh-CN') }}</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import http from '../../api/http'

const clock = ref('')
const lines = ref<any[]>([])
const alarms = ref<any[]>([])
let timer: any

async function load() {
  const [lineRes, todayRes, alarmRes]: any[] = await Promise.all([
    http.get('/admin/production-lines'),
    http.get('/stats/today-by-line'),
    http.get('/alarms?page=1&page_size=10&status=active'),
  ])
  const todayData = todayRes.data || []
  const lineList = (lineRes.data || []).filter((l: any) => l.station_type === 'winding')
  lines.value = lineList.map((l: any) => {
    const s = todayData.find((d: any) => d.line_code === l.code)
    return { ...l, doffCount: s?.doff_count || 0, bobbinCount: s?.bobbin_count || 0 }
  })
  alarms.value = alarmRes.data?.items || []
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
  display: flex;
  flex-direction: column;
  gap: 12px;
  overflow: auto;
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

.lines-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 12px;
}
.line-card {
  background: var(--plate);
  border: 1px solid var(--line);
  border-radius: 2px;
  padding: 16px 18px;
}
.line-title {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 14px;
  padding-bottom: 10px;
  border-bottom: 1px solid var(--line-soft);
}
.line-name {
  font-family: var(--font-display);
  font-size: 17px;
  font-weight: 600;
}
.line-code {
  font-family: var(--font-mono);
  font-size: 12px;
  color: var(--steel);
}
.line-stats {
  display: flex;
  gap: 32px;
}
.stat {
  display: flex;
  align-items: baseline;
  gap: 6px;
}
.num {
  font-family: var(--font-display);
  font-size: 38px;
  font-weight: 600;
  line-height: 1;
  font-variant-numeric: tabular-nums;
  color: var(--graphite);
}
.lbl {
  font-size: 13px;
  color: var(--steel);
}

.alarm-scroll {
  background: var(--plate);
  border: 1px solid var(--line);
  border-left: 3px solid var(--alarm);
  border-radius: 2px;
  padding: 12px 16px;
  max-height: 220px;
  overflow: auto;
}
.alarm-title {
  font-family: var(--font-display);
  font-size: 15px;
  font-weight: 600;
  color: var(--alarm);
  margin-bottom: 8px;
}
.alarm-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 0;
  border-bottom: 1px solid var(--line-soft);
  font-size: 14px;
}
.alarm-item:last-child {
  border-bottom: none;
}
.alarm-msg {
  color: var(--graphite);
}
.alarm-time {
  margin-left: auto;
  font-family: var(--font-mono);
  color: var(--steel);
  font-size: 12px;
}

@media (max-width: 900px) {
  .lines-grid { grid-template-columns: 1fr; }
}
</style>
