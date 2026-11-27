<template>
  <div class="station">
    <header class="station-header">
      <div class="nameplate">
        <span class="nameplate-bar" aria-hidden="true"></span>
        <div class="nameplate-text">
          <h1>包装工位</h1>
          <span class="nameplate-sub">PACKAGING · 包装区域</span>
        </div>
      </div>
      <div class="header-info">
        <n-tag :type="online ? 'success' : 'error'" size="large" round>
          {{ online ? '在线' : `离线 · 待同步 ${queueDepth} 条` }}
        </n-tag>
        <span class="clock">{{ clock }}</span>
      </div>
    </header>

    <div class="body" :class="{ dim: !online }">
      <div class="row grow">
        <!-- 垛型网格：回答"下一个装哪"，进度条只能回答"装了多少" -->
        <section class="panel grid-panel">
          <h2>
            当前栈板
            <span v-if="layout" class="layout-tag">{{ layout.code }}</span>
          </h2>

          <template v-if="layout">
            <div class="progress-line">
              已装 <span class="mono big">{{ loaded }}</span> / {{ layout.capacity }} 锭
              <span v-if="currentCell" class="cell-hint">
                · 当前 第{{ currentCell.layer }}层第{{ currentCell.position }}位
              </span>
            </div>

            <!-- 层从上层往下渲染：视觉上像在往下堆 -->
            <div class="layers">
              <div v-for="l in layout.layers.slice().reverse()" :key="l.layer" class="layer">
                <span class="layer-label">L{{ l.layer }}</span>
                <div class="cells">
                  <div
                    v-for="p in l.positions"
                    :key="p"
                    class="cell"
                    :class="cellState(l.layer, p)"
                  />
                </div>
              </div>
            </div>
          </template>

          <div v-else class="empty-hint">
            未配置垛型。<code>pallets</code> 段需要在现场配置里填写，否则无法显示装载位置。
          </div>
        </section>

        <section class="panel side-panel">
          <h2>当前订单</h2>
          <template v-if="activeOrder">
            <div class="order-progress">
              <span class="mono big">{{ activeOrder.packed_quantity }}</span>
              <span class="muted"> / {{ activeOrder.target_quantity }}</span>
            </div>
            <div class="kv"><span>等级</span><span class="mono">{{ activeOrder.grade_code }}</span></div>
            <div class="kv"><span>状态</span><span>{{ statusText(activeOrder.status) }}</span></div>
          </template>
          <div v-else class="empty-hint">暂无进行中的订单</div>

          <h2 style="margin-top:16px">PLC 已装锭数</h2>
          <div class="plc-count" :class="{ warn: plcDelayed }">
            {{ plcDelayed ? 'PLC 未连接' : (plcCount ?? '—') }}
          </div>

          <button class="btn-primary" :disabled="!pallets.length" @click="openPrint">
            补打标签
          </button>
        </section>
      </div>

      <section class="panel list-panel">
        <h2>最近栈板</h2>
        <div class="list">
          <div v-for="p in pallets" :key="p.id" class="list-item">
            <span class="mono">{{ p.code }}</span>
            <span class="mono">{{ p.grade_code || '—' }}</span>
            <span class="mono">{{ p.bobbin_count }} 锭</span>
            <span class="muted">{{ formatTime(p.created_at) }}</span>
          </div>
          <div v-if="!pallets.length" class="empty-hint">暂无栈板记录</div>
        </div>
      </section>
    </div>

    <n-modal v-model:show="showCreateOrder" preset="card" title="新建包装订单" style="width:420px">
      <n-form label-placement="left" label-width="80">
        <n-form-item label="批次">
          <n-select v-model:value="newOrder.lot_id" :options="lotOpts" filterable placeholder="选择批次" />
        </n-form-item>
        <n-form-item label="等级">
          <n-input v-model:value="newOrder.grade_code" placeholder="A / AA / B" />
        </n-form-item>
        <n-form-item label="目标数量">
          <n-input-number v-model:value="newOrder.target_quantity" :min="1" style="width:100%" />
        </n-form-item>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showCreateOrder = false">取消</n-button>
          <n-button type="primary" :loading="creating" @click="doCreateOrder">创建</n-button>
        </n-space>
      </template>
    </n-modal>

    <LoginModal />
    <PrintLabelModal v-model:show="showPrint" :preset="printPreset" />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import http from '../../api/http'
import { tStatus, formatTime } from '../../utils/status'
import { useAuth } from '../../composables/useAuth'
import { useMessage } from 'naive-ui'
import LoginModal from '../../components/LoginModal.vue'
import PrintLabelModal from '../../components/PrintLabelModal.vue'

const { requireAuth } = useAuth()
const message = useMessage()

const clock = ref('')
const online = ref(true)
const queueDepth = ref(0)
const orders = ref<any[]>([])
const pallets = ref<any[]>([])
const lots = ref<any[]>([])
const showPrint = ref(false)
const printPreset = ref<any>({})

