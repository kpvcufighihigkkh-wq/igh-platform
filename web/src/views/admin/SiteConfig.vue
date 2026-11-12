<template>
  <div class="site-config">
    <!-- 左：边端节点 -->
    <div class="node-list">
      <div class="list-title">边端节点</div>
      <div v-for="n in nodes" :key="n.code" class="node-item"
           :class="{ active: selected?.code === n.code }" @click="selectNode(n)">
        <div class="node-name">{{ n.name || n.code }}</div>
        <div class="node-meta">
          <span>{{ n.code }}</span>
          <n-tag size="tiny" :type="n.status === 'online' ? 'success' : 'default'" :bordered="false">
            {{ n.status === 'online' ? '在线' : '离线' }}
          </n-tag>
        </div>
      </div>
      <n-empty v-if="!nodes.length" description="暂无边端节点" size="small" style="margin-top:32px" />
    </div>

    <!-- 右：配置编辑 -->
    <div class="editor" v-if="selected">
      <n-card size="small">
        <template #header>
          <n-space justify="space-between" align="center" style="width:100%">
            <span>
              现场配置 · {{ selected.name || selected.code }}
              <n-tag v-if="doc.version" size="small" :bordered="false" style="margin-left:8px">
                版本 {{ doc.version }}
              </n-tag>
            </span>
            <n-space :size="8">
              <n-button size="small" @click="validateOnly" :loading="validating">仅校验</n-button>
              <n-button size="small" type="primary" @click="save" :loading="saving">保存并下发</n-button>
            </n-space>
          </n-space>
        </template>

        <n-alert v-if="problems.length" type="error" title="配置校验未通过" style="margin-bottom:12px">
          <ul class="problems">
            <li v-for="(p, i) in problems" :key="i">{{ p }}</li>
          </ul>
        </n-alert>
        <n-alert v-else-if="okTip" type="success" :bordered="false" style="margin-bottom:12px">
          {{ okTip }}
        </n-alert>

        <n-alert type="info" :bordered="false" style="margin-bottom:12px">
          保存后配置会随心跳下发到边端，但<b>不会自动生效</b>：需现场人员在边端页面确认应用。
          产线运行中切换点表会改变读取地址，必须人工确认。
        </n-alert>

        <n-form-item label="修改说明" label-placement="left" label-width="76">
          <n-input v-model:value="doc.remark" placeholder="例如：07 线新增 3 号机位 / 修正装载表第 5 行" />
        </n-form-item>

        <n-tabs type="line" v-model:value="tab">
          <!-- ============ PLC 点表 ============ -->
          <n-tab-pane name="plc" :tab="`PLC 点表 (${doc.plc?.length || 0})`">
            <n-space style="margin-bottom:12px">
              <n-button size="small" @click="addDevice">添加设备</n-button>
              <n-button size="small" @click="addDeviceFrom(selected.plc_device)" v-if="selected.plc_device">
                按线路绑定初始化
              </n-button>
            </n-space>

            <div class="device-list">
              <div v-for="(d, di) in doc.plc" :key="di" class="device-card">
                <div class="device-head">
                  <span class="device-code">{{ d.code || '(未命名)' }}</span>
                  <span class="device-host">{{ d.host }}:{{ d.port }} · DB{{ d.poll_groups?.[0]?.db ?? '-' }}</span>
                  <n-space :size="4">
                    <n-button text type="info" size="small" @click="openPoints(di)">编辑点表</n-button>
                    <n-button text type="error" size="small" @click="doc.plc.splice(di, 1)">删除</n-button>
                  </n-space>
                </div>
                <n-grid :cols="6" :x-gap="10" style="margin-top:10px">
                  <n-gi><n-form-item label="编号" size="small" :show-label="true">
                    <n-input v-model:value="d.code" size="small" /></n-form-item></n-gi>
                  <n-gi><n-form-item label="IP" size="small" :show-label="true">
                    <n-input v-model:value="d.host" size="small" /></n-form-item></n-gi>
                  <n-gi><n-form-item label="端口" size="small" :show-label="true">
                    <n-input-number v-model:value="d.port" size="small" :min="1" style="width:100%" /></n-form-item></n-gi>
                  <n-gi><n-form-item label="Rack" size="small" :show-label="true">
                    <n-input-number v-model:value="d.rack" size="small" :min="0" style="width:100%" /></n-form-item></n-gi>
                  <n-gi><n-form-item label="Slot" size="small" :show-label="true">
                    <n-input-number v-model:value="d.slot" size="small" :min="0" style="width:100%" /></n-form-item></n-gi>
                  <n-gi><n-form-item label="型号" size="small" :show-label="true">
                    <n-input v-model:value="d.model" size="small" /></n-form-item></n-gi>
                </n-grid>
              </div>
              <n-empty v-if="!doc.plc?.length" description="尚未配置 PLC 点表" size="small" style="margin:24px 0" />
            </div>
          </n-tab-pane>

          <!-- ============ 装载位置查表 ============ -->
          <n-tab-pane name="loading" tab="装载位置查表">
            <n-space align="center" style="margin-bottom:12px">
              <span class="hint">容器容量</span>
              <n-input-number v-model:value="doc.loading.capacity" size="small" :min="1" style="width:110px" />
              <n-button size="small" @click="addPattern">添加模式</n-button>
              <span class="hint">
                语义：卷绕机锭位 + 落筒侧(A/B) → 容器位置。每个模式的 A、B 两列合起来必须是 1..容量 的完整排列。
              </span>
            </n-space>

            <n-collapse v-if="doc.loading?.patterns?.length">
              <n-collapse-item v-for="(p, pi) in doc.loading.patterns" :key="pi"
                               :name="String(pi)" :title="p.code || `模式 ${pi + 1}`">
                <template #header-extra>
                  <n-space :size="4">
                    <n-tag size="tiny" :bordered="false">{{ p.map?.length || 0 }} 行</n-tag>
                    <n-button text type="error" size="small" @click.stop="doc.loading.patterns.splice(pi, 1)">删除</n-button>
                  </n-space>
                </template>
                <n-space align="center" style="margin-bottom:8px">
                  <span class="hint">模式代码</span>
                  <n-input v-model:value="p.code" size="small" style="width:120px" placeholder="dx/dxw/sx/sxw/fdy" />
                  <n-button size="small" @click="addLoadingRow(p)">添加行</n-button>
                </n-space>
                <n-data-table :columns="loadingCols" :data="p.map || []" size="small" :bordered="false" max-height="360" />
              </n-collapse-item>
            </n-collapse>
            <n-empty v-else description="尚未配置装载位置查表" size="small" style="margin:24px 0" />
          </n-tab-pane>

          <!-- ============ 外检锭位映射 ============ -->
          <n-tab-pane name="heads" tab="外检锭位映射">
            <n-space style="margin-bottom:12px">
              <n-button size="small" @click="addHeadMap">添加头数配置</n-button>
              <span class="hint">语义：打标头 + 段序 → 模块内锭位；position 必须是 1..容量 的完整排列。</span>
            </n-space>

            <div v-for="(m, mi) in doc.heads.head_maps" :key="mi" class="device-card">
              <div class="device-head">
                <n-space align="center" :size="8">
                  <span class="hint">头数</span>
                  <n-input-number v-model:value="m.head_count" size="small" :min="1" style="width:90px" />
                  <span class="hint">每头段数</span>
                  <n-input-number v-model:value="m.steps_per_head" size="small" :min="1" style="width:90px" />
                  <span class="hint">容量</span>
                  <n-input-number v-model:value="m.capacity" size="small" :min="1" style="width:90px" />
                </n-space>
                <n-space :size="4">
                  <n-button text size="small" @click="fillHeadMap(m)">按头数生成空表</n-button>
                  <n-button text type="error" size="small" @click="doc.heads.head_maps.splice(mi, 1)">删除</n-button>
                </n-space>
              </div>
              <n-data-table :columns="headCols" :data="m.entries || []" size="small" :bordered="false" max-height="320" style="margin-top:8px" />
            </div>
            <n-empty v-if="!doc.heads?.head_maps?.length" description="尚未配置外检锭位映射" size="small" style="margin:24px 0" />
          </n-tab-pane>
        </n-tabs>
      </n-card>
    </div>
    <div class="editor" v-else>
      <n-empty description="请选择左侧边端节点" style="margin-top:120px" />
    </div>

    <!-- 点表编辑抽屉 -->
    <n-drawer v-model:show="showPoints" :width="880" placement="right">
      <n-drawer-content :title="`点表 · ${editDevice?.code || ''}`" closable>
        <template #header>
          <n-space align="center">
            <span>点表 · {{ editDevice?.code }}</span>
            <n-button size="tiny" @click="addPollGroup">添加分组</n-button>
          </n-space>
        </template>

        <div v-for="(g, gi) in editDevice?.poll_groups" :key="gi" class="device-card">
          <div class="device-head">
            <n-space align="center" :size="10">
              <span class="hint">分组名</span>
              <n-input v-model:value="g.name" size="small" style="width:130px" />
              <span class="hint">DB</span>
              <n-input-number v-model:value="g.db" size="small" :min="0" style="width:80px" />
              <span class="hint">起始</span>
              <n-input-number v-model:value="g.start" size="small" :min="0" style="width:80px" />
              <span class="hint">长度</span>
              <n-input-number v-model:value="g.length" size="small" :min="0" style="width:90px" />
            </n-space>
            <n-button text type="error" size="small" @click="editDevice.poll_groups.splice(gi, 1)">删除分组</n-button>
          </div>

          <n-grid :cols="6" :x-gap="10" style="margin-top:8px">
            <n-gi><n-form-item label="机位数" size="small">
              <n-input-number v-model:value="g.repeat.count" size="small" :min="0" style="width:100%" /></n-form-item></n-gi>
            <n-gi><n-form-item label="步长" size="small">
              <n-input-number v-model:value="g.repeat.stride" size="small" :min="0" style="width:100%" /></n-form-item></n-gi>
            <n-gi><n-form-item label="左侧偏移" size="small">
              <n-input-number v-model:value="g.repeat.sides.left.offset" size="small" :min="0" style="width:100%" /></n-form-item></n-gi>
            <n-gi><n-form-item label="右侧偏移" size="small">
              <n-input-number v-model:value="g.repeat.sides.right.offset" size="small" :min="0" style="width:100%" /></n-form-item></n-gi>
            <n-gi><n-form-item label="单面锭位" size="small">
              <n-switch v-model:value="g._single" size="small" /></n-form-item></n-gi>
            <n-gi><n-form-item label="轮询(ms)" size="small">
              <n-input-number v-model:value="g.interval_ms" size="small" :min="100" :step="100" style="width:100%" /></n-form-item></n-gi>
          </n-grid>

          <n-space style="margin:6px 0">
            <n-button size="tiny" @click="addPoint(g)">添加字段</n-button>
          </n-space>
          <n-data-table :columns="pointCols(g)" :data="g.points || []" size="small" :bordered="false" max-height="300" />
        </div>

        <template #footer>
          <n-space justify="end">
            <n-button @click="showPoints = false">关闭</n-button>
          </n-space>
        </template>
      </n-drawer-content>
    </n-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, h } from 'vue'
