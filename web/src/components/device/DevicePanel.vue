<template>
  <div class="device-panel">
    <div v-for="node in nodes" :key="node.edge_id" class="device-card">
      <div class="device-header">
        <span :class="['status-dot', node.online ? 'online' : 'offline']"></span>
        <span class="device-name">{{ node.edge_id }}</span>
      </div>
      <div class="plc-list">
        <div v-for="(plc, code) in node.plc" :key="code" class="plc-item">
          <span :class="['plc-dot', plc.connected ? 'online' : 'offline']"></span>
          <span class="plc-code">{{ code }}</span>
          <span class="plc-latency">{{ plc.last_read_ms }}ms</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useWebSocket } from '../../composables/useWebSocket'

interface PLCStatus { connected: boolean; last_read_ms: number }
interface EdgeNode { edge_id: string; online: boolean; plc: Record<string, PLCStatus> }

const nodes = ref<EdgeNode[]>([])
const { lastMessage } = useWebSocket(['device'])

watch(lastMessage, (msg) => {
  if (!msg || msg.topic !== 'device.status') return
  const data = msg.data as { edge_id: string; plc: Record<string, PLCStatus> }
  const idx = nodes.value.findIndex(n => n.edge_id === data.edge_id)
  const node: EdgeNode = { edge_id: data.edge_id, online: true, plc: data.plc }
  if (idx >= 0) {
    nodes.value[idx] = node
  } else {
    nodes.value.push(node)
  }
})
</script>

<style scoped>
.device-panel { display: grid; grid-template-columns: repeat(auto-fill, minmax(240px, 1fr)); gap: 12px; }
.device-card { background: var(--bg-card); border-radius: 8px; padding: 12px; }
.device-header { display: flex; align-items: center; gap: 8px; margin-bottom: 8px; font-weight: 500; }
.device-name { font-size: 14px; }
.status-dot { width: 8px; height: 8px; border-radius: 50%; }
.online { background: var(--color-success); }
.offline { background: var(--color-danger); }
.plc-list { display: flex; flex-direction: column; gap: 4px; }
.plc-item { display: flex; align-items: center; gap: 6px; font-size: 12px; color: var(--text-secondary); }
.plc-dot { width: 6px; height: 6px; border-radius: 50%; }
.plc-code { flex: 1; }
.plc-latency { font-family: monospace; }
</style>
