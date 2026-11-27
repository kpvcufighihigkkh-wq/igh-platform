<template>
  <div class="station">
    <header class="station-header">
      <div class="nameplate">
        <span class="nameplate-bar" aria-hidden="true"></span>
        <div class="nameplate-text">
          <h1>外检工位</h1>
          <span class="nameplate-sub">SORTING · 外检区域</span>
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
      <!-- 扫码框是第一元素并自动聚焦：工人上手就是扫，不需要先找输入框 -->
      <section class="panel scan-panel">
        <input
          ref="scanInput"
          v-model="scanCode"
          class="scan-input"
          placeholder="扫描丝锭标签"
          autocomplete="off"
          @keyup.enter="lookup"
        />
        <div class="scan-status" :class="scanState">
          {{ scanMessage }}
        </div>
      </section>

      <div class="row grow">
        <section class="panel detail-panel">
          <h2>丝锭</h2>
          <template v-if="bobbin">
            <div class="bobbin-code">{{ bobbin.code }}</div>
            <div class="kv">
              <span>锭位</span><span class="mono">{{ bobbin.place_in_winder }}</span>
            </div>
            <div class="kv">
              <span>重量</span>
              <span class="mono weight">{{ weightText }}</span>
            </div>
            <div class="kv">
              <span>已判等级</span><span class="mono">{{ bobbin.sorting_grade || '—' }}</span>
            </div>
            <div class="kv">
              <span>最终等级</span><span class="mono">{{ bobbin.final_grade || '—' }}</span>
            </div>
            <div class="kv">
              <span>缺陷</span><span class="mono">{{ bobbin.defect_code || '—' }}</span>
            </div>
          </template>
          <div v-else class="empty-hint">扫码后显示丝锭信息</div>
        </section>

        <section class="panel action-panel">
          <h2>判级</h2>
          <!-- 大按钮组而不是下拉：下拉在触摸屏上要点开、滚动、再点，还容易点错相邻项 -->
          <div class="grade-grid">
            <button
              v-for="g in grades"
              :key="g.value"
              class="btn-grade"
              :class="{ active: grade === g.value }"
              @click="grade = g.value"
            >
              {{ g.value }}
            </button>
          </div>

          <h2 style="margin-top:16px">缺陷（可多选）</h2>
          <div class="defect-grid">
            <button
              v-for="d in defects"
              :key="d.code"
              class="btn-defect"
              :class="{ active: selectedDefects.includes(d.code) }"
              @click="toggleDefect(d.code)"
            >
              {{ d.name || d.code }}
            </button>
            <div v-if="!defects.length" class="empty-hint">未配置缺陷代码</div>
          </div>

          <button
            class="btn-primary"
            :disabled="!bobbin || !grade || busy"
            @click="submit"
          >
            {{ busy ? '提交中…' : '提交判级' }}
          </button>
        </section>
      </div>

      <section class="panel dist-panel">
        <h2>等级统计</h2>
        <div class="dist-row">
          <div v-for="d in dist" :key="d.final_grade" class="dist-item">
            <span class="dist-grade">{{ d.final_grade }}</span>
            <span class="dist-count">{{ d.count }}</span>
          </div>
          <div v-if="!dist.length" class="empty-hint">暂无数据</div>
        </div>
      </section>
    </div>

    <transition name="toast">
      <div v-if="toast" class="toast">{{ toast }}</div>
    </transition>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import http from '../../api/http'

const clock = ref('')
const online = ref(true)
const queueDepth = ref(0)

const scanInput = ref<HTMLInputElement>()
const scanCode = ref('')
const scanState = ref<'idle' | 'ok' | 'bad'>('idle')
const scanMessage = ref('等待扫码')
const bobbin = ref<any>(null)

const grades = [
  { value: 'AAA' }, { value: 'AA' }, { value: 'AA1' },
  { value: 'AA2' }, { value: 'A1' }, { value: 'A' },
]
const grade = ref('')
const defects = ref<any[]>([])
const selectedDefects = ref<string[]>([])
const dist = ref<any[]>([])
const busy = ref(false)
const toast = ref('')

