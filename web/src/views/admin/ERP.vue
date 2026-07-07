<template>
  <div>
    <n-grid :cols="3" :x-gap="12" style="margin-bottom:16px">
      <n-gi>
        <n-statistic label="队列总数" :value="stats.total" />
      </n-gi>
      <n-gi>
        <n-statistic label="待处理" :value="stats.pending" />
      </n-gi>
      <n-gi>
        <n-statistic label="失败" :value="stats.failed">
          <template #suffix>
            <n-button v-if="stats.failed > 0" size="tiny" type="warning" @click="retry" style="margin-left:8px">重试失败</n-button>
          </template>
        </n-statistic>
      </n-gi>
    </n-grid>
    <n-data-table :columns="cols" :data="queue" :bordered="false" size="small" />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, h } from 'vue'
import { NTag } from 'naive-ui'
import http from '../../api/http'
import { tStatus, formatTime } from '../../utils/status'

const stats = ref({ total: 0, pending: 0, failed: 0 })
const queue = ref<any[]>([])

const cols = [
  { title: '类型', key: 'type', width: 100 },
  { title: '关联ID', key: 'reference_id', width: 80 },
  { title: '状态', key: 'status', width: 80, render: (r: any) => h(NTag, { type: r.status === 'failed' ? 'error' : r.status === 'synced' ? 'success' : 'default', size: 'small' }, () => tStatus(r.status)) },
  { title: '重试次数', key: 'retry_count', width: 70 },
  { title: '错误信息', key: 'last_error', ellipsis: { tooltip: true } },
  { title: '时间', key: 'created_at', width: 150, render: (r: any) => formatTime(r.created_at) },
]

async function load() {
  try {
    const [sRes, qRes]: any[] = await Promise.all([
      http.get('/erp/status'),
      http.get('/erp/queue?page=1&page_size=50'),
    ])
    stats.value = sRes.data || { total: 0, pending: 0, failed: 0 }
    queue.value = qRes.data?.items || []
  } catch {}
}

async function retry() {
  await http.post('/erp/retry')
  load()
}

onMounted(load)
</script>
