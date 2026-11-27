<template>
  <div>
    <n-card>
      <n-tabs type="line" v-model:value="activeTab">
        <n-tab-pane name="picking" tab="拣选订单">
          <n-space justify="space-between" style="margin-bottom: 12px">
            <n-select v-model:value="pickStatus" :options="statusOpts" style="width: 120px" clearable placeholder="状态" />
            <n-button type="primary" @click="showPickModal = true">新建拣选单</n-button>
          </n-space>
          <n-data-table :columns="pickColumns" :data="pickOrders" :bordered="false" :loading="loading" />
          <n-pagination v-model:page="pickPage" :page-count="Math.ceil(pickTotal / 20)" style="margin-top: 12px; justify-content: flex-end" @update:page="fetchPickOrders" />
        </n-tab-pane>

        <n-tab-pane name="packing" tab="打包订单">
          <n-space justify="space-between" style="margin-bottom: 12px">
            <n-select v-model:value="packStatus" :options="statusOpts" style="width: 120px" clearable placeholder="状态" />
            <n-button type="primary" @click="showPackModal = true">新建打包单</n-button>
          </n-space>
          <n-data-table :columns="packColumns" :data="packOrders" :bordered="false" />
          <n-pagination v-model:page="packPage" :page-count="Math.ceil(packTotal / 20)" style="margin-top: 12px; justify-content: flex-end" @update:page="fetchPackOrders" />
        </n-tab-pane>

        <n-tab-pane name="pallets" tab="托盘列表">
          <n-data-table :columns="palletColumns" :data="pallets" :bordered="false" />
          <n-pagination v-model:page="palletPage" :page-count="Math.ceil(palletTotal / 20)" style="margin-top: 12px; justify-content: flex-end" @update:page="fetchPallets" />
        </n-tab-pane>
      </n-tabs>
    </n-card>

    <n-modal v-model:show="showPickModal" preset="card" title="新建拣选单" style="width: 420px">
      <n-form :model="pickForm" label-placement="left" label-width="80">
        <n-form-item label="批次ID"><n-input-number v-model:value="pickForm.lot_id" style="width: 100%" /></n-form-item>
        <n-form-item label="仓库ID"><n-input-number v-model:value="pickForm.warehouse_id" style="width: 100%" /></n-form-item>
        <n-form-item label="等级"><n-input v-model:value="pickForm.grade_code" placeholder="AA" /></n-form-item>
        <n-form-item label="数量"><n-input-number v-model:value="pickForm.quantity" style="width: 100%" /></n-form-item>
        <n-button type="primary" block @click="createPickOrder">创建</n-button>
      </n-form>
    </n-modal>

    <n-modal v-model:show="showPackModal" preset="card" title="新建打包单" style="width: 420px">
      <n-form :model="packForm" label-placement="left" label-width="80">
        <n-form-item label="批次ID"><n-input-number v-model:value="packForm.lot_id" style="width: 100%" /></n-form-item>
        <n-form-item label="码垛机ID"><n-input-number v-model:value="packForm.palletizer_id" style="width: 100%" /></n-form-item>
        <n-form-item label="等级"><n-input v-model:value="packForm.grade_code" placeholder="AA" /></n-form-item>
        <n-form-item label="目标数量"><n-input-number v-model:value="packForm.target_quantity" style="width: 100%" /></n-form-item>
        <n-button type="primary" block @click="createPackOrder">创建</n-button>
      </n-form>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch, h } from 'vue'
import { NTag, NButton } from 'naive-ui'
import http from '../../api/http'
import { tStatus } from '../../utils/status'

const activeTab = ref('picking')
const loading = ref(false)

const statusOpts = [
  { label: '待处理', value: 'pending' },
  { label: '进行中', value: 'in_progress' },
  { label: '已完成', value: 'completed' },
]

// --- Picking Orders ---
const pickOrders = ref<any[]>([])
const pickTotal = ref(0)
const pickPage = ref(1)
const pickStatus = ref<string | null>(null)
const showPickModal = ref(false)
const pickForm = ref({ lot_id: 1, warehouse_id: 1, grade_code: 'AA', quantity: 48 })

const pickColumns = [
  { title: 'ID', key: 'id', width: 60 },
  { title: '批次', key: 'lot_id', width: 60 },
  { title: '仓库', key: 'warehouse_id', width: 60 },
  { title: '等级', key: 'grade_code', width: 60 },
  { title: '数量', key: 'quantity', width: 60 },
  { title: '状态', key: 'status', width: 80, render: (r: any) => h(NTag, { type: r.status === 'completed' ? 'success' : r.status === 'in_progress' ? 'warning' : 'default', size: 'small' }, () => tStatus(r.status)) },
  { title: '创建时间', key: 'created_at', width: 160, render: (r: any) => new Date(r.created_at).toLocaleString('zh-CN') },
  { title: '操作', key: 'actions', width: 140, render: (r: any) => {
    const btns: any[] = []
    if (r.status === 'pending') btns.push(h(NButton, { size: 'small', type: 'info', onClick: () => startPick(r.id) }, () => '开始'))
    if (r.status === 'in_progress') btns.push(h(NButton, { size: 'small', type: 'success', onClick: () => completePick(r.id) }, () => '完成'))
    return btns
  }},
]

