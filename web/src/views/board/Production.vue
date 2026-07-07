<template>
  <div class="board">
    <header class="board-header"><h1>产线实时状态</h1><span class="time">{{ clock }}</span></header>
    <div class="lines-grid">
      <div class="line-card" v-for="line in lines" :key="line.code">
        <div class="line-title">{{ line.name }} <n-tag :type="line.status==='running'?'success':'warning'" size="small">{{ line.status==='running'?'运行中':'停机' }}</n-tag></div>
        <div class="line-stats">
          <div><span class="num">{{ line.doffCount }}</span><span class="lbl">落纱</span></div>
          <div><span class="num">{{ line.bobbinCount }}</span><span class="lbl">丝饼</span></div>
        </div>
      </div>
    </div>
    <div class="alarm-scroll" v-if="alarms.length">
      <div class="alarm-title">实时告警</div>
      <div class="alarm-item" v-for="a in alarms" :key="a.id">
        <n-tag :type="a.level==='urgent'?'error':'warning'" size="small">{{ a.level==='urgent'?'紧急':'警告' }}</n-tag>
        <span>{{ a.message }}</span>
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
.board { height:100vh; background:#0d1117; padding:16px; }
.board-header { display:flex; justify-content:space-between; align-items:center; margin-bottom:16px; }
.board-header h1 { font-size:20px; color:#00d4ff; margin:0; }
.time { font-size:16px; color:#8b949e; font-family:monospace; }
.lines-grid { display:grid; grid-template-columns:repeat(2,1fr); gap:12px; margin-bottom:16px; }
.line-card { background:#161b22; border:1px solid #30363d; border-radius:8px; padding:16px; }
.line-title { font-size:15px; font-weight:bold; margin-bottom:12px; display:flex; align-items:center; gap:8px; }
.line-stats { display:flex; gap:24px; }
.line-stats .num { font-size:28px; font-weight:bold; color:#63e2b7; }
.line-stats .lbl { font-size:12px; color:#8b949e; margin-left:4px; }
.alarm-scroll { background:#161b22; border:1px solid #30363d; border-radius:8px; padding:12px; max-height:200px; overflow:auto; }
.alarm-title { font-size:14px; font-weight:bold; color:#e88080; margin-bottom:8px; }
.alarm-item { display:flex; align-items:center; gap:8px; padding:4px 0; border-bottom:1px solid #21262d; font-size:13px; }
.alarm-time { margin-left:auto; color:#8b949e; font-size:11px; }
</style>