import { NInput, NInputNumber, NSelect, NButton, useMessage } from 'naive-ui'
import http from '../../api/http'

const msg = useMessage()

const nodes = ref<any[]>([])
const selected = ref<any>(null)
const tab = ref('plc')
const saving = ref(false)
const validating = ref(false)
const problems = ref<string[]>([])
const okTip = ref('')

const showPoints = ref(false)
const editDevice = ref<any>(null)

// 与后端 siteconfig.Document 一一对应
const doc = reactive<any>({
  version: 0,
  remark: '',
  plc: [],
  loading: { capacity: 24, patterns: [] },
  heads: { head_maps: [] },
})

onMounted(loadNodes)

async function loadNodes() {
  try {
    const res: any = await http.get('/edge-nodes')
    nodes.value = res.data || []
    if (nodes.value.length && !selected.value) selectNode(nodes.value[0])
  } catch (e: any) {
    msg.error('加载边端节点失败：' + (e?.message || e))
  }
}

async function selectNode(n: any) {
  selected.value = n
  problems.value = []
  okTip.value = ''
  try {
    const res: any = await http.get(`/edge-nodes/${n.code}/site-config`)
    const d = res.data
    if (d) {
      doc.version = d.version || 0
      doc.remark = d.remark || ''
      doc.plc = (d.plc || []).map(normalizeDevice)
      doc.loading = normalizeLoading(d.loading)
      doc.heads = { head_maps: (d.heads?.head_maps || []).map((m: any) => ({ ...m, entries: m.entries || [] })) }
    } else {
      doc.version = 0
      doc.remark = ''
      doc.plc = []
      doc.loading = { capacity: 24, patterns: [] }
      doc.heads = { head_maps: [] }
    }
  } catch (e: any) {
    msg.error('加载现场配置失败：' + (e?.message || e))
  }
}

