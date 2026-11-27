<template>
  <div class="board">
    <header class="board-header">
      <div class="brand">
        <span class="title-bar" aria-hidden="true"></span>
        <h1>生产看板</h1>
        <span class="sub">华祥智纤（九江）· 卷绕线</span>
      </div>
      <div class="header-right">
        <span class="time">{{ clock }}</span>
        <n-button text size="small" @click="$router.push('/login')">管理登录</n-button>
      </div>
    </header>

    <!-- 计数牌：像机台计数器一样横排，数字用窄体，便于远距离读数 -->
    <section class="counter-row">
      <div class="counter" v-for="kpi in kpis" :key="kpi.label">
        <span class="counter-label">{{ kpi.label }}</span>
        <span class="counter-value" :class="kpi.tone">{{ kpi.value }}</span>
      </div>
    </section>

    <div class="chart-row">
      <div class="chart-box">
        <div class="chart-title">7 日产量趋势</div>
        <div v-if="empty.output" class="chart-empty" style="height:260px">暂无数据</div>
        <div v-else ref="outputRef" style="height:260px"></div>
      </div>
      <div class="chart-box">
        <div class="chart-title">等级分布</div>
        <div v-if="empty.grade" class="chart-empty" style="height:260px">暂无数据</div>
        <div v-else ref="gradeRef" style="height:260px"></div>
      </div>
    </div>
    <div class="chart-row">
      <div class="chart-box">
        <div class="chart-title">各线今日产量</div>
        <div v-if="empty.line" class="chart-empty" style="height:240px">暂无数据</div>
        <div v-else ref="lineRef" style="height:240px"></div>
      </div>
      <div class="chart-box">
        <div class="chart-title">告警频率（30 天）</div>
        <div v-if="empty.alarm" class="chart-empty" style="height:240px">暂无数据</div>
        <div v-else ref="alarmRef" style="height:240px"></div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import * as echarts from 'echarts'
import http from '../../api/http'

const clock = ref('')
const kpis = ref([
  { label: '今日落纱', value: 0, tone: '' },
  { label: '今日丝饼', value: 0, tone: '' },
  { label: '活跃告警', value: 0, tone: 'alarm' },
  { label: '仓库占用率', value: '0%', tone: 'warn' },
])

const outputRef = ref<HTMLElement>()
const gradeRef = ref<HTMLElement>()
const lineRef = ref<HTMLElement>()
const alarmRef = ref<HTMLElement>()

// 空数据集画空坐标轴会让人以为大屏坏了，直接显示"暂无数据"。
const empty = ref({ output: false, grade: false, line: false, alarm: false })
let charts: echarts.ECharts[] = []
let timer: any

// 与全站共用一套图表样式：细轴线、暖灰文字、品牌红作主数据色。
const AXIS = { axisLine: { lineStyle: { color: '#c4c0b8' } }, axisLabel: { color: '#5a5e62' } }
const SPLIT = { lineStyle: { color: '#ebe8e2' } }
const TEXT = { color: '#3e4245', fontFamily: 'inherit' }

