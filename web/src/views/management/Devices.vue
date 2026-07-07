<template>
  <div>
    <n-space style="margin-bottom:12px" justify="space-between">
      <n-space>
        <n-select v-model:value="typeFilter" :options="typeOpts" placeholder="全部类型" clearable style="width:140px" @update:value="load" />
        <n-select v-model:value="stationFilter" :options="stationOpts" placeholder="全部区域" clearable style="width:120px" @update:value="load" />
        <n-select v-model:value="statusFilter" :options="statusOpts" placeholder="全部状态" clearable style="width:120px" @update:value="load" />
      </n-space>
      <n-button type="primary" size="small" @click="openForm()">新增设备</n-button>
    </n-space>
    <n-data-table :columns="cols" :data="items" :bordered="false" size="small" />

    <n-modal v-model:show="showForm" preset="card" :title="form.id?'编辑设备':'新增设备'" style="width:520px">
      <n-form :model="form" label-placement="left" label-width="80" size="small">
        <!-- 第一步：选类型 -->
        <n-form-item label="设备类型"><n-select v-model:value="form.type" :options="typeOpts" @update:value="onTypeChange" /></n-form-item>
        <!-- 基本信息 -->
        <n-grid :cols="2" :x-gap="12">
          <n-gi><n-form-item label="编号"><n-input v-model:value="form.code" /></n-form-item></n-gi>
          <n-gi><n-form-item label="名称"><n-input v-model:value="form.name" /></n-form-item></n-gi>
          <n-gi><n-form-item label="所属区域"><n-select v-model:value="form.station_type" :options="stationOpts" clearable @update:value="form.line_code=''" /></n-form-item></n-gi>
          <n-gi><n-form-item label="所属产线"><n-select v-model:value="form.line_code" :options="lineOpts" clearable /></n-form-item></n-gi>
          <n-gi><n-form-item label="物理位置"><n-input v-model:value="form.location" /></n-form-item></n-gi>
        </n-grid>
        <!-- 通讯类设备 -->
        <template v-if="isCommDevice">
          <n-divider style="margin:8px 0">通讯参数</n-divider>
          <n-grid :cols="2" :x-gap="12">
            <n-gi><n-form-item label="品牌"><n-input v-model:value="form.brand" /></n-form-item></n-gi>
            <n-gi><n-form-item label="型号"><n-input v-model:value="form.model" /></n-form-item></n-gi>
            <n-gi><n-form-item label="IP"><n-input v-model:value="form.ip" /></n-form-item></n-gi>
            <n-gi><n-form-item label="端口"><n-input-number v-model:value="form.port" /></n-form-item></n-gi>
            <n-gi><n-form-item label="通讯协议"><n-select v-model:value="form.protocol" :options="protocolOpts" clearable /></n-form-item></n-gi>
            <n-gi v-if="form.type==='printer'"><n-form-item label="备用产线"><n-input v-model:value="form.backup_for" placeholder="L02,L03" /></n-form-item></n-gi>
          </n-grid>
        </template>
        <!-- 存置架 -->
        <template v-if="form.type==='storage_rack'">
          <n-divider style="margin:8px 0">存置参数</n-divider>
          <n-grid :cols="2" :x-gap="12">
            <n-gi><n-form-item label="行数"><n-input-number v-model:value="form.rows" :min="1" /></n-form-item></n-gi>
            <n-gi><n-form-item label="列数"><n-input-number v-model:value="form.cols" :min="1" /></n-form-item></n-gi>
          </n-grid>
          <n-form-item label="位置编号"><n-input v-model:value="form.slot_labels" placeholder="如 A1,A2,A3...（逗号分隔，留空自动生成）" /></n-form-item>
        </template>
        <!-- 状态与维保 -->
        <n-divider style="margin:8px 0" />
        <n-grid :cols="2" :x-gap="12">
          <n-gi><n-form-item label="状态"><n-select v-model:value="form.status" :options="formStatusOpts" /></n-form-item></n-gi>
          <n-gi><n-form-item label="纳入维保"><n-switch v-model:value="form.enable_maintenance" /></n-form-item></n-gi>
          <n-gi><n-form-item label="接受告警"><n-switch v-model:value="form.enable_alarm" /></n-form-item></n-gi>
        </n-grid>
        <n-button type="primary" block @click="save">保存</n-button>
      </n-form>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, h, computed } from 'vue'
import { NButton, NSpace, NTag } from 'naive-ui'
import http from '../../api/http'

const items = ref<any[]>([])
const lines = ref<any[]>([])
const typeFilter = ref<string | null>(null)
const stationFilter = ref<string | null>(null)
const statusFilter = ref<string | null>(null)
const showForm = ref(false)
const form = ref(emptyForm())
const dictMap = ref<Record<string, any[]>>({})

const commTypes = ['edge_pc', 'printer', 'plc', 'scanner', 'pda', 'winder', 'packaging_machine', 'robot_arm']