// 后端 JSON 里 repeat 可能缺失，补成可编辑结构
function normalizeDevice(d: any) {
  const dev = { ...d }
  dev.poll_groups = (dev.poll_groups || []).map((g: any) => {
    const repeat = g.repeat || { count: 0, stride: 0, sides: null }
    const sides = repeat.sides || { left: { offset: 0 }, right: { offset: 0 } }
    return {
      ...g,
      interval_ms: g.interval_ms || 500,
      start: g.start || 0,
      repeat: {
        count: repeat.count || 0,
        stride: repeat.stride || 0,
        sides,
      },
      points: g.points || [],
      // 单面锭位：sides 为 null
      _single: repeat.sides == null,
    }
  })
  return dev
}

function normalizeLoading(l: any) {
  if (!l) return { capacity: 24, patterns: [] }
  return {
    capacity: l.capacity || 24,
    patterns: (l.patterns || []).map((p: any) => ({ code: p.code, map: p.map || [] })),
  }
}

// ---- PLC ----
function addDevice() {
  doc.plc.push(normalizeDevice({
    code: '', model: '', host: '', port: 102, rack: 0, slot: 1, protocol: 's7',
    poll_groups: [],
  }))
}

function addDeviceFrom(code: string) {
  if (!code) return
  const exist = doc.plc.find((d: any) => d.code === code)
  if (exist) {
    msg.warning(`设备 ${code} 已存在`)
    return
  }
  const found = nodes.value.find((n: any) => n.code === selected.value?.code)
  doc.plc.push(normalizeDevice({
    code, model: found?.type || 'tmt_winder', host: '', port: 102, rack: 0, slot: 1,
    protocol: 's7', poll_groups: [],
  }))
}

