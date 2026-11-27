<template>
  <div class="warehouse-3d">
    <div class="iso-toolbar">
      <button class="iso-btn" @click="zoomIn" title="放大">+</button>
      <span class="zoom-pct">{{ zoomPct }}%</span>
      <button class="iso-btn" @click="zoomOut" title="缩小">−</button>
      <button class="iso-btn" @click="resetView" title="重置视图">↺</button>
    </div>
    <div
      ref="vpRef"
      class="iso-viewport"
      @wheel.prevent="onWheel"
      @mousedown="onDragStart"
    >
      <svg
        :viewBox="viewBox"
        class="iso-svg"
        preserveAspectRatio="xMidYMid meet"
        :style="svgStyle"
      >
        <g :transform="`translate(${ox}, ${oy})`">
          <polygon :points="floorPts" class="floor" />
          <text
            v-for="l in layers" :key="'lb'+l"
            :x="iso(0, rows, l).x - 8"
            :y="labelY(l)"
            text-anchor="end" class="layer-label"
          >L{{ l }}</text>
          <template v-for="r in rows" :key="'R'+r">
            <template v-for="l in layers" :key="'L'+l">
              <template v-for="c in cols" :key="'C'+c">
                <polygon v-if="r === rows" :points="fFace(r,c,l)" :class="['ff', 'st-'+gs(r,c,l)]" />
                <polygon v-if="c === cols" :points="rFace(r,c,l)" :class="['rf', 'st-'+gs(r,c,l)]" />
                <polygon
                  :points="tFace(r,c,l)"
                  :class="['tf', 'st-'+gs(r,c,l)]"
                  @click="onClick(r,c,l)"
                >
                  <title>行{{ r }} 列{{ c }} 层{{ l }} · {{ stText(gs(r,c,l)) }}</title>
                </polygon>
              </template>
            </template>
          </template>
        </g>
      </svg>
    </div>
    <div class="legend">
      <span class="leg"><span class="ld ld-empty"></span>空闲</span>
      <span class="leg"><span class="ld ld-occupied"></span>占用</span>
      <span class="leg"><span class="ld ld-reserved"></span>预留</span>
      <span class="leg"><span class="ld ld-disabled"></span>禁用</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'

interface Slot {
  id: number
  row_no: number
  col_no: number
  layer_no: number
  status: string
  occupant_type?: string
  occupant_id?: number
}

const props = defineProps<{
  rows: number
  cols: number
  layers: number
  slots: Slot[]
}>()

const emit = defineEmits<{ slotClick: [slot: Slot] }>()

const maxSvgH = 560
const md = computed(() => Math.max(props.rows + props.cols, 4))

const cw = computed(() => {
  const fitH = 2 * (maxSvgH - 24 - 8 * props.layers) / (md.value * (1 + props.layers))
  return Math.max(8, Math.min(22, fitH))
})
const ch = computed(() => cw.value / 2)
const lh = computed(() => md.value * ch.value + 8)
const edge = computed(() => Math.min(1, cw.value / lh.value))

function iso(c: number, r: number, l: number) {
  return { x: (c - r) * cw.value, y: (c + r) * ch.value - l * lh.value }
}

function pts(...coords: { x: number; y: number }[]) {
  return coords.map(p => `${p.x},${p.y}`).join(' ')
}

const contentW = computed(() => (props.rows + props.cols) * cw.value + 72)
const svgH = computed(() => (props.rows + props.cols) * ch.value + props.layers * lh.value + 24)
const svgW = computed(() => Math.max(contentW.value, svgH.value * 1.4))
const ox = computed(() => (svgW.value - contentW.value) / 2 + props.rows * cw.value + 36)
const oy = computed(() => props.layers * lh.value + 10)
const viewBox = computed(() => `0 0 ${svgW.value} ${svgH.value}`)

const floorPts = computed(() =>
  pts(iso(0, 0, 0), iso(props.cols, 0, 0), iso(props.cols, props.rows, 0), iso(0, props.rows, 0))
)

function labelY(l: number) {
  const top = iso(0, props.rows, l).y
  const bot = iso(0, props.rows, l - edge.value).y
  return (top + bot) / 2 + 4
}

const slotMap = computed(() => {
  const m = new Map<string, Slot>()
  for (const s of props.slots) m.set(`${s.row_no}-${s.col_no}-${s.layer_no}`, s)
  return m
})

function gs(r: number, c: number, l: number) {
  return slotMap.value.get(`${r}-${c}-${l}`)?.status || 'empty'
}

function tFace(r: number, c: number, l: number) {
  return pts(iso(c - 1, r - 1, l), iso(c, r - 1, l), iso(c, r, l), iso(c - 1, r, l))
}

function fFace(r: number, c: number, l: number) {
  const bl = l - edge.value
  return pts(iso(c - 1, r, l), iso(c, r, l), iso(c, r, bl), iso(c - 1, r, bl))
}

function rFace(r: number, c: number, l: number) {
  const bl = l - edge.value
  return pts(iso(c, r - 1, l), iso(c, r, l), iso(c, r, bl), iso(c, r - 1, bl))
}

