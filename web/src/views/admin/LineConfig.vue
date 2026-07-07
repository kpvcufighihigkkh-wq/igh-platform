<template>
  <div class="line-config">
    <div class="line-list">
      <n-button size="small" type="primary" block style="margin-bottom:12px" @click="openLineForm()">新增产线</n-button>
      <template v-for="g in groupedLines" :key="g.type">
        <div class="group-header">{{ g.label }}</div>
        <div v-for="l in g.lines" :key="l.id" class="line-item"
             :class="{ active: selectedLine?.id === l.id }" @click="selectLine(l)">
          <span>{{ l.name }}</span>
          <n-tag v-if="l.status === 'stopped'" type="warning" size="small">停机</n-tag>
        </div>
      </template>
      <n-empty v-if="!allLines.length" description="暂无产线" size="small" style="margin-top:40px" />
    </div>

    <div class="line-detail" v-if="selectedLine">
      <n-card size="small">
        <template #header>
          <n-space justify="space-between" align="center" style="width:100%">
            <span>基本信息</span>
            <n-space :size="8">
              <n-button size="small" type="info" @click="openLineForm(selectedLine)">编辑</n-button>
              <n-popconfirm @positive-click="deleteLine(selectedLine.id)">
                <template #trigger><n-button size="small" type="error">删除</n-button></template>
                确定删除该产线？
              </n-popconfirm>
            </n-space>
          </n-space>
        </template>
        <n-descriptions :column="3" label-placement="left" size="small" bordered>
          <n-descriptions-item label="编号">{{ selectedLine.code }}</n-descriptions-item>
          <n-descriptions-item label="名称">{{ selectedLine.name }}</n-descriptions-item>
          <n-descriptions-item label="工位区域">{{ tStation(selectedLine.station_type) }}</n-descriptions-item>
          <n-descriptions-item label="产品类型">{{ selectedLine.product_type || '-' }}</n-descriptions-item>
          <n-descriptions-item label="状态">{{ tStatus(selectedLine.status) }}</n-descriptions-item>
        </n-descriptions>
      </n-card>

      <n-card v-if="selectedLine.station_type === 'winding'" title="PLC 绑定" size="small" style="margin-top:12px">
        <n-space align="center">
          <span style="color:#8b949e;font-size:13px">绑定 PLC：</span>
          <n-select v-model:value="selectedPlcCode" :options="plcOpts" clearable
                    placeholder="选择PLC设备" style="width:240px" />
          <n-button size="small" type="primary" @click="savePlcBinding">保存</n-button>
        </n-space>
      </n-card>

      <n-card v-if="selectedLine.station_type === 'winding'" size="small" style="margin-top:12px">
        <template #header>
          <n-space justify="space-between" align="center" style="width:100%">
            <span>卷绕机列表</span>
            <n-button size="small" type="primary" @click="openWinderForm()">添加卷绕机</n-button>
          </n-space>
        </template>
        <div class="winder-list">
          <div v-for="w in winders" :key="w.id" class="winder-item">
            <div class="winder-info">
              <span class="winder-code">{{ w.code }}</span>
              <span class="winder-name">{{ w.name }}</span>
              <n-tag size="small" :bordered="false">DB:{{ parseConfig(w).db_block ?? '-' }}</n-tag>
              <span class="reg-count">{{ parseConfig(w).registers?.length || 0 }} 寄存器</span>
            </div>
            <n-space :size="4">
              <n-button text type="info" size="small" @click="openWinderForm(w)">编辑</n-button>
              <n-button text type="error" size="small" @click="deleteWinder(w.id)">删除</n-button>
            </n-space>
          </div>
          <n-empty v-if="!winders.length" description="暂未配置卷绕机" size="small" style="margin:20px 0" />
        </div>
      </n-card>
    </div>
    <div class="line-detail" v-else>
      <n-empty description="请选择左侧产线" style="margin-top:100px" />
    </div>

    <!-- Line Add/Edit Modal -->
    <n-modal v-model:show="showLineModal" preset="card" :title="lineForm.id ? '编辑产线' : '新增产线'" style="width:400px">
      <n-form :model="lineForm" label-placement="left" label-width="70">
        <n-form-item label="工位区域">
          <n-select v-model:value="lineForm.station_type" :options="stationOpts" />
        </n-form-item>
        <n-form-item label="编号"><n-input v-model:value="lineForm.code" placeholder="L01" /></n-form-item>
        <n-form-item label="名称"><n-input v-model:value="lineForm.name" placeholder="卷绕线01" /></n-form-item>
        <n-form-item label="产品类型">
          <n-select v-model:value="lineForm.product_types" :options="ptOpts" multiple placeholder="可多选" />
        </n-form-item>
        <n-form-item label="状态">
          <n-select v-model:value="lineForm.status" :options="lineStatusOpts" />
        </n-form-item>
        <n-button type="primary" block @click="saveLine">保存</n-button>
      </n-form>
    </n-modal>

    <!-- Winder Add/Edit Modal -->
    <n-modal v-model:show="showWinderModal" preset="card" :title="winderForm.id ? '编辑卷绕机' : '添加卷绕机'" style="width:640px">
      <n-form :model="winderForm" label-placement="left" label-width="80">
        <n-grid :cols="3" :x-gap="12">
          <n-gi><n-form-item label="编号"><n-input v-model:value="winderForm.code" placeholder="WDC-L01-3A" /></n-form-item></n-gi>
          <n-gi><n-form-item label="名称"><n-input v-model:value="winderForm.name" placeholder="卷绕机L01-3A" /></n-form-item></n-gi>
          <n-gi><n-form-item label="DB块号"><n-input-number v-model:value="winderForm.db_block" :min="0" /></n-form-item></n-gi>
        </n-grid>
      </n-form>

      <n-divider style="margin:8px 0">寄存器映射</n-divider>
      <n-space style="margin-bottom:8px">
        <n-button size="small" @click="addRegister">添加寄存器</n-button>
      </n-space>
      <n-data-table :columns="registerCols" :data="winderForm.registers" :bordered="false" size="small" />

      <template #footer>
        <n-space justify="end">
          <n-button @click="showWinderModal = false">取消</n-button>
          <n-button type="primary" @click="saveWinder" :loading="saving">保存</n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, h } from 'vue'