function openPoints(di: number) {
  editDevice.value = doc.plc[di]
  showPoints.value = true
}

function addPollGroup() {
  if (!editDevice.value) return
  editDevice.value.poll_groups.push({
    name: 'winder_data', interval_ms: 500, db: 0, start: 0, length: 0,
    repeat: { count: 0, stride: 0, sides: { left: { offset: 0 }, right: { offset: 0 } } },
    points: [], _single: false,
  })
}

function addPoint(g: any) {
  g.points.push({ name: '', offset: 0, type: 'uint16', length: 0, direction: 'read' })
}

const pointCols = (g: any) => [
  {
    title: '字段名', key: 'name', width: 260,
    render: (row: any) => h(NInput, {
      value: row.name, size: 'small',
      'onUpdate:value': (v: string) => { row.name = v },
    }),
  },
  {
    title: '偏移', key: 'offset', width: 100,
    render: (row: any) => h(NInputNumber, {
      value: row.offset, size: 'small', min: 0,
      'onUpdate:value': (v: number | null) => { row.offset = v ?? 0 },
    }),
  },
  {
    title: '类型', key: 'type', width: 130,
    render: (row: any) => h(NSelect, {
      value: row.type, size: 'small',
      options: ['bool', 'uint8', 'int8', 'uint16', 'int16', 'uint32', 'int32', 'float32', 'float64', 'string']
        .map((t) => ({ label: t, value: t })),
      'onUpdate:value': (v: string) => { row.type = v },
    }),
  },
  {
    title: '字节长度', key: 'length', width: 110,
    render: (row: any) => h(NInputNumber, {
      value: row.length, size: 'small', min: 0,
      'onUpdate:value': (v: number | null) => { row.length = v ?? 0 },
    }),
  },
  {
    title: '位号', key: 'bit', width: 80,
    render: (row: any) => h(NInputNumber, {
      value: row.bit, size: 'small', min: 0,
      'onUpdate:value': (v: number | null) => { row.bit = v ?? 0 },
    }),
  },
  {
    title: '', key: 'op', width: 60,
    render: (row: any) => h(NButton, {
      size: 'tiny', text: true, type: 'error',
      onClick: () => { g.points.splice(g.points.indexOf(row), 1) },
    }, () => '删除'),
  },
]

