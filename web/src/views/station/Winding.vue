<template>
  <div class="station">
    <!-- 顶部常驻：网络与同步状态。断网时必须能一眼看出，且要知道数据会不会丢 -->
    <header class="station-header">
      <h1>卷绕工位</h1>
      <div class="header-info">
        <n-tag :type="online ? 'success' : 'error'" size="large" round>
          {{ online ? '在线' : `离线 · 待同步 ${queueDepth} 条` }}
        </n-tag>
        <span class="clock">{{ clock }}</span>
      </div>
    </header>

    <!-- 离线时整块工作区降饱和：非正常状态在余光里就能感知 -->
    <div class="body" :class="{ dim: !online }">
      <div class="row">
        <section class="panel lot-panel">
          <h2>当前批次</h2>
          <template v-if="lot">
            <div class="lot-code">{{ lot.code }}</div>
            <div class="lot-meta">{{ lot.specification }} · {{ lot.product_type }}</div>
          </template>
          <div v-else class="empty-hint">暂无活跃批次</div>
        </section>

        <section class="panel stat-panel">
          <div class="stat">
            <div class="stat-value">{{ todayDoff }}</div>
            <div class="stat-label">今日落纱</div>
          </div>
          <div class="stat">
            <div class="stat-value">{{ todayBobbin }}</div>
            <div class="stat-label">今日丝饼</div>
          </div>
          <div class="stat">
            <div class="stat-value" :class="{ warn: plcDelayed }">{{ plcDelayText }}</div>
            <div class="stat-label">PLC 数据</div>
          </div>
        </section>
      </div>

      <div class="row grow">
        <section class="panel list-panel">
          <h2>最近落纱</h2>
          <div class="list">
            <div v-for="d in doffings" :key="d.id" class="list-item" :class="{ fresh: isFresh(d) }">
              <span class="mono">{{ d.winder_name }}</span>
              <span class="mono">#{{ d.doff_no }}</span>
              <span class="mono">{{ d.bobbin_count }} 锭</span>
              <span class="muted">{{ formatTime(d.created_at) }}</span>
            </div>
            <div v-if="!doffings.length" class="empty-hint">暂无落纱记录</div>
          </div>
        </section>

        <section class="panel trolley-panel">
          <h2>料车</h2>
          <div class="rfid-hint">当前料车：{{ currentTrolley ?? '未读到' }}</div>
          <!-- 料车号由 PLC 读 RFID 后给出，不允许手打：手套环境下手打必错 -->
          <button
            class="btn-primary"
            :disabled="!currentTrolley || !online || busy"
            @click="loadTrolley"
          >
            {{ busy ? '处理中…' : '装车' }}
          </button>
          <div class="trolley-list">
            <div v-for="t in trolleys" :key="t.id" class="list-item">
              <span class="mono">#{{ t.number }}</span>
              <span class="muted">{{ statusText(t.status) }}</span>
            </div>
          </div>
        </section>
      </div>
    </div>

    <!-- 落筒提示：不是确认按钮，只是告知"刚刚落了一筒" -->
    <transition name="toast">
      <div v-if="toast" class="toast">{{ toast }}</div>
    </transition>

    <LoginModal />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import http from '../../api/http'
import { formatTime, tStatus } from '../../utils/status'
import LoginModal from '../../components/LoginModal.vue'

const clock = ref('')
const online = ref(true)
const queueDepth = ref(0)
const lot = ref<any>(null)
const todayDoff = ref(0)
const todayBobbin = ref(0)
const doffings = ref<any[]>([])
const trolleys = ref<any[]>([])
const busy = ref(false)
const toast = ref('')

// PLC 最近一次读到的料车 RFID。
// 值来自 /edge/plc/values，页面不生成、不修改，只展示与使用。
const currentTrolley = ref<number | null>(null)
const plcDelayed = ref(false)

// 已提示过的落纱 id：避免轮询每次都重复提示"新落纱"
const seenDoffIds = new Set<number>()
let firstLoadDone = false

let timer: any
let plcTimer: any

const plcDelayText = computed(() => (plcDelayed.value ? '延迟' : '正常'))

function statusText(s: string) {
  return tStatus(s)
}

function isFresh(d: any) {
  return Date.now() - new Date(d.created_at).getTime() < 60_000
}

function showToast(msg: string) {
  toast.value = msg
  setTimeout(() => { if (toast.value === msg) toast.value = '' }, 4000)
}

// 检测到新落纱时提示。
//
// 不做"确认"按钮：PLC 已经自动写库了，每筒弹一次确认，
// 一个班要点上千次，工人只会盲目连点，反而更容易错过真正需要留意的异常。
function notifyNewDoffings(list: any[]) {
  const fresh = list.filter((d) => !seenDoffIds.has(d.id))
  fresh.forEach((d) => seenDoffIds.add(d.id))
  // 首次加载只是把历史读出来，不提示
  if (!firstLoadDone) { firstLoadDone = true; return }
  if (fresh.length) {
    const latest = fresh[0]
    showToast(`新落纱：${latest.winder_name} #${latest.doff_no}（${latest.bobbin_count} 锭）`)
  }
}

async function loadStatus() {
  try {
    const r: any = await http.get('/edge/sync/status')
    online.value = r.data?.online !== false
    queueDepth.value = r.data?.queue_depth || 0
  } catch {
    online.value = false
  }
}