import { NInput, NInputNumber, NSelect, NButton, useMessage } from 'naive-ui'
import http from '../../api/http'
import { tStation, tStatus } from '../../utils/status'

const msg = useMessage()

interface Register { name: string; key: string; type: string; offset: number }
interface WinderConfig { db_block: number; registers: Register[] }

const allLines = ref<any[]>([])
const selectedLine = ref<any>(null)
const plcDevices = ref<any[]>([])
const selectedPlcCode = ref<string | null>(null)
const winders = ref<any[]>([])
const productTypes = ref<any[]>([])
const saving = ref(false)

const stationGroups = [
  { type: 'winding', label: '卷绕区' },
  { type: 'sorting', label: '分拣区' },
  { type: 'packaging', label: '包装区' },
  { type: 'warehouse', label: '仓库区' },
]

const groupedLines = computed(() =>
  stationGroups
    .map(g => ({ ...g, lines: allLines.value.filter(l => l.station_type === g.type) }))
    .filter(g => g.lines.length > 0)
)

const plcOpts = computed(() => plcDevices.value.map(d => ({ label: `${d.code} - ${d.name} (${d.ip})`, value: d.code })))
const ptOpts = computed(() => productTypes.value.map(p => ({ label: `${p.code} - ${p.name}`, value: p.code })))

const stationOpts = [
  { label: '卷绕区', value: 'winding' },
  { label: '分拣区', value: 'sorting' },
  { label: '包装区', value: 'packaging' },
  { label: '仓库区', value: 'warehouse' },
]
const lineStatusOpts = [
  { label: '运行中', value: 'running' },
  { label: '停机', value: 'stopped' },
  { label: '维护中', value: 'maintenance' },
]

