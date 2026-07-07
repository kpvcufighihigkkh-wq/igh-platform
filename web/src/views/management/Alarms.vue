<template>
  <n-card title="异常管理">
    <template #header-extra>
      <n-space>
        <n-select v-model:value="filterStatus" :options="statusOptions" style="width: 120px" clearable placeholder="状态" />
        <n-select v-model:value="filterLevel" :options="levelOptions" style="width: 120px" clearable placeholder="级别" />
      </n-space>
    </template>
    <n-data-table :columns="columns" :data="alarms" :bordered="false" :loading="loading" />
    <n-pagination
      v-model:page="page"
      :page-count="Math.ceil(total / 20)"
      style="margin-top: 12px; justify-content: flex-end"
      @update:page="fetchAlarms"
    />
  </n-card>
</template>

<script setup lang="ts">
import { ref, onMounted, watch, h } from 'vue'
import { NTag, NButton, NSpace } from 'naive-ui'
import http from '../../api/http'
import { tStatus, tLevel } from '../../utils/status'

const alarms = ref<any[]>([])
const total = ref(0)
const page = ref(1)
const loading = ref(false)
const filterStatus = ref<string | null>(null)
const filterLevel = ref<string | null>(null)

const statusOptions = [
  { label: '活跃', value: 'active' },
  { label: '已确认', value: 'acknowledged' },
  { label: '已解决', value: 'resolved' },
]
const levelOptions = [
  { label: '紧急', value: 'urgent' },
  { label: '重要', value: 'important' },
  { label: '一般', value: 'normal' },
]

const columns = [
  { title: '时间', key: 'created_at', width: 160, render: (r: any) => new Date(r.created_at).toLocaleString('zh-CN') },
  { title: '级别', key: 'level', width: 80, render: (r: any) => h(NTag, { type: r.level === 'urgent' ? 'error' : r.level === 'important' ? 'warning' : 'info', size: 'small' }, () => tLevel(r.level)) },
  { title: '来源', key: 'source_id', width: 120 },
  { title: '信息', key: 'message' },
  { title: '建议', key: 'suggestion', ellipsis: { tooltip: true } },
  { title: '状态', key: 'status', width: 90, render: (r: any) => h(NTag, { type: r.status === 'active' ? 'error' : r.status === 'acknowledged' ? 'warning' : 'success', size: 'small' }, () => tStatus(r.status)) },
  {
    title: '操作', key: 'actions', width: 140,
    render: (r: any) => h(NSpace, { size: 'small' }, () => [
      r.status === 'active' ? h(NButton, { size: 'tiny', onClick: () => ack(r.id) }, () => '确认') : null,
      r.status !== 'resolved' ? h(NButton, { size: 'tiny', type: 'success', onClick: () => resolve(r.id) }, () => '解决') : null,
    ])
  },
]

async function fetchAlarms() {
  loading.value = true
  const params = new URLSearchParams({ page: String(page.value), page_size: '20' })
  if (filterStatus.value) params.set('status', filterStatus.value)
  if (filterLevel.value) params.set('level', filterLevel.value)
  const res: any = await http.get(`/alarms?${params}`)
  alarms.value = res.data?.items || []
  total.value = res.data?.total || 0
  loading.value = false
}

async function ack(id: number) {
  await http.put(`/alarms/${id}/acknowledge`)
  fetchAlarms()
}

async function resolve(id: number) {
  await http.put(`/alarms/${id}/resolve`, { note: '' })
  fetchAlarms()
}

watch([filterStatus, filterLevel], () => { page.value = 1; fetchAlarms() })
onMounted(fetchAlarms)
</script>
