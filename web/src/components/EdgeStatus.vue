<template>
  <div v-if="isEdge" class="edge-status">
    <span class="edge-dot" :class="{ online: serverOnline, offline: !serverOnline }"></span>
    <span class="edge-text">{{ edgeLabel }} - {{ serverOnline ? '在线' : '离线' }}</span>
  </div>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted } from 'vue'
import { useEdge } from '../composables/useEdge'

const { isEdge, serverOnline, edgeLabel, detect, checkServer } = useEdge()

let timer: ReturnType<typeof setInterval> | null = null

onMounted(async () => {
  await detect()
  if (isEdge.value) {
    timer = setInterval(checkServer, 15000)
  }
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
})
</script>

<style scoped>
.edge-status {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  padding: 2px 10px;
  border-radius: 2px;
  border: 1px solid var(--line);
  background: var(--plate);
}
.edge-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
}
.edge-dot.online { background: var(--ok); }
.edge-dot.offline { background: var(--alarm); }
.edge-text { color: var(--steel); }
</style>
