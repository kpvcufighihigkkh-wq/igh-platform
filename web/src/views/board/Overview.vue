<template>
  <div class="board">
    <header class="board-header">
      <h1>IGH 智能制造平台</h1>
      <div class="header-right">
        <span class="time">{{ clock }}</span>
        <n-button text size="small" @click="$router.push('/login')">管理登录</n-button>
      </div>
    </header>
    <div class="kpi-row">
      <div class="kpi-card" v-for="kpi in kpis" :key="kpi.label">
        <div class="kpi-value" :style="{color: kpi.color}">{{ kpi.value }}</div>
        <div class="kpi-label">{{ kpi.label }}</div>
      </div>
    </div>
    <div class="chart-row">
      <div class="chart-box"><div ref="outputRef" style="height:260px"></div></div>
      <div class="chart-box"><div ref="gradeRef" style="height:260px"></div></div>
    </div>
    <div class="chart-row">
      <div class="chart-box"><div ref="lineRef" style="height:240px"></div></div>
      <div class="chart-box"><div ref="alarmRef" style="height:240px"></div></div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import * as echarts from 'echarts'
import http from '../../api/http'

const clock = ref('')
const kpis = ref([
  { label: '今日落纱', value: 0, color: '#63e2b7' },
  { label: '今日丝饼', value: 0, color: '#70c0e8' },
  { label: '活跃告警', value: 0, color: '#e88080' },
  { label: '仓库占用率', value: '0%', color: '#f2c97d' },
])

const outputRef = ref<HTMLElement>()
const gradeRef = ref<HTMLElement>()
const lineRef = ref<HTMLElement>()
const alarmRef = ref<HTMLElement>()
let charts: echarts.ECharts[] = []
let timer: any

function initChart(el: HTMLElement) { const c = echarts.init(el, 'dark'); charts.push(c); return c }

onMounted(async () => {
  timer = setInterval(() => { clock.value = new Date().toLocaleTimeString('zh-CN', { hour12: false }) }, 1000)
  clock.value = new Date().toLocaleTimeString('zh-CN', { hour12: false })

  const [outputRes, gradeRes, lineRes, alarmRes, countRes, whRes]: any[] = await Promise.all([
    http.get('/stats/daily-output?days=7'),
    http.get('/stats/grade-distribution'),
    http.get('/stats/today-by-line'),
    http.get('/stats/alarm-frequency?days=30'),
    http.get('/alarms/active-count'),
    http.get('/warehouses'),
  ])

  const today = (outputRes.data || []).slice(-1)[0]
  if (today) { kpis.value[0].value = today.doff_count; kpis.value[1].value = today.bobbin_count }
  kpis.value[2].value = countRes.data?.count || 0
  if (whRes.data?.length) {
    const s: any = await http.get(`/warehouses/${whRes.data[0].id}/stats`)
    const d = s.data || {}; const total = (d.empty||0)+(d.occupied||0)
    kpis.value[3].value = (total > 0 ? Math.round((d.occupied/total)*100) : 0) + '%'
  }

  if (outputRef.value) {
    const data = outputRes.data || []
    initChart(outputRef.value).setOption({ title:{text:'7日产量趋势',textStyle:{fontSize:13}}, tooltip:{trigger:'axis'}, xAxis:{type:'category',data:data.map((d:any)=>d.date?.slice(5,10))}, yAxis:{type:'value'}, series:[{type:'bar',data:data.map((d:any)=>d.bobbin_count),itemStyle:{color:'#63e2b7'}}] })
  }
  if (gradeRef.value) {
    const data = gradeRes.data || []
    initChart(gradeRef.value).setOption({ title:{text:'等级分布',textStyle:{fontSize:13}}, tooltip:{trigger:'item'}, series:[{type:'pie',radius:['35%','65%'],data:data.map((d:any)=>({name:d.grade||d.final_grade,value:d.count}))}] })
  }
  if (lineRef.value) {
    const data = lineRes.data || []
    initChart(lineRef.value).setOption({ title:{text:'各线今日产量',textStyle:{fontSize:13}}, tooltip:{trigger:'axis'}, xAxis:{type:'category',data:data.map((d:any)=>d.line_code)}, yAxis:{type:'value'}, series:[{type:'bar',data:data.map((d:any)=>d.bobbin_count),itemStyle:{color:'#70c0e8'}}] })
  }
  if (alarmRef.value) {
    const data = alarmRes.data || []
    initChart(alarmRef.value).setOption({ title:{text:'告警频率(30天)',textStyle:{fontSize:13}}, tooltip:{trigger:'axis'}, xAxis:{type:'category',data:data.map((d:any)=>d.alarm_code||d.date)}, yAxis:{type:'value'}, series:[{type:'bar',data:data.map((d:any)=>d.count),itemStyle:{color:'#e88080'}}] })
  }
})

onUnmounted(() => { clearInterval(timer); charts.forEach(c => c.dispose()) })
</script>

<style scoped>
.board { height:100vh; background:#0d1117; padding:16px; overflow:auto; }
.board-header { display:flex; justify-content:space-between; align-items:center; margin-bottom:16px; }
.board-header h1 { font-size:20px; color:#00d4ff; margin:0; }
.header-right { display:flex; align-items:center; gap:16px; }
.time { font-size:18px; color:#aaa; font-family:monospace; }
.kpi-row { display:grid; grid-template-columns:repeat(4,1fr); gap:12px; margin-bottom:16px; }
.kpi-card { background:#161b22; border-radius:8px; padding:20px; text-align:center; border:1px solid #30363d; }
.kpi-value { font-size:36px; font-weight:bold; }
.kpi-label { font-size:13px; color:#8b949e; margin-top:4px; }
.chart-row { display:grid; grid-template-columns:1fr 1fr; gap:12px; margin-bottom:12px; }
.chart-box { background:#161b22; border-radius:8px; padding:12px; border:1px solid #30363d; }
</style>