// ---- 装载表 ----
const loadingCols = [
  {
    title: '锭位', key: 'spindle_place', width: 110,
    render: (row: any) => h(NInputNumber, {
      value: row.spindle_place, size: 'small', min: 1,
      'onUpdate:value': (v: number | null) => { row.spindle_place = v ?? 0 },
    }),
  },
  {
    title: 'A 侧位置', key: 'a', width: 110,
    render: (row: any) => h(NInputNumber, {
      value: row.a, size: 'small', min: 1,
      'onUpdate:value': (v: number | null) => { row.a = v ?? 0 },
    }),
  },
  {
    title: 'B 侧位置', key: 'b', width: 110,
    render: (row: any) => h(NInputNumber, {
      value: row.b, size: 'small', min: 1,
      'onUpdate:value': (v: number | null) => { row.b = v ?? 0 },
    }),
  },
]

function addPattern() {
  doc.loading.patterns.push({ code: '', map: [] })
}

function addLoadingRow(p: any) {
  const n = (p.map?.length || 0) + 1
  p.map.push({ spindle_place: n, a: n, b: n })
}

// ---- 锭位映射 ----
const headCols = [
  {
    title: '头号', key: 'head', width: 90,
    render: (row: any) => h(NInputNumber, {
      value: row.head, size: 'small', min: 1,
      'onUpdate:value': (v: number | null) => { row.head = v ?? 0 },
    }),
  },
  {
    title: '段序', key: 'step', width: 90,
    render: (row: any) => h(NInputNumber, {
      value: row.step, size: 'small', min: 1,
      'onUpdate:value': (v: number | null) => { row.step = v ?? 0 },
    }),
  },
  {
    title: '锭位', key: 'position', width: 90,
    render: (row: any) => h(NInputNumber, {
      value: row.position, size: 'small', min: 1,
      'onUpdate:value': (v: number | null) => { row.position = v ?? 0 },
    }),
  },
]

function addHeadMap() {
  doc.heads.head_maps.push({ head_count: 6, steps_per_head: 4, capacity: 24, entries: [] })
}

// 按头数 × 段数生成一份顺序填充的空表，现场只需调整顺序
function fillHeadMap(m: any) {
  const total = (m.head_count || 0) * (m.steps_per_head || 0)
  if (total <= 0) {
    msg.warning('请先填写头数与每头段数')
    return
  }
  m.capacity = total
  m.entries = []
  let pos = 1
  for (let head = 1; head <= m.head_count; head++) {
    for (let step = 1; step <= m.steps_per_head; step++) {
      m.entries.push({ head, step, position: pos++ })
    }
  }
}

