<template>
  <div>
    <section class="kpi-row">
      <div class="kpi">
        <span class="kpi-label">队列总数</span>
        <span class="kpi-value">{{ stats.total }}</span>
      </div>
      <div class="kpi">
        <span class="kpi-label">待处理</span>
        <span class="kpi-value">{{ stats.pending }}</span>
      </div>
      <div class="kpi">
        <span class="kpi-label">失败</span>
        <span class="kpi-value" :class="{ alarm: stats.failed > 0 }">{{ stats.failed }}</span>
        <n-button v-if="stats.failed > 0" size="tiny" type="warning" class="retry" @click="retry">重试失败</n-button>
      </div>
    </section>

    <n-card size="small" class="queue-card">
      <template #header><span class="panel-title">同步队列</span></template>
      <n-data-table :columns="cols" :data="queue" :bordered="false" size="small" />
    </n-card>
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

<style scoped>
.kpi-row {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
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
.retry { align-self: flex-start; }
.queue-card { margin-top: 16px; }
.panel-title {
  font-family: var(--font-display);
  font-size: 15px;
  font-weight: 600;
  color: var(--steel);
}
</style>
