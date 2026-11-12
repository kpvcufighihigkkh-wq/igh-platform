import { ref, computed } from 'vue'
import http from '../api/http'

interface EdgeInfo {
  edge_id: string
  type: string
  line: { code: string; name: string }
  mode: 'edge'
}

const edgeInfo = ref<EdgeInfo | null>(null)
const isEdge = ref(false)
const serverOnline = ref(true)
const checking = ref(false)

export function useEdge() {
  async function detect() {
    if (checking.value) return
    checking.value = true
    try {
      const res: any = await http.get('/edge/info')
      if (res.data?.mode === 'edge') {
        edgeInfo.value = res.data
        isEdge.value = true
      }
    } catch {
      isEdge.value = false
    } finally {
      checking.value = false
    }
  }

  async function checkServer() {
    if (!isEdge.value) return
    try {
      const res = await fetch('/health', { signal: AbortSignal.timeout(3000) })
      serverOnline.value = res.ok
    } catch {
      serverOnline.value = false
    }
  }

  const edgeLabel = computed(() => {
    if (!edgeInfo.value) return ''
    return `${edgeInfo.value.line.name} (${edgeInfo.value.edge_id})`
  })

  return { edgeInfo, isEdge, serverOnline, edgeLabel, detect, checkServer }
}