// 垛型来自现场配置（/edge/config 的 applied.pallets）。
// 不写死层数与列数：不同产品/包装机的垛型不同，且 PLC 只给得出总层数。
const layout = ref<any>(null)

// PLC 实时已装锭数：用来算"当前装到第几格"
const plcCount = ref<number | null>(null)
const plcDelayed = ref(false)

const creating = ref(false)
const showCreateOrder = ref(false)
const newOrder = ref<{ lot_id: number | null; grade_code: string; target_quantity: number }>({
  lot_id: null, grade_code: 'A', target_quantity: 48,
})
const lotOpts = computed(() => lots.value.map(l => ({ label: `${l.code} ${l.name || ''}`.trim(), value: l.id })))

let timer: any
let plcTimer: any

const activeOrder = computed(() => orders.value.find(o => o.status === 'in_progress'))

// loaded：以 PLC 为准。PLC 读不到时退回最近一垛的记录数，
// 但界面上会明确标 PLC 未连接——冻结的数字会被误读成"没在装"。
const loaded = computed(() => {
  if (plcCount.value !== null) return plcCount.value
  return 0
})

// 把"整垛第 n 锭"换算成层/位。与后端 bobbincode 的 Locate 同一套语义：
// 每位置放 bobbins_per_position 锭，按层从下往上顺序填。
const currentCell = computed(() => {
  if (!layout.value || plcCount.value === null || plcCount.value <= 0) return null
  const per = layout.value.bobbins_per_position || 1
  const posIndex = Math.floor((plcCount.value - 1) / per) + 1
  let acc = 0
  for (const l of layout.value.layers) {
    if (posIndex <= acc + l.positions) {
      return { layer: l.layer, position: posIndex - acc }
    }
    acc += l.positions
  }
  return null
})

function cellState(layer: number, position: number) {
  if (!layout.value) return {}
  const per = layout.value.bobbins_per_position || 1
  // 该格之前已经装了多少锭
  let acc = 0
  for (const l of layout.value.layers) {
    if (l.layer === layer) { acc += position - 1; break }
    acc += l.positions
  }
  const filled = acc * per
  const cur = currentCell.value
  return {
    filled: loaded.value > filled + per - 1,
    current: !!cur && cur.layer === layer && cur.position === position,
  }
}

function statusText(s: string) { return tStatus(s) }

function openPrint() {
  requireAuth(() => {
    const last = pallets.value[0]
    if (last) {
      // 预填最近一垛，避免人工重敲栈板号敲错
      printPreset.value = {
        palletCode: last.code,
        gradeCode: last.grade_code,
        bobbinCount: last.bobbin_count,
        layers: last.layers,
        totalWeight: last.total_weight || 0,
      }
    }
    showPrint.value = true
  })
}