function parseConfig(w: any): WinderConfig {
  try { return w.protocol_config ? JSON.parse(w.protocol_config) : { db_block: 0, registers: [] } }
  catch { return { db_block: 0, registers: [] } }
}

// --- Data fetching ---

async function fetchLines() {
  const res: any = await http.get('/admin/production-lines')
  allLines.value = res.data || []
}

async function fetchPlcDevices() {
  const res: any = await http.get('/devices?type=plc')
  plcDevices.value = (res.data as any)?.items || res.data || []
}

async function fetchProductTypes() {
  const res: any = await http.get('/admin/product-types')
  productTypes.value = res.data || []
}

async function selectLine(line: any) {
  selectedLine.value = line
  if (line.station_type === 'winding') {
    let cfg: any = {}
    try { cfg = typeof line.config === 'string' ? JSON.parse(line.config) : (line.config || {}) } catch {}
    selectedPlcCode.value = cfg.plc_code || null
    await fetchWinders(line.code)
  }
}

async function fetchWinders(lineCode: string) {
  const res: any = await http.get(`/devices?type=winder&line_code=${lineCode}`)
  winders.value = (res.data as any)?.items || res.data || []
}

// --- Line CRUD ---

const showLineModal = ref(false)
const lineForm = ref({ id: 0, code: '', name: '', station_type: 'winding', product_types: [] as string[], status: 'running' })

function openLineForm(row?: any) {
  if (row) {
    lineForm.value = { id: row.id, code: row.code, name: row.name, station_type: row.station_type || 'winding', product_types: row.product_type ? row.product_type.split(',') : [], status: row.status || 'running' }
  } else {
    lineForm.value = { id: 0, code: '', name: '', station_type: 'winding', product_types: [], status: 'running' }
  }
  showLineModal.value = true
}

async function saveLine() {
  const payload = { ...lineForm.value, product_type: lineForm.value.product_types.join(',') }
  if (lineForm.value.id) {
    await http.put(`/admin/production-lines/${lineForm.value.id}`, payload)
  } else {
    await http.post('/admin/production-lines', payload)
  }
  msg.success('产线已保存')
  showLineModal.value = false
  await fetchLines()
  if (lineForm.value.id) {
    const updated = allLines.value.find(l => l.id === lineForm.value.id)
    if (updated) selectedLine.value = updated
  }
}

async function deleteLine(id: number) {
  await http.delete(`/admin/production-lines/${id}`)
  msg.success('已删除')
  selectedLine.value = null
  await fetchLines()
}

// --- PLC Binding ---

async function savePlcBinding() {
  if (!selectedLine.value) return
  const config = selectedPlcCode.value ? { plc_code: selectedPlcCode.value } : {}
  await http.put(`/admin/production-lines/${selectedLine.value.id}`, { ...selectedLine.value, config })
  msg.success('PLC 绑定已保存')
  await fetchLines()
  const updated = allLines.value.find(l => l.id === selectedLine.value.id)
  if (updated) selectedLine.value = updated
}

// --- Winder Modal ---
const showWinderModal = ref(false)
const winderForm = ref({ id: 0, code: '', name: '', db_block: 0, registers: [] as Register[] })

const typeOptions = [
  { label: 'Bool', value: 'bool' }, { label: 'Int16', value: 'int16' },
  { label: 'Int32', value: 'int32' }, { label: 'Float32', value: 'float32' },
  { label: 'String', value: 'string' },
]