function onClick(r: number, c: number, l: number) {
  if (dragged) return
  const s = slotMap.value.get(`${r}-${c}-${l}`)
  if (s) emit('slotClick', s)
}

const stMap: Record<string, string> = { empty: '空闲', occupied: '占用', reserved: '预留', disabled: '禁用' }
function stText(s: string) { return stMap[s] || s }

// --- zoom & pan ---
const vpRef = ref<HTMLElement>()
const zoom = ref(1)
const panX = ref(0)
const panY = ref(0)
let dragged = false

const zoomPct = computed(() => Math.round(zoom.value * 100))
const svgStyle = computed(() => ({
  transform: `translate(${panX.value}px,${panY.value}px) scale(${zoom.value})`,
  transformOrigin: '0 0',
}))

function clampZoom(v: number) { return Math.min(4, Math.max(0.4, v)) }

function zoomIn() { applyZoom(zoom.value * 1.25) }
function zoomOut() { applyZoom(zoom.value / 1.25) }
function resetView() { zoom.value = 1; panX.value = 0; panY.value = 0 }

function applyZoom(next: number, cx?: number, cy?: number) {
  next = clampZoom(next)
  if (cx !== undefined && cy !== undefined) {
    const r = next / zoom.value
    panX.value = cx - r * (cx - panX.value)
    panY.value = cy - r * (cy - panY.value)
  }
  zoom.value = next
}

function onWheel(e: WheelEvent) {
  const rect = vpRef.value!.getBoundingClientRect()
  const cx = e.clientX - rect.left
  const cy = e.clientY - rect.top
  const factor = e.deltaY < 0 ? 1.15 : 1 / 1.15
  applyZoom(zoom.value * factor, cx, cy)
}

function onDragStart(e: MouseEvent) {
  if (e.button !== 0) return
  dragged = false
  const sx = e.clientX, sy = e.clientY
  const spx = panX.value, spy = panY.value

  const onMove = (ev: MouseEvent) => {
    const dx = ev.clientX - sx, dy = ev.clientY - sy
    if (Math.abs(dx) > 3 || Math.abs(dy) > 3) dragged = true
    panX.value = spx + dx
    panY.value = spy + dy
  }
  const onUp = () => {
    document.removeEventListener('mousemove', onMove)
    document.removeEventListener('mouseup', onUp)
  }
  document.addEventListener('mousemove', onMove)
  document.addEventListener('mouseup', onUp)
}
</script>

<style scoped>
.warehouse-3d { border-radius: 2px; padding: 16px; position: relative; }

.iso-toolbar {
  display: flex; align-items: center; gap: 6px; margin-bottom: 8px; user-select: none;
}
.iso-btn {
  width: 28px; height: 28px; border-radius: 2px; border: 1px solid var(--line);
  background: var(--plate); color: var(--graphite); font-size: 16px; cursor: pointer;
  display: flex; align-items: center; justify-content: center; transition: border-color .15s, color .15s;
}
.iso-btn:hover { border-color: var(--igh-red); color: var(--igh-red); }
.zoom-pct { font-size: 12px; color: var(--steel); font-family: var(--font-mono); min-width: 40px; text-align: center; }

.iso-viewport { overflow: hidden; cursor: grab; max-height: 520px; }
.iso-viewport:active { cursor: grabbing; }
.iso-svg { width: 100%; max-height: 520px; display: block; }

.floor { fill: none; stroke: var(--line); stroke-width: 0.8; stroke-dasharray: 3,2; }
.layer-label { fill: var(--steel-light); font-size: 10px; font-family: var(--font-mono); }

/* 库位三态用同一套语义色：空闲=白、占用=品牌红、预留=琥珀、禁用=灰。
   侧面用同色加深做出等轴测的体块感，正面最浅、顶面最亮。 */
.tf { stroke: var(--line); stroke-width: 0.5; cursor: pointer; transition: opacity 0.15s; }
.tf:hover { opacity: 0.75; stroke: var(--graphite); stroke-width: 1; }
.tf.st-empty    { fill: #ffffff; }
.tf.st-occupied { fill: #e02118; }
.tf.st-reserved { fill: #c98a2e; }
.tf.st-disabled { fill: #d9d6cf; }

.ff { stroke: #c4c0b8; stroke-width: 0.3; }
.ff.st-empty    { fill: #f3f2ee; }
.ff.st-occupied { fill: #b2160f; }
.ff.st-reserved { fill: #9a6a1f; }
.ff.st-disabled { fill: #c9c5bd; }

.rf { stroke: #c4c0b8; stroke-width: 0.3; }
.rf.st-empty    { fill: #e6e3dc; }
.rf.st-occupied { fill: #8f110c; }
.rf.st-reserved { fill: #7d5518; }
.rf.st-disabled { fill: #b9b5ac; }

.legend { display: flex; gap: 16px; margin-top: 12px; font-size: 13px; color: var(--steel); }
.leg { display: flex; align-items: center; gap: 4px; }
.ld { width: 12px; height: 12px; border-radius: 1px; }
.ld-empty    { background: #ffffff; border: 1px solid var(--line); }
.ld-occupied { background: #e02118; }
.ld-reserved { background: #c98a2e; }
.ld-disabled { background: #d9d6cf; }
</style>