function emptyForm() {
  return { id: 0, code: '', name: '', type: 'edge_pc', station_type: '', line_code: '', brand: '', model: '', ip: '', port: 0, protocol: '', status: 'unknown', backup_for: '', location: '', enable_maintenance: false, enable_alarm: true, rows: 0, cols: 0, slot_labels: '' }
}

const isCommDevice = computed(() => commTypes.includes(form.value.type))
const lineOpts = computed(() => lines.value.filter(l => !form.value.station_type || l.station_type === form.value.station_type).map(l => ({ label: `${l.code} - ${l.name}`, value: l.code })))

function onTypeChange(t: string) {
  if (commTypes.includes(t)) {
    if (form.value.status === 'in_use') form.value.status = 'unknown'
  } else {
    form.value.status = 'in_use'
    form.value.ip = ''; form.value.port = 0; form.value.protocol = ''
  }
}

const typeOpts = computed(() => (dictMap.value['device_type'] || []).map((d: any) => ({ label: d.name, value: d.code })))
const stationOpts = computed(() => (dictMap.value['station_type'] || []).map((d: any) => ({ label: d.name, value: d.code })))
const protocolOpts = computed(() => (dictMap.value['protocol'] || []).map((d: any) => ({ label: d.name, value: d.code })))
const statusOpts = computed(() => (dictMap.value['device_status'] || []).map((d: any) => ({ label: d.name, value: d.code })))
const formStatusOpts = computed(() => {
  const all = dictMap.value['device_status'] || []
  if (isCommDevice.value) {
    return all.filter((d: any) => ['running', 'offline', 'fault', 'unknown'].includes(d.code)).map((d: any) => ({ label: d.name, value: d.code }))
  }
  return all.filter((d: any) => ['in_use', 'deactivated', 'maintenance'].includes(d.code)).map((d: any) => ({ label: d.name, value: d.code }))
})

const dictLabel = (category: string, code: string) => (dictMap.value[category] || []).find((d: any) => d.code === code)?.name || code || '-'

const cols = [
  { title: '编号', key: 'code', width: 110 },
  { title: '名称', key: 'name', width: 120 },
  { title: '类型', key: 'type', width: 90, render: (r: any) => dictLabel('device_type', r.type) },
  { title: '区域', key: 'station_type', width: 70, render: (r: any) => dictLabel('station_type', r.station_type) },
  { title: '产线', key: 'line_code', width: 60 },
  { title: '品牌/型号', key: 'brand', width: 120, render: (r: any) => [r.brand, r.model].filter(Boolean).join(' ') || '-' },
  { title: 'IP', key: 'ip', width: 120, render: (r: any) => r.ip ? `${r.ip}${r.port ? ':' + r.port : ''}` : '-' },
  { title: '协议', key: 'protocol', width: 80, render: (r: any) => r.protocol ? dictLabel('protocol', r.protocol) : '-' },
  { title: '状态', key: 'status', width: 70, render: (r: any) => {
    const type = r.protocol ? (r.status === 'running' ? 'success' : r.status === 'fault' ? 'error' : 'default') : (r.status === 'in_use' ? 'success' : r.status === 'maintenance' ? 'warning' : 'default')
    return h(NTag, { type, size: 'small' }, () => dictLabel('device_status', r.status))
  } },
  { title: '操作', key: 'a', width: 100, render: (r: any) => h(NSpace, { size: 4 }, () => [
    h(NButton, { size: 'tiny', onClick: () => openForm(r) }, () => '编辑'),
    h(NButton, { size: 'tiny', type: 'error', onClick: () => del(r.id) }, () => '删除'),
  ]) },
]

function openForm(row?: any) {
  form.value = row ? { ...row, slot_labels: row.slot_labels || '' } : emptyForm()
  showForm.value = true
}

async function loadDict() {
  const res: any = await http.get('/admin/dict-items')
  const map: Record<string, any[]> = {}
  for (const item of (res.data || [])) {
    if (!map[item.category]) map[item.category] = []
    map[item.category].push(item)
  }
  dictMap.value = map
}

async function load() {
  const params = new URLSearchParams()
  if (typeFilter.value) params.set('type', typeFilter.value)
  if (stationFilter.value) params.set('station_type', stationFilter.value)
  if (statusFilter.value) params.set('status', statusFilter.value)
  const qs = params.toString()
  items.value = ((await http.get(`/devices${qs ? '?' + qs : ''}`)) as any).data || []
}

async function save() {
  if (form.value.id) {
    await http.put(`/devices/${form.value.id}`, form.value)
  } else {
    await http.post('/devices', form.value)
  }
  showForm.value = false; load()
}
async function del(id: number) { await http.delete(`/devices/${id}`); load() }

onMounted(async () => {
  await loadDict()
  lines.value = ((await http.get('/admin/production-lines')) as any).data || []
  load()
})
</script>