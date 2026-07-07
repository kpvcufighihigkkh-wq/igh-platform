import { ref, onUnmounted } from 'vue'
import type { Ref } from 'vue'

export function useWebSocket(topics: string[]) {
  const connected = ref(false)
  const lastMessage: Ref<any> = ref(null)
  let ws: WebSocket | null = null
  let reconnectTimer: number | null = null

  function connect() {
    const token = localStorage.getItem('token')
    if (!token) return

    const subscribe = topics.join(',')
    const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:'
    ws = new WebSocket(`${protocol}//${location.host}/api/v1/ws?token=${token}&subscribe=${subscribe}`)

    ws.onopen = () => { connected.value = true }
    ws.onclose = () => {
      connected.value = false
      reconnectTimer = window.setTimeout(connect, 3000)
    }
    ws.onmessage = (event) => {
      try { lastMessage.value = JSON.parse(event.data) } catch {}
    }
  }

  function disconnect() {
    if (reconnectTimer) clearTimeout(reconnectTimer)
    ws?.close()
    ws = null
  }

  connect()
  onUnmounted(disconnect)

  return { connected, lastMessage, disconnect }
}