function initChart(el: HTMLElement) {
  const c = echarts.init(el)
  charts.push(c)
  return c
}

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

  // 「今日」必须按日期精确匹配，不能取数组最后一条：
  // 停产的周末/节假日最后一条是历史日期，直接取末位会把那天的产量当成今天的显示。
  const todayStr = new Date().toLocaleDateString('sv-SE')
  const today = (outputRes.data || []).find((d: any) => (d.date || '').slice(0, 10) === todayStr)
  if (today) { kpis.value[0].value = today.doff_count; kpis.value[1].value = today.bobbin_count }
  kpis.value[2].value = countRes.data?.count || 0
  if (whRes.data?.length) {
    const s: any = await http.get(`/warehouses/${whRes.data[0].id}/stats`)
    const d = s.data || {}; const total = (d.empty || 0) + (d.occupied || 0)
    kpis.value[3].value = (total > 0 ? Math.round((d.occupied / total) * 100) : 0) + '%'
  }

  const outputData = outputRes.data || []
  empty.value.output = outputData.length === 0
  if (!empty.value.output && outputRef.value) {
    const data = outputData
    initChart(outputRef.value).setOption({
      tooltip: { trigger: 'axis' },
      grid: { left: 40, right: 16, top: 16, bottom: 28 },
      xAxis: { type: 'category', data: data.map((d: any) => d.date?.slice(5, 10)), ...AXIS },
      yAxis: { type: 'value', splitLine: SPLIT, axisLabel: AXIS.axisLabel },
      series: [{
        type: 'bar', barMaxWidth: 28,
        data: data.map((d: any) => d.bobbin_count),
        itemStyle: { color: '#e02118' },
      }],
    })
  }
  const gradeData = gradeRes.data || []
  empty.value.grade = gradeData.length === 0
  if (!empty.value.grade && gradeRef.value) {
    const data = gradeData
    initChart(gradeRef.value).setOption({
      tooltip: { trigger: 'item' },
      color: ['#e02118', '#25282b', '#5a5e62', '#2f7d4f', '#a56a00', '#8a8d8f', '#b72e24'],
      textStyle: TEXT,
      series: [{
        type: 'pie', radius: ['38%', '66%'],
        itemStyle: { borderColor: '#ffffff', borderWidth: 2 },
        label: { color: '#3e4245' },
        data: data.map((d: any) => ({ name: d.grade || d.final_grade || '未知', value: d.count })),
      }],
    })
  }
  const lineData = lineRes.data || []
  empty.value.line = lineData.length === 0
  if (!empty.value.line && lineRef.value) {
    const data = lineData
    initChart(lineRef.value).setOption({
      tooltip: { trigger: 'axis' },
      grid: { left: 40, right: 16, top: 16, bottom: 28 },
      xAxis: { type: 'category', data: data.map((d: any) => d.line_code), ...AXIS },
      yAxis: { type: 'value', splitLine: SPLIT, axisLabel: AXIS.axisLabel },
      series: [{
        type: 'bar', barMaxWidth: 28,
        data: data.map((d: any) => d.bobbin_count),
        itemStyle: { color: '#25282b' },
      }],
    })
  }
  const alarmData = alarmRes.data || []
  empty.value.alarm = alarmData.length === 0
  if (!empty.value.alarm && alarmRef.value) {
    const data = alarmData
    initChart(alarmRef.value).setOption({
      tooltip: { trigger: 'axis' },
      grid: { left: 40, right: 16, top: 16, bottom: 28 },
      xAxis: { type: 'category', data: data.map((d: any) => d.alarm_code || d.date), ...AXIS },
      yAxis: { type: 'value', splitLine: SPLIT, axisLabel: AXIS.axisLabel },
      series: [{
        type: 'bar', barMaxWidth: 28,
        data: data.map((d: any) => d.count),
        itemStyle: { color: '#b72e24' },
      }],
    })
  }
})

onUnmounted(() => { clearInterval(timer); charts.forEach(c => c.dispose()) })
</script>

<style scoped>
.board {
  height: 100vh;
  padding: 16px;
  overflow: auto;
  background: var(--paper);
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.board-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0 0 12px;
  border-bottom: 2px solid var(--graphite);
  flex-shrink: 0;
}
.brand {
  display: flex;
  align-items: baseline;
  gap: 12px;
}
.title-bar {
  align-self: center;
  width: 4px;
  height: 22px;
  background: var(--igh-red);
}
.brand h1 {
  margin: 0;
  font-family: var(--font-display);
  font-size: 26px;
  font-weight: 600;
  letter-spacing: 0.02em;
  color: var(--graphite);
}
.sub {
  font-size: 13px;
  color: var(--steel);
}
.header-right {
  display: flex;
  align-items: center;
  gap: 16px;
}
.time {
  font-family: var(--font-mono);
  font-size: 22px;
  font-variant-numeric: tabular-nums;
  color: var(--graphite);
}

.counter-row {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  background: var(--plate);
  border: 1px solid var(--line);
  border-radius: 2px;
  flex-shrink: 0;
}
.counter {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 18px 24px;
  position: relative;
}
.counter + .counter::before {
  content: '';
  position: absolute;
  left: 0;
  top: 16px;
  bottom: 16px;
  width: 1px;
  background: var(--line-soft);
}
.counter-label {
  font-size: 13px;
  color: var(--steel);
}
.counter-value {
  font-family: var(--font-display);
  font-size: 46px;
  font-weight: 600;
  line-height: 1;
  font-variant-numeric: tabular-nums;
  color: var(--graphite);
}
.counter-value.alarm { color: var(--alarm); }
.counter-value.warn { color: var(--warn); }

.chart-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}
.chart-box {
  background: var(--plate);
  border: 1px solid var(--line);
  border-radius: 2px;
  padding: 14px 16px 8px;
}
.chart-title {
  font-family: var(--font-display);
  font-size: 15px;
  font-weight: 600;
  color: var(--steel);
  padding-bottom: 10px;
  border-bottom: 1px solid var(--line-soft);
  margin-bottom: 8px;
}
.chart-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1px dashed var(--line);
  color: var(--steel-light);
  font-size: 13px;
  letter-spacing: 0.04em;
}

@media (max-width: 1100px) {
  .counter-row { grid-template-columns: repeat(2, 1fr); }
  .chart-row { grid-template-columns: 1fr; }
}
</style>
