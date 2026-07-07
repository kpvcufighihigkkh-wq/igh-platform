<template>
  <div>
    <n-card title="统计分析">
      <n-grid :cols="2" :x-gap="16" :y-gap="16">
        <n-gi>
          <n-card title="每日产量趋势" size="small">
            <div ref="outputChartRef" style="height: 280px"></div>
          </n-card>
        </n-gi>
        <n-gi>
          <n-card title="等级分布" size="small">
            <div ref="gradeChartRef" style="height: 280px"></div>
          </n-card>
        </n-gi>
        <n-gi>
          <n-card title="今日各线产量" size="small">
            <div ref="lineChartRef" style="height: 280px"></div>
          </n-card>
        </n-gi>
        <n-gi>
          <n-card title="异常频率 (30天)" size="small">
            <div ref="alarmChartRef" style="height: 280px"></div>
          </n-card>
        </n-gi>
      </n-grid>
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import * as echarts from 'echarts'
import http from '../../api/http'

const outputChartRef = ref<HTMLElement>()
const gradeChartRef = ref<HTMLElement>()
const lineChartRef = ref<HTMLElement>()
const alarmChartRef = ref<HTMLElement>()

let charts: echarts.ECharts[] = []

function initChart(el: HTMLElement) {
  const c = echarts.init(el, 'dark')
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
  if (outputChartRef.value) {
    const data = outputRes.data || []
    const chart = initChart(outputChartRef.value)
    chart.setOption({
      tooltip: { trigger: 'axis' },
      xAxis: { type: 'category', data: data.map((d: any) => d.date) },
      yAxis: { type: 'value', name: '丝饼数' },
      series: [{ type: 'bar', data: data.map((d: any) => d.count), itemStyle: { color: '#63e2b7' } }],
    })
  }

  // Grade distribution
  if (gradeChartRef.value) {
    const data = gradeRes.data || []
    const chart = initChart(gradeChartRef.value)
    chart.setOption({
      tooltip: { trigger: 'item' },
      series: [{
        type: 'pie', radius: ['40%', '70%'],
        data: data.map((d: any) => ({ name: d.final_grade || 'N/A', value: d.count })),
      }],
    })
  }

  // Today by line
  if (lineChartRef.value) {
    const data = lineRes.data || []
    const chart = initChart(lineChartRef.value)
    chart.setOption({
      tooltip: { trigger: 'axis' },
      xAxis: { type: 'category', data: data.map((d: any) => d.line_code || `Line ${d.line_id}`) },
      yAxis: { type: 'value', name: '丝饼数' },
      series: [{ type: 'bar', data: data.map((d: any) => d.count), itemStyle: { color: '#70c0e8' } }],
    })
  }

  // Alarm frequency
  if (alarmChartRef.value) {
    const data = alarmRes.data || []
    const chart = initChart(alarmChartRef.value)
    chart.setOption({
      tooltip: { trigger: 'axis' },
      xAxis: { type: 'category', data: data.map((d: any) => d.date) },
      yAxis: { type: 'value', name: '次数' },
      series: [{ type: 'line', data: data.map((d: any) => d.count), smooth: true, areaStyle: { opacity: 0.3 }, itemStyle: { color: '#e88080' } }],
    })
  }
}

onMounted(loadCharts)
onUnmounted(() => charts.forEach(c => c.dispose()))
</script>
