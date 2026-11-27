<template>
  <n-grid :cols="2" :x-gap="16" :y-gap="16">
    <n-gi>
      <n-card title="每日产量趋势" size="small">
        <div v-if="empty.output" class="chart-empty">暂无数据</div>
        <div v-else ref="outputChartRef" style="height: 280px"></div>
      </n-card>
    </n-gi>
    <n-gi>
      <n-card title="等级分布" size="small">
        <div v-if="empty.grade" class="chart-empty">暂无数据</div>
        <div v-else ref="gradeChartRef" style="height: 280px"></div>
      </n-card>
    </n-gi>
    <n-gi>
      <n-card title="今日各线产量" size="small">
        <div v-if="empty.line" class="chart-empty">暂无数据</div>
        <div v-else ref="lineChartRef" style="height: 280px"></div>
      </n-card>
    </n-gi>
    <n-gi>
      <n-card title="异常频率 (30天)" size="small">
        <div v-if="empty.alarm" class="chart-empty">暂无数据</div>
        <div v-else ref="alarmChartRef" style="height: 280px"></div>
      </n-card>
    </n-gi>
  </n-grid>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import * as echarts from 'echarts'
import http from '../../api/http'

const outputChartRef = ref<HTMLElement>()
const gradeChartRef = ref<HTMLElement>()
const lineChartRef = ref<HTMLElement>()
const alarmChartRef = ref<HTMLElement>()

// 没有数据时画空坐标轴会让人误以为图表坏了，直接给"暂无数据"。
const empty = ref({ output: false, grade: false, line: false, alarm: false })

let charts: echarts.ECharts[] = []

function initChart(el: HTMLElement) {
  const c = echarts.init(el)
  charts.push(c)
  return c
}

async function loadCharts() {
  const [outputRes, gradeRes, lineRes, alarmRes]: any[] = await Promise.all([
    http.get('/stats/daily-output?days=7'),
    http.get('/stats/grade-distribution'),
    http.get('/stats/today-by-line'),
    http.get('/stats/alarm-frequency?days=30'),
  ])

  // Daily output
  // 字段名以接口为准：daily-output 返回 {date, doff_count, bobbin_count}，
  // 早期这里读 `count`，图上永远是空柱——接口有数据、页面没数字，最容易误判成后端坏了。
  const outputData = outputRes.data || []
  empty.value.output = outputData.length === 0
  if (!empty.value.output && outputChartRef.value) {
    const data = outputData
    const chart = initChart(outputChartRef.value)
    chart.setOption({
      tooltip: { trigger: 'axis' },
      textStyle: { color: '#3e4245' },
      grid: { left: 44, right: 16, top: 28, bottom: 28 },
      xAxis: { type: 'category', data: data.map((d: any) => (d.date || '').slice(5, 10)), axisLine: { lineStyle: { color: '#c4c0b8' } }, axisLabel: { color: '#5a5e62' } },
      yAxis: { type: 'value', name: '丝饼数', splitLine: { lineStyle: { color: '#ebe8e2' } }, axisLabel: { color: '#5a5e62' } },
      series: [{ type: 'bar', barMaxWidth: 28, data: data.map((d: any) => d.bobbin_count), itemStyle: { color: '#e02118' } }],
    })
  }

  // Grade distribution：接口字段是 grade（不是 final_grade）
  const gradeData = gradeRes.data || []
  empty.value.grade = gradeData.length === 0
  if (!empty.value.grade && gradeChartRef.value) {
    const data = gradeData
    const chart = initChart(gradeChartRef.value)
    chart.setOption({
      tooltip: { trigger: 'item' },
      series: [{
        type: 'pie', radius: ['40%', '70%'],
        data: data.map((d: any) => ({ name: d.grade || '未知', value: d.count })),
      }],
    })
  }

  // Today by line：接口给的是 bobbin_count
  const lineData = lineRes.data || []
  empty.value.line = lineData.length === 0
  if (!empty.value.line && lineChartRef.value) {
    const data = lineData
    const chart = initChart(lineChartRef.value)
    chart.setOption({
      tooltip: { trigger: 'axis' },
      textStyle: { color: '#3e4245' },
      grid: { left: 44, right: 16, top: 28, bottom: 28 },
      xAxis: { type: 'category', data: data.map((d: any) => d.line_code || `Line ${d.line_id}`), axisLine: { lineStyle: { color: '#c4c0b8' } }, axisLabel: { color: '#5a5e62' } },
      yAxis: { type: 'value', name: '丝饼数', splitLine: { lineStyle: { color: '#ebe8e2' } }, axisLabel: { color: '#5a5e62' } },
      series: [{ type: 'bar', barMaxWidth: 28, data: data.map((d: any) => d.bobbin_count), itemStyle: { color: '#25282b' } }],
    })
  }

  // Alarm frequency：接口按告警码聚合，字段是 alarm_code（不是 date）
  const alarmData = alarmRes.data || []
  empty.value.alarm = alarmData.length === 0
  if (!empty.value.alarm && alarmChartRef.value) {
    const data = alarmData
    const chart = initChart(alarmChartRef.value)
    chart.setOption({
      tooltip: { trigger: 'axis' },
      textStyle: { color: '#3e4245' },
      grid: { left: 44, right: 16, top: 28, bottom: 28 },
      xAxis: { type: 'category', data: data.map((d: any) => d.alarm_code || '未知'), axisLine: { lineStyle: { color: '#c4c0b8' } }, axisLabel: { color: '#5a5e62' } },
      yAxis: { type: 'value', name: '次数', splitLine: { lineStyle: { color: '#ebe8e2' } }, axisLabel: { color: '#5a5e62' } },
      series: [{ type: 'bar', barMaxWidth: 28, data: data.map((d: any) => d.count), itemStyle: { color: '#b72e24' } }],
    })
  }
}

onMounted(loadCharts)
onUnmounted(() => charts.forEach(c => c.dispose()))
</script>

<style scoped>
.chart-empty {
  height: 280px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1px dashed var(--line);
  color: var(--steel-light);
  font-size: 13px;
  letter-spacing: 0.04em;
}
</style>