async function load() {
  try {
    const [lotRes, doffRes, trolleyRes, statsRes]: any[] = await Promise.all([
      http.get('/lots?status=active&page=1&page_size=1'),
      http.get('/doffings?page=1&page_size=10'),
      http.get('/trolleys?page=1&page_size=10'),
      http.get('/stats/daily-output?days=1'),
    ])
    lot.value = (lotRes.data?.items || [])[0] || null
    const ds = doffRes.data?.items || []
    notifyNewDoffings(ds)
    doffings.value = ds
    trolleys.value = trolleyRes.data?.items || []
    const today = (statsRes.data || [])[0]
    if (today) { todayDoff.value = today.doff_count; todayBobbin.value = today.bobbin_count }
  } catch {
    online.value = false
  }
  loadStatus()
}

// 料车 RFID 来自 PLC：页面只读，不提供手输入口。
async function loadPlc() {
  try {
    const r: any = await http.get('/edge/plc/values')
    plcDelayed.value = false
    const devices = (r.data?.devices || {}) as Record<string, Record<string, any>>
    for (const values of Object.values(devices)) {
      for (const [key, v] of Object.entries(values || {})) {
        // 点表里料车相关字段的命名不固定，按语义匹配
        if (/Trolley|RFID|Car_?No/i.test(key)) {
          const n = Number(v)
          if (Number.isFinite(n) && n > 0) { currentTrolley.value = n; return }
        }
      }
    }
    currentTrolley.value = null
  } catch {
    // 读不到 PLC 时明确显示"延迟"，而不是继续显示上一次的值——
    // 冻结的数字比错误提示更容易被误认为"机器停了"
    plcDelayed.value = true
    currentTrolley.value = null
  }
}

async function loadTrolley() {
  if (!currentTrolley.value || !lot.value) return
  busy.value = true
  try {
    await http.post('/trolleys/load', {
      number: currentTrolley.value,
      rfid: String(currentTrolley.value),
      lot_id: lot.value.id,
      doffing_ids: doffings.value.map((d) => d.id).join(','),
    })
    showToast(`料车 #${currentTrolley.value} 已装车`)
    await load()
  } catch (e: any) {
    showToast(e?.response?.data?.message || '装车失败')
  } finally {
    busy.value = false
  }
}

onMounted(() => {
  const tick = () => { clock.value = new Date().toLocaleTimeString('zh-CN', { hour12: false }) }
  tick()
  load()
  loadPlc()
  // 业务数据 2s：本机触摸屏轮询中心以外的本地接口，开销可接受
  timer = setInterval(() => { tick(); load() }, 2000)
  // PLC 值 1s：称重与 RFID 需要更跟手
  plcTimer = setInterval(loadPlc, 1000)
})
onUnmounted(() => { clearInterval(timer); clearInterval(plcTimer) })
</script>

<style scoped>
.station { height:100vh; background:#0d1117; padding:16px; display:flex; flex-direction:column; overflow:hidden; }
.station-header { display:flex; justify-content:space-between; align-items:center; margin-bottom:12px; flex-shrink:0; }
.station-header h1 { font-size:22px; color:#00d4ff; margin:0; }
.header-info { display:flex; align-items:center; gap:16px; }
.clock { font-family:monospace; color:#8b949e; font-size:20px; }

.body { flex:1; display:flex; flex-direction:column; gap:12px; min-height:0; transition:filter .2s; }
.body.dim { filter:grayscale(.75); }

.row { display:flex; gap:12px; }
.row.grow { flex:1; min-height:0; }

.panel { background:#161b22; border:1px solid #30363d; border-radius:10px; padding:16px; }
.panel h2 { margin:0 0 12px; font-size:16px; color:#8b949e; font-weight:500; }

.lot-panel { flex:1; }
.lot-code { font-size:34px; font-weight:700; color:#63e2b7; font-family:monospace; }
.lot-meta { font-size:16px; color:#8b949e; margin-top:6px; }

.stat-panel { display:flex; gap:32px; align-items:center; }
.stat { text-align:center; min-width:96px; }
.stat-value { font-size:44px; font-weight:700; color:#e6edf3; font-variant-numeric:tabular-nums; line-height:1.1; }
.stat-value.warn { color:#f0b429; font-size:28px; }
.stat-label { font-size:14px; color:#8b949e; margin-top:4px; }

.list-panel { flex:1.6; display:flex; flex-direction:column; min-height:0; }
.list { flex:1; overflow-y:auto; }
.list-item { display:flex; gap:16px; align-items:center; padding:10px 8px; border-bottom:1px solid #21262d;
  font-size:18px; color:#c9d1d9; transition:background .3s; }
.list-item.fresh { background:#052e16; }
.mono { font-family:monospace; }
.muted { color:#6e7681; font-size:15px; }

.trolley-panel { flex:1; display:flex; flex-direction:column; min-height:0; }
.rfid-hint { font-size:20px; color:#c9d1d9; margin-bottom:12px; font-family:monospace; }
.trolley-list { flex:1; overflow-y:auto; margin-top:12px; }

/* 主操作：112px 高，戴手套也能稳按 */
.btn-primary {
  height:112px; width:100%; border-radius:10px; border:none; cursor:pointer;
  background:#1f6feb; color:#fff; font-size:28px; font-weight:600;
  transition: transform .06s, filter .06s;
}
.btn-primary:active:not(:disabled) { transform:translateY(2px); filter:brightness(1.35); }
.btn-primary:disabled { background:#30363d; color:#6e7681; cursor:not-allowed; }

.toast {
  position:fixed; left:50%; top:80px; transform:translateX(-50%);
  background:#1f6feb; color:#fff; padding:14px 28px; border-radius:10px;
  font-size:22px; box-shadow:0 6px 20px rgba(0,0,0,.4); z-index:100;
}
.toast-enter-active, .toast-leave-active { transition:opacity .2s, transform .2s; }
.toast-enter-from, .toast-leave-to { opacity:0; transform:translate(-50%, -12px); }

.empty-hint { color:#6e7681; font-size:16px; padding:12px 0; }
</style>