async function fetchPickOrders() {
  loading.value = true
  const params = new URLSearchParams({ page: String(pickPage.value), page_size: '20' })
  if (pickStatus.value) params.set('status', pickStatus.value)
  const res: any = await http.get(`/picking-orders?${params}`)
  pickOrders.value = res.data?.items || []
  pickTotal.value = res.data?.total || 0
  loading.value = false
}

async function createPickOrder() {
  await http.post('/picking-orders', pickForm.value)
  showPickModal.value = false
  fetchPickOrders()
}

async function startPick(id: number) {
  await http.put(`/picking-orders/${id}/start`)
  fetchPickOrders()
}

async function completePick(id: number) {
  await http.put(`/picking-orders/${id}/complete`)
  fetchPickOrders()
}

// --- Packing Orders ---
const packOrders = ref<any[]>([])
const packTotal = ref(0)
const packPage = ref(1)
const packStatus = ref<string | null>(null)
const showPackModal = ref(false)
const packForm = ref({ lot_id: 1, palletizer_id: 1, grade_code: 'AA', target_quantity: 48 })

const packColumns = [
  { title: 'ID', key: 'id', width: 60 },
  { title: '批次', key: 'lot_id', width: 60 },
  { title: '等级', key: 'grade_code', width: 60 },
  { title: '目标', key: 'target_quantity', width: 60 },
  { title: '已打包', key: 'packed_quantity', width: 70 },
  { title: '状态', key: 'status', width: 80, render: (r: any) => h(NTag, { type: r.status === 'completed' ? 'success' : r.status === 'in_progress' ? 'warning' : 'default', size: 'small' }, () => tStatus(r.status)) },
  { title: '创建时间', key: 'created_at', width: 160, render: (r: any) => new Date(r.created_at).toLocaleString('zh-CN') },
  { title: '操作', key: 'actions', width: 80, render: (r: any) => {
    if (r.status === 'pending') return h(NButton, { size: 'small', type: 'info', onClick: () => startPack(r.id) }, () => '开始')
    return null
  }},
]

async function fetchPackOrders() {
  const params = new URLSearchParams({ page: String(packPage.value), page_size: '20' })
  if (packStatus.value) params.set('status', packStatus.value)
  const res: any = await http.get(`/packing-orders?${params}`)
  packOrders.value = res.data?.items || []
  packTotal.value = res.data?.total || 0
}

async function createPackOrder() {
  await http.post('/packing-orders', packForm.value)
  showPackModal.value = false
  fetchPackOrders()
}

async function startPack(id: number) {
  await http.put(`/packing-orders/${id}/start`)
  fetchPackOrders()
}

// --- Pallets ---
const pallets = ref<any[]>([])
const palletTotal = ref(0)
const palletPage = ref(1)

const palletColumns = [
  { title: 'ID', key: 'id', width: 60 },
  { title: '托盘号', key: 'code' },
  { title: '批次', key: 'lot_id', width: 60 },
  { title: '等级', key: 'grade_code', width: 60 },
  { title: '层数', key: 'layers', width: 60 },
  { title: '丝饼数', key: 'bobbin_count', width: 70 },
  { title: '总重量', key: 'total_weight', width: 80, render: (r: any) => r.total_weight ? `${r.total_weight}kg` : '-' },
  { title: 'ERP同步', key: 'erp_synced', width: 80, render: (r: any) => h(NTag, { type: r.erp_synced ? 'success' : 'default', size: 'small' }, () => r.erp_synced ? '已同步' : '未同步') },
  { title: '创建时间', key: 'created_at', width: 160, render: (r: any) => new Date(r.created_at).toLocaleString('zh-CN') },
]

async function fetchPallets() {
  const params = new URLSearchParams({ page: String(palletPage.value), page_size: '20' })
  const res: any = await http.get(`/pallets?${params}`)
  pallets.value = res.data?.items || []
  palletTotal.value = res.data?.total || 0
}

watch(pickStatus, () => { pickPage.value = 1; fetchPickOrders() })
watch(packStatus, () => { packPage.value = 1; fetchPackOrders() })
watch(activeTab, (tab) => {
  if (tab === 'packing') fetchPackOrders()
  if (tab === 'pallets') fetchPallets()
})
onMounted(fetchPickOrders)
</script>