// 重量来自 PLC 模拟量：只读展示，不提供手输入口。
// 点表里字段名不固定，按语义匹配，并区分"未读到"与"读到 0"。
const plcWeight = ref<number | null>(null)
const plcDelayed = ref(false)

const weightText = computed(() => {
  if (plcDelayed.value) return 'PLC 未连接'
  if (plcWeight.value === null) return '—'
  return `${plcWeight.value.toFixed(3)} kg`
})

let timer: any
let plcTimer: any

function showToast(msg: string) {
  toast.value = msg
  setTimeout(() => { if (toast.value === msg) toast.value = '' }, 3000)
}

function toggleDefect(code: string) {
  const i = selectedDefects.value.indexOf(code)
  if (i >= 0) selectedDefects.value.splice(i, 1)
  else selectedDefects.value.push(code)
}

async function lookup() {
  const code = scanCode.value.trim()
  if (!code) return
  try {
    const r: any = await http.get(`/bobbins/by-code/${encodeURIComponent(code)}`)
    bobbin.value = r.data
    scanState.value = 'ok'
    scanMessage.value = `已定位：${r.data.code}`
    grade.value = ''
    selectedDefects.value = []
  } catch (e: any) {
    bobbin.value = null
    scanState.value = 'bad'
    // 扫到不存在的码是常态（扫错、别线的锭），用明确措辞而不是"系统错误"
    scanMessage.value = e?.response?.status === 404 ? '查无此丝锭，请确认标签' : '查询失败'
  } finally {
    scanCode.value = ''
    scanInput.value?.focus()
  }
}

async function submit() {
  if (!bobbin.value || !grade.value) return
  busy.value = true
  try {
    await http.post('/quality/sorting', {
      bobbin_code: bobbin.value.code,
      grade: grade.value,
      // 多选缺陷按逗号拼接：后端字段是单值字符串，保持既有数据格式
      defect_code: selectedDefects.value.join(','),
    })
    showToast(`${bobbin.value.code} 已判 ${grade.value}`)
    bobbin.value = null
    grade.value = ''
    selectedDefects.value = []
    scanState.value = 'idle'
    scanMessage.value = '等待扫码'
    scanInput.value?.focus()
    load()
  } catch (e: any) {
    // 判级是不可撤销的业务动作，失败必须明确提示，不能只弹一句"操作失败"
    showToast(e?.response?.data?.message || '提交失败')
  } finally {
    busy.value = false
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
    const [bRes, dRes, defRes]: any[] = await Promise.all([
      http.get('/bobbins?page=1&page_size=30'),
      http.get('/quality/distribution'),
      http.get('/defects'),
    ])
    dist.value = dRes.data?.items || dRes.data || []
    defects.value = defRes.data?.items || defRes.data || []
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
        if (/Weight|Gross/i.test(key)) {
          const n = Number(v)
          if (Number.isFinite(n)) { plcWeight.value = n; return }
        }
      }
    }
    plcWeight.value = null
  } catch {
    plcDelayed.value = true
    plcWeight.value = null
  }
}

