<template>
  <div>
    <n-space justify="space-between" style="margin-bottom:12px">
      <n-space>
        <n-select v-model:value="typeFilter" :options="typeOpts" placeholder="全部类型" clearable style="width:140px" @update:value="load" />
        <n-select v-model:value="statusFilter" :options="statusOpts" placeholder="全部状态" clearable style="width:120px" @update:value="load" />
      </n-space>
      <n-button type="primary" size="small" @click="openForm()">新增载具</n-button>
    </n-space>
    <n-data-table :columns="cols" :data="items" :bordered="false" size="small" />

    <n-modal v-model:show="showForm" preset="card" :title="form.id?'编辑载具':'新增载具'" style="width:500px">
      <n-form :model="form" label-placement="left" label-width="80" size="small">
        <!-- 第一步：选类型 -->
        <n-form-item label="载具类型"><n-select v-model:value="form.type" :options="typeOpts" /></n-form-item>
        <!-- 基本信息 -->
        <n-grid :cols="2" :x-gap="12">
          <n-gi><n-form-item label="编号"><n-input v-model:value="form.code" /></n-form-item></n-gi>
          <n-gi><n-form-item label="名称"><n-input v-model:value="form.name" /></n-form-item></n-gi>
          <n-gi><n-form-item label="当前位置"><n-input v-model:value="form.position" placeholder="如 L01落纱区" /></n-form-item></n-gi>
          <n-gi><n-form-item label="状态"><n-select v-model:value="form.status" :options="statusOpts" /></n-form-item></n-gi>
        </n-grid>
        <!-- 物理载具: 丝车/吊车 -->
        <template v-if="isPhysical">
          <n-divider style="margin:8px 0">载具规格</n-divider>
          <n-grid :cols="2" :x-gap="12">
            <n-gi><n-form-item label="行数"><n-input-number v-model:value="form.rows" :min="1" /></n-form-item></n-gi>
            <n-gi><n-form-item label="列数"><n-input-number v-model:value="form.cols" :min="1" /></n-form-item></n-gi>
            <n-gi><n-form-item label="正反两面"><n-switch v-model:value="form.double_side" /></n-form-item></n-gi>
            <n-gi><n-form-item label="有RFID"><n-switch v-model:value="form.has_rfid" /></n-form-item></n-gi>
            <n-gi v-if="form.has_rfid" :span="2"><n-form-item label="RFID编码"><n-input v-model:value="form.rfid" /></n-form-item></n-gi>
            <n-gi><n-form-item label="规格型号"><n-input v-model:value="form.spec" /></n-form-item></n-gi>
            <n-gi><n-form-item label="最大承重"><n-input v-model:value="form.max_weight" placeholder="如 200kg" /></n-form-item></n-gi>
            <n-gi><n-form-item label="轮径/轮距"><n-input v-model:value="form.wheel_spec" /></n-form-item></n-gi>
            <n-gi><n-form-item label="适配丝饼"><n-input v-model:value="form.bobbin_spec" placeholder="如 40D/34F" /></n-form-item></n-gi>
          </n-grid>
        </template>
        <!-- 智能载具: 天轨/AGV/穿梭车 -->
        <template v-if="isSmart">
          <n-divider style="margin:8px 0">通讯参数</n-divider>
          <n-grid :cols="2" :x-gap="12">
            <n-gi><n-form-item label="通讯协议"><n-select v-model:value="form.protocol" :options="protocolOpts" clearable /></n-form-item></n-gi>
            <n-gi><n-form-item label="IP"><n-input v-model:value="form.ip" /></n-form-item></n-gi>
            <n-gi><n-form-item label="端口"><n-input-number v-model:value="form.port" /></n-form-item></n-gi>
            <n-gi><n-form-item label="容量"><n-input v-model:value="form.capacity" /></n-form-item></n-gi>
          </n-grid>
        </template>
        <n-divider style="margin:8px 0" />
        <n-grid :cols="2" :x-gap="12">
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
const typeFilter = ref<string | null>(null)
const statusFilter = ref<string | null>(null)
const showForm = ref(false)
const form = ref(emptyForm())
const dictMap = ref<Record<string, any[]>>({})

const physicalTypes = ['trolley', 'crane']
const smartTypes = ['rail_car', 'agv', 'shuttle']

function emptyForm() {
  return { id: 0, code: '', name: '', type: 'trolley', status: 'idle', position: '', capacity: '', protocol: '', ip: '', port: 0, enable_maintenance: false, enable_alarm: true, rows: 0, cols: 0, double_side: false, has_rfid: false, rfid: '', spec: '', max_weight: '', wheel_spec: '', bobbin_spec: '' }
}

const isPhysical = computed(() => physicalTypes.includes(form.value.type))
const isSmart = computed(() => smartTypes.includes(form.value.type))

const typeOpts = computed(() => (dictMap.value['vehicle_type'] || []).map((d: any) => ({ label: d.name, value: d.code })))
const statusOpts = computed(() => (dictMap.value['vehicle_status'] || []).map((d: any) => ({ label: d.name, value: d.code })))
const protocolOpts = computed(() => (dictMap.value['protocol'] || []).map((d: any) => ({ label: d.name, value: d.code })))

const dictLabel = (category: string, code: string) => (dictMap.value[category] || []).find((d: any) => d.code === code)?.name || code || '-'

const cols = [
  { title: '编号', key: 'code', width: 80 },
  { title: '名称', key: 'name', width: 100 },
  { title: '类型', key: 'type', width: 80, render: (r: any) => dictLabel('vehicle_type', r.type) },
  { title: '状态', key: 'status', width: 70, render: (r: any) => h(NTag, { type: r.status === 'fault' ? 'error' : r.status === 'idle' ? 'default' : 'success', size: 'small' }, () => dictLabel('vehicle_status', r.status)) },
  { title: '位置', key: 'position', width: 100 },
  { title: '容量', key: '_cap', width: 90, render: (r: any) => r.rows && r.cols ? `${r.rows}×${r.cols}${r.double_side ? '×2面' : ''}` : r.capacity || '-' },
  { title: 'RFID', key: 'rfid', width: 60, render: (r: any) => r.has_rfid ? (r.rfid || '有') : '-' },
  { title: '协议', key: 'protocol', width: 70, render: (r: any) => r.protocol ? dictLabel('protocol', r.protocol) : '-' },
  { title: '操作', key: 'a', width: 100, render: (r: any) => h(NSpace, { size: 4 }, () => [
    h(NButton, { size: 'tiny', onClick: () => openForm(r) }, () => '编辑'),
    h(NButton, { size: 'tiny', type: 'error', onClick: () => del(r.id) }, () => '删除'),
  ]) },
]

function openForm(row?: any) {
  form.value = row ? { ...row } : emptyForm()
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
  if (statusFilter.value) params.set('status', statusFilter.value)
  const qs = params.toString()
  items.value = ((await http.get(`/vehicles${qs ? '?' + qs : ''}`)) as any).data || []
}

async function save() {
  if (form.value.id) {
    await http.put(`/vehicles/${form.value.id}`, form.value)
  } else {
    await http.post('/vehicles', form.value)
  }
  showForm.value = false; load()
}
async function del(id: number) { await http.delete(`/vehicles/${id}`); load() }

onMounted(async () => { await loadDict(); load() })
</script>