const registerCols = [
  { title: '名称', key: 'name', render: (_: any, idx: number) => h(NInput, { value: winderForm.value.registers[idx].name, size: 'small', onUpdateValue: (v: string) => { winderForm.value.registers[idx].name = v } }) },
  { title: 'Key', key: 'key', width: 110, render: (_: any, idx: number) => h(NInput, { value: winderForm.value.registers[idx].key, size: 'small', onUpdateValue: (v: string) => { winderForm.value.registers[idx].key = v } }) },
  { title: '类型', key: 'type', width: 110, render: (_: any, idx: number) => h(NSelect, { value: winderForm.value.registers[idx].type, options: typeOptions, size: 'small', onUpdateValue: (v: string) => { winderForm.value.registers[idx].type = v } }) },
  { title: '偏移量', key: 'offset', width: 90, render: (_: any, idx: number) => h(NInputNumber, { value: winderForm.value.registers[idx].offset, size: 'small', min: 0, onUpdateValue: (v: number | null) => { winderForm.value.registers[idx].offset = v ?? 0 } }) },
  { title: '', key: 'op', width: 50, render: (_: any, idx: number) => h(NButton, { text: true, type: 'error', size: 'small', onClick: () => winderForm.value.registers.splice(idx, 1) }, () => '删除') },
]

function addRegister() {
  winderForm.value.registers.push({ name: '', key: '', type: 'int16', offset: 0 })
}

function openWinderForm(w?: any) {
  if (w) {
    const cfg = parseConfig(w)
    winderForm.value = { id: w.id, code: w.code, name: w.name, db_block: cfg.db_block, registers: [...cfg.registers] }
  } else {
    winderForm.value = { id: 0, code: '', name: '', db_block: 0, registers: [] }
  }
  showWinderModal.value = true
}

async function saveWinder() {
  if (!selectedLine.value) return
  saving.value = true
  const protocolConfig = JSON.stringify({ db_block: winderForm.value.db_block, registers: winderForm.value.registers })
  const body: any = {
    code: winderForm.value.code, name: winderForm.value.name,
    type: 'winder', station_type: 'winding', line_code: selectedLine.value.code,
    protocol_config: protocolConfig, status: 'unknown',
    enable_maintenance: true, enable_alarm: true,
  }
  try {
    if (winderForm.value.id) {
      await http.put(`/devices/${winderForm.value.id}`, body)
    } else {
      await http.post('/devices', body)
    }
    msg.success('保存成功')
    showWinderModal.value = false
    await fetchWinders(selectedLine.value.code)
  } catch (e: any) {
    msg.error(e?.message || '保存失败')
  }
  saving.value = false
}

async function deleteWinder(id: number) {
  await http.delete(`/devices/${id}`)
  msg.success('已删除')
  if (selectedLine.value) await fetchWinders(selectedLine.value.code)
}

onMounted(() => { fetchLines(); fetchPlcDevices(); fetchProductTypes() })
</script>

<style scoped>
.line-config { display: flex; gap: 16px; min-height: calc(100vh - 120px); }
.line-list {
  width: 220px; flex-shrink: 0; background: #161b22; border-radius: 8px;
  border: 1px solid #30363d; padding: 12px;
}
.group-header { font-size: 12px; color: #8b949e; margin: 12px 0 6px 4px; font-weight: 600; }
.group-header:first-of-type { margin-top: 0; }
.line-item {
  display: flex; justify-content: space-between; align-items: center;
  padding: 8px 10px; border-radius: 6px; cursor: pointer; font-size: 13px;
  color: #c9d1d9; transition: all 0.15s; margin-bottom: 4px;
}
.line-item:hover { background: #1c2333; }
.line-item.active { background: #1f6feb; color: #fff; }
.line-detail { flex: 1; min-width: 0; }

.winder-list { display: flex; flex-direction: column; gap: 8px; }
.winder-item {
  display: flex; justify-content: space-between; align-items: center;
  padding: 10px 12px; background: #0d1117; border: 1px solid #30363d;
  border-radius: 6px; transition: border-color 0.15s;
}
.winder-item:hover { border-color: #22d3ee; }
.winder-info { display: flex; align-items: center; gap: 12px; }
.winder-code { font-weight: 600; font-size: 13px; }
.winder-name { color: #8b949e; font-size: 12px; }
.reg-count { color: #8b949e; font-size: 11px; }
</style>