async function doCreateOrder() {
  if (!newOrder.value.lot_id) { message.warning('请选择批次'); return }
  creating.value = true
  try {
    await http.post('/packing-orders', {
      lot_id: newOrder.value.lot_id,
      grade_code: newOrder.value.grade_code,
      target_quantity: newOrder.value.target_quantity,
    })
    message.success('订单已创建')
    showCreateOrder.value = false
    await load()
  } catch (e: any) {
    message.error(e?.response?.data?.message || '创建失败')
  } finally {
    creating.value = false
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

async function loadLayout() {
  try {
    const r: any = await http.get('/edge/config')
    const doc = r.data?.applied
    const pallets = doc?.pallets
    if (!pallets?.layouts?.length) { layout.value = null; return }
    const code = pallets.default || pallets.layouts[0].code
    const raw = pallets.layouts.find((l: any) => l.code === code) || pallets.layouts[0]

    // 配置里只有"每层几个位置"，容量与每层起始序号要现算。
    // 后端 BuildPalletLayout 在应用配置时也算过一次，但那是服务端结构，
    // 随 /edge/config 下发的是**配置原文**，不含推导结果。
    // 这里按同一套规则（层从下往上、每位置 bobbins_per_position 锭）复算。
    const per = raw.bobbins_per_position || 1
    const layers = (raw.layers || [])
      .slice()
      .sort((a: any, b: any) => a.layer - b.layer)
      .map((l: any) => ({ layer: l.layer, positions: l.positions }))
    const totalPositions = layers.reduce((s: number, l: any) => s + l.positions, 0)

    layout.value = {
      code: raw.code,
      bobbins_per_position: per,
      layers,
      capacity: totalPositions * per,
    }
  } catch {
    layout.value = null
  }
}

async function load() {
  try {
    const [oRes, pRes, lRes]: any[] = await Promise.all([
      http.get('/packing-orders?page=1&page_size=20'),
      http.get('/pallets?page=1&page_size=20'),
      http.get('/lots?status=active&page=1&page_size=100'),
    ])
    orders.value = oRes.data?.items || []
    pallets.value = pRes.data?.items || []
    lots.value = lRes.data?.items || []
  } catch {
    online.value = false
  }
  loadStatus()
}

async function loadPlc() {
  try {
    const r: any = await http.get('/edge/plc/values')
    plcDelayed.value = false
    const devices = (r.data?.devices || {}) as Record<string, Record<string, any>>
    for (const values of Object.values(devices)) {
      for (const [key, v] of Object.entries(values || {})) {
        // 打包机上报的已装锭数
        if (/BobbinCount|Bobbin_Count|Palletizer.*Count/i.test(key)) {
          const n = Number(v)
          if (Number.isFinite(n) && n >= 0) { plcCount.value = n; return }
        }
      }
    }
    plcCount.value = null
  } catch {
    plcDelayed.value = true
    plcCount.value = null
  }
}

onMounted(() => {
  const tick = () => { clock.value = new Date().toLocaleTimeString('zh-CN', { hour12: false }) }
  tick()
  load()
  loadLayout()
  loadPlc()
  timer = setInterval(() => { tick(); load() }, 5000)
  plcTimer = setInterval(loadPlc, 1000)
})
onUnmounted(() => { clearInterval(timer); clearInterval(plcTimer) })
</script>

<style scoped>
.station { height:100vh; background:var(--paper); padding:16px; display:flex; flex-direction:column; overflow:hidden; }
.station-header { display:flex; justify-content:space-between; align-items:center; margin-bottom:12px; flex-shrink:0; }
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
.layout-tag { margin-left:8px; font-family:var(--font-mono); font-size:12px; font-weight:400; color:var(--steel-light); }

.grid-panel { flex:1.6; display:flex; flex-direction:column; min-height:0; }
.progress-line { margin-bottom:16px; font-size:17px; color:var(--graphite); }
.cell-hint { color:var(--igh-red); font-weight:600; }
.big { font-family:var(--font-display); font-size:38px; font-weight:600; color:var(--graphite); font-variant-numeric:tabular-nums; }

.layers { flex:1; display:flex; flex-direction:column; gap:10px; justify-content:center; overflow-y:auto; }
.layer { display:flex; align-items:center; gap:12px; }
.layer-label { width:36px; font-family:var(--font-mono); font-size:13px; color:var(--steel-light); text-align:right; }
.cells { display:flex; gap:8px; flex-wrap:wrap; }

/* 格子比进度条有用：它回答"下一个装哪" */
.cell {
  width:64px; height:64px; border-radius:2px;
  border:2px solid var(--line); background:var(--plate);
  transition:background .2s, border-color .2s, transform .2s;
}
/* 已装：石墨实心块，远距离一眼能数 */
.cell.filled { background:var(--graphite); border-color:var(--graphite); }
/* 当前位：品牌红闪烁，是"下一步动作"的唯一提示 */
.cell.current {
  background:var(--igh-red); border-color:var(--igh-red);
  animation:pulse 1.2s ease-in-out infinite;
}
@keyframes pulse {
  0%, 100% { transform:scale(1); }
  50% { transform:scale(1.08); }
}

.side-panel { flex:1; display:flex; flex-direction:column; min-height:0; }
.order-progress { margin-bottom:12px; }
.kv { display:flex; justify-content:space-between; padding:8px 0; border-bottom:1px solid var(--line-soft); font-size:17px; color:var(--graphite); }
.plc-count { margin-bottom:16px; font-family:var(--font-display); font-size:44px; font-weight:600; color:var(--graphite); font-variant-numeric:tabular-nums; }
.plc-count.warn { font-size:19px; color:var(--warn); }

/* 主操作：112px 高，戴手套也能稳按 */
.btn-primary {
  height:112px; width:100%; margin-top:auto; border-radius:2px; border:none; cursor:pointer;
  background:var(--igh-red); color:#fff; font-size:28px; font-weight:600;
  transition: transform .06s, background .15s;
}
.btn-primary:hover:not(:disabled) { background:var(--igh-red-hover); }
.btn-primary:active:not(:disabled) { transform:translateY(2px); background:var(--igh-red-deep); }
.btn-primary:disabled { background:var(--line); color:var(--steel-light); cursor:not-allowed; }

.list-panel { flex:0 0 180px; display:flex; flex-direction:column; min-height:0; }
.list { flex:1; overflow-y:auto; }
.list-item { display:flex; gap:20px; align-items:center; padding:8px; border-bottom:1px solid var(--line-soft);
  font-size:16px; color:var(--graphite); }
.mono { font-family:var(--font-mono); }
.muted { color:var(--steel-light); font-size:14px; }
.empty-hint { padding:12px 0; font-size:15px; color:var(--steel-light); }
</style>