onMounted(() => {
  const tick = () => { clock.value = new Date().toLocaleTimeString('zh-CN', { hour12: false }) }
  tick()
  load()
  loadPlc()
  scanInput.value?.focus()
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

.panel { background:var(--plate); border:1px solid var(--line); border-radius:2px; padding:16px; }
.panel h2 { margin:0 0 12px; padding-bottom:8px; border-bottom:1px solid var(--line-soft); font-size:13px; font-weight:600; letter-spacing:.04em; color:var(--steel); }

.scan-panel { display:flex; align-items:center; gap:16px; flex-shrink:0; }
.scan-input {
  flex:1; height:72px; border-radius:2px; border:2px solid var(--line-strong, #c4c0b8); background:var(--plate);
  color:var(--graphite); font-size:26px; font-family:var(--font-mono); padding:0 16px; outline:none;
}
.scan-input:focus { border-color:var(--igh-red); box-shadow:0 0 0 3px rgba(224,33,24,.12); }
.scan-status { min-width:280px; font-size:18px; color:var(--steel); }
.scan-status.ok { color:var(--ok); font-weight:600; }
.scan-status.bad { color:var(--alarm); font-weight:600; }

.row { display:flex; gap:12px; }
.row.grow { flex:1; min-height:0; }

.detail-panel { flex:1; overflow-y:auto; }
.bobbin-code { margin-bottom:12px; font-family:var(--font-mono); font-size:24px; font-weight:700; color:var(--graphite); word-break:break-all; }
.kv { display:flex; justify-content:space-between; padding:8px 0; border-bottom:1px solid var(--line-soft);
  font-size:17px; color:var(--graphite); }
.kv .weight { font-size:22px; font-weight:600; color:var(--graphite); }
.mono { font-family:var(--font-mono); }

.action-panel { flex:1.4; display:flex; flex-direction:column; min-height:0; }

/* 单行 6 列：两行会吃掉 74px 纵向空间，在 1280x720 的工控屏上
   足以把缺陷区压成 0 高度，导致缺陷按钮被提交按钮盖住、点不到。 */
.grade-grid { display:grid; grid-template-columns:repeat(6, 1fr); gap:10px; }
.btn-grade {
  height:72px; border-radius:2px; border:2px solid var(--line); background:var(--plate);
  color:var(--graphite); font-family:var(--font-display); font-size:24px; font-weight:600; cursor:pointer;
  transition:transform .06s, border-color .15s, background .15s;
}
.btn-grade:hover { border-color:var(--steel-light); }
/* 选中用品牌红：判级是这一页唯一的主决策 */
.btn-grade.active { border-color:var(--igh-red); background:var(--igh-red-tint); color:var(--igh-red); }
.btn-grade:active { transform:translateY(2px); }

/* min-height 保证缺陷区永远有可点的空间；空间不足时它自己内部滚动，
   而不是被后面的主按钮挤掉——那会让按钮"消失"或被遮挡。 */
.defect-grid { display:grid; grid-template-columns:repeat(4, 1fr); gap:10px;
  flex:1; min-height:74px; overflow-y:auto; align-content:start; }
.btn-defect {
  height:64px; border-radius:2px; border:2px solid var(--line); background:var(--plate);
  color:var(--graphite); font-size:18px; cursor:pointer;
  transition:transform .06s, border-color .15s, background .15s;
}
.btn-defect:hover { border-color:var(--steel-light); }
/* 缺陷用琥珀色，与"品牌红 = 主操作 / 告警红 = 异常"区分开 */
.btn-defect.active { border-color:var(--warn); background:var(--warn-tint); color:var(--warn); font-weight:600; }
.btn-defect:active { transform:translateY(2px); }

/* 主操作：112px 高，戴手套也能稳按 */
.btn-primary {
  height:112px; width:100%; margin-top:12px; border-radius:2px; border:none; cursor:pointer;
  background:var(--igh-red); color:#fff; font-size:28px; font-weight:600;
  transition: transform .06s, background .15s; flex-shrink:0;
}
.btn-primary:hover:not(:disabled) { background:var(--igh-red-hover); }
.btn-primary:active:not(:disabled) { transform:translateY(2px); background:var(--igh-red-deep); }
.btn-primary:disabled { background:var(--line); color:var(--steel-light); cursor:not-allowed; }

.dist-panel { flex-shrink:0; }
.dist-row { display:flex; gap:28px; flex-wrap:wrap; }
.dist-item { display:flex; align-items:baseline; gap:8px; }
.dist-grade { font-size:17px; color:var(--steel); }
.dist-count { font-family:var(--font-display); font-size:32px; font-weight:600; color:var(--graphite); font-variant-numeric:tabular-nums; }

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