// ---- 保存 ----
// 组装为后端文档：去掉前端辅助字段 _single，并把单面锭位的 sides 置为 null
function buildPayload() {
  const plc = doc.plc.map((d: any) => {
    const dev: any = {
      code: d.code, model: d.model, host: d.host,
      port: d.port, rack: d.rack, slot: d.slot, protocol: d.protocol || 's7',
    }
    const groups = (d.poll_groups || []).map((g: any) => {
      const out: any = {
        name: g.name, interval_ms: g.interval_ms, db: g.db,
        start: g.start, length: g.length, points: g.points || [],
      }
      if (g.repeat && (g.repeat.count > 0 || g.repeat.stride > 0)) {
        out.repeat = {
          count: g.repeat.count, stride: g.repeat.stride,
          sides: g._single ? null : g.repeat.sides,
        }
      }
      return out
    })
    if (groups.length) dev.poll_groups = groups
    return dev
  })

  return {
    remark: doc.remark,
    plc,
    loading: doc.loading?.patterns?.length ? doc.loading : null,
    heads: doc.heads?.head_maps?.length ? doc.heads : null,
  }
}

async function validateOnly() {
  if (!selected.value) return
  validating.value = true
  problems.value = []
  okTip.value = ''
  try {
    const res: any = await http.post(`/edge-nodes/${selected.value.code}/site-config/validate`, buildPayload())
    problems.value = res.data?.problems || []
    if (!problems.value.length) okTip.value = '校验通过，可以保存'
  } catch (e: any) {
    problems.value = [e?.response?.data?.message || e?.message || String(e)]
  } finally {
    validating.value = false
  }
}

async function save() {
  if (!selected.value) return
  saving.value = true
  problems.value = []
  okTip.value = ''
  try {
    const res: any = await http.put(`/edge-nodes/${selected.value.code}/site-config`, buildPayload())
    doc.version = res.data?.version || doc.version
    okTip.value = `已保存为版本 ${doc.version}，等待边端人工确认应用`
    msg.success('配置已下发')
  } catch (e: any) {
    const body = e?.response?.data
    problems.value = body?.data?.problems || [body?.message || e?.message || String(e)]
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.site-config { display: flex; gap: 16px; min-height: calc(100vh - 120px); }
.node-list {
  width: 220px; flex-shrink: 0; background: #161b22; border-radius: 8px;
  border: 1px solid #30363d; padding: 12px;
}
.list-title { font-size: 12px; color: #8b949e; font-weight: 600; margin-bottom: 8px; }
.node-item {
  padding: 8px 10px; border-radius: 6px; cursor: pointer; margin-bottom: 4px;
  border: 1px solid transparent; transition: all 0.15s;
}
.node-item:hover { background: #1c2333; }
.node-item.active { background: #1f6feb; }
.node-item.active .node-name,
.node-item.active .node-meta { color: #fff; }
.node-name { font-size: 13px; color: #c9d1d9; }
.node-meta { display: flex; justify-content: space-between; align-items: center; font-size: 11px; color: #8b949e; margin-top: 2px; }
.editor { flex: 1; min-width: 0; }

.device-list { display: flex; flex-direction: column; gap: 10px; }
.device-card { background: #0d1117; border: 1px solid #30363d; border-radius: 6px; padding: 12px; margin-bottom: 10px; }
.device-head { display: flex; justify-content: space-between; align-items: center; }
.device-code { font-weight: 600; font-size: 13px; }
.device-host { color: #8b949e; font-size: 12px; }
.hint { color: #8b949e; font-size: 12px; }
.problems { margin: 6px 0 0 18px; padding: 0; }
.problems li { font-size: 12px; line-height: 1.7; }
</style>
