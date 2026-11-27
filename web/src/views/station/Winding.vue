<template>
  <div class="station">
    <!-- 顶部常驻：网络与同步状态。断网时必须能一眼看出，且要知道数据会不会丢 -->
    <header class="station-header">
      <div class="nameplate">
        <span class="nameplate-bar" aria-hidden="true"></span>
        <div class="nameplate-text">
          <h1>卷绕工位</h1>
          <span class="nameplate-sub">WINDING · 卷绕区域</span>
        </div>
      </div>
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
.station { height:100vh; background:var(--paper); padding:16px; display:flex; flex-direction:column; overflow:hidden; }
.station-header { display:flex; justify-content:space-between; align-items:center; margin-bottom:12px; flex-shrink:0; }

/* 机台铭牌：白底 + 左侧红色刻线，与登录页、管理端同一套品牌语言 */
.nameplate { display:flex; align-items:stretch; gap:12px; background:var(--plate); border:1px solid var(--line); border-radius:2px; padding:10px 18px 10px 14px; }
.nameplate-bar { width:4px; background:var(--igh-red); }
.nameplate-text { display:flex; flex-direction:column; gap:2px; }
.nameplate h1 { margin:0; font-family:var(--font-display); font-size:22px; font-weight:600; line-height:1.15; letter-spacing:.02em; color:var(--graphite); }
.nameplate-sub { font-family:var(--font-mono); font-size:11px; letter-spacing:.08em; color:var(--steel-light); }

.header-info { display:flex; align-items:center; gap:16px; }
.clock { font-family:var(--font-mono); font-size:22px; font-variant-numeric:tabular-nums; color:var(--graphite); }

.body { flex:1; display:flex; flex-direction:column; gap:12px; min-height:0; transition:filter .2s; }
.body.dim { filter:grayscale(.8); }

.row { display:flex; gap:12px; }
.row.grow { flex:1; min-height:0; }

.panel { background:var(--plate); border:1px solid var(--line); border-radius:2px; padding:16px; }
.panel h2 { margin:0 0 12px; padding-bottom:8px; border-bottom:1px solid var(--line-soft); font-size:13px; font-weight:600; letter-spacing:.04em; color:var(--steel); }

.lot-panel { flex:1; }
.lot-code { font-family:var(--font-mono); font-size:34px; font-weight:700; color:var(--graphite); }
.lot-meta { margin-top:6px; font-size:15px; color:var(--steel); }

.stat-panel { display:flex; gap:32px; align-items:center; }
.stat { text-align:center; min-width:96px; }
.stat-value { font-family:var(--font-display); font-size:44px; font-weight:600; line-height:1.1; font-variant-numeric:tabular-nums; color:var(--graphite); }
.stat-value.warn { font-size:26px; color:var(--warn); }
.stat-label { margin-top:4px; font-size:13px; color:var(--steel); }

.list-panel { flex:1.6; display:flex; flex-direction:column; min-height:0; }
.list { flex:1; overflow-y:auto; }
.list-item { display:flex; gap:16px; align-items:center; padding:10px 8px; border-bottom:1px solid var(--line-soft);
  font-size:17px; color:var(--graphite); transition:background .3s; }
/* 新落纱：浅红染 + 左侧红线，和告警同色系但更轻，不打断阅读 */
.list-item.fresh { background:var(--igh-red-tint); box-shadow:inset 3px 0 0 var(--igh-red); }
.mono { font-family:var(--font-mono); }
.muted { color:var(--steel-light); font-size:14px; }

.trolley-panel { flex:1; display:flex; flex-direction:column; min-height:0; }
.rfid-hint { margin-bottom:12px; font-family:var(--font-mono); font-size:19px; color:var(--graphite); }
.trolley-list { flex:1; overflow-y:auto; margin-top:12px; }

/* 主操作：112px 高，戴手套也能稳按；红色只出现在可执行的主操作上 */
.btn-primary {
  height:112px; width:100%; border:none; border-radius:2px; cursor:pointer;
  background:var(--igh-red); color:#fff; font-size:28px; font-weight:600;
  transition: transform .06s, background .15s;
}
.btn-primary:hover:not(:disabled) { background:var(--igh-red-hover); }
.btn-primary:active:not(:disabled) { transform:translateY(2px); background:var(--igh-red-deep); }
.btn-primary:disabled { background:var(--line); color:var(--steel-light); cursor:not-allowed; }

.toast {
  position:fixed; left:50%; top:84px; transform:translateX(-50%);
  background:var(--graphite); color:#fff; padding:14px 26px 14px 22px;
  border-left:4px solid var(--igh-red); border-radius:2px;
  font-size:21px; box-shadow:0 6px 18px rgba(37,40,43,.28); z-index:100;
}
.toast-enter-active, .toast-leave-active { transition:opacity .2s, transform .2s; }
.toast-enter-from, .toast-leave-to { opacity:0; transform:translate(-50%, -12px); }

.empty-hint { padding:12px 0; font-size:15px; color:var(--steel-light); }
</style>
