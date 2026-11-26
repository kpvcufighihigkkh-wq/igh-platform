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
  // 字段名以接口为准：daily-output 返回 {date, doff_count, bobbin_count}，
  // 早期这里读 `count`，图上永远是空柱——接口有数据、页面没数字，最容易误判成后端坏了。
  if (outputChartRef.value) {
    const data = outputRes.data || []
    const chart = initChart(outputChartRef.value)
    chart.setOption({
      tooltip: { trigger: 'axis' },
      xAxis: { type: 'category', data: data.map((d: any) => (d.date || '').slice(5, 10)) },
      yAxis: { type: 'value', name: '丝饼数' },
      series: [{ type: 'bar', data: data.map((d: any) => d.bobbin_count), itemStyle: { color: '#63e2b7' } }],
    })
  }

  // Grade distribution：接口字段是 grade（不是 final_grade）
  if (gradeChartRef.value) {
    const data = gradeRes.data || []
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
  if (lineChartRef.value) {
    const data = lineRes.data || []
    const chart = initChart(lineChartRef.value)
    chart.setOption({
      tooltip: { trigger: 'axis' },
      xAxis: { type: 'category', data: data.map((d: any) => d.line_code || `Line ${d.line_id}`) },
      yAxis: { type: 'value', name: '丝饼数' },
      series: [{ type: 'bar', data: data.map((d: any) => d.bobbin_count), itemStyle: { color: '#70c0e8' } }],
    })
  }

  // Alarm frequency：接口按告警码聚合，字段是 alarm_code（不是 date）
  if (alarmChartRef.value) {
    const data = alarmRes.data || []
    const chart = initChart(alarmChartRef.value)
    chart.setOption({
      tooltip: { trigger: 'axis' },
      xAxis: { type: 'category', data: data.map((d: any) => d.alarm_code || '未知') },
      yAxis: { type: 'value', name: '次数' },
      series: [{ type: 'bar', data: data.map((d: any) => d.count), itemStyle: { color: '#e88080' } }],
    })
  }
}

onMounted(loadCharts)
onUnmounted(() => charts.forEach(c => c.dispose()))
</script>
