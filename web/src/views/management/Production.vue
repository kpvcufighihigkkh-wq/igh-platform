<template>
  <div>
    <n-card>
      <n-tabs type="line" v-model:value="activeTab">
        <n-tab-pane name="lots" tab="批次管理">
          <n-space justify="space-between" style="margin-bottom: 12px">
            <n-space>
              <n-select v-model:value="lotStatus" :options="lotStatusOptions" style="width: 120px" clearable placeholder="状态" />
              <n-tag v-if="batchSource" :type="batchSource === 'manual' ? 'default' : 'info'" size="small">来源: {{ tSource(batchSource) }}</n-tag>
            </n-space>
            <n-button type="primary" @click="openCreate">新建批次</n-button>
          </n-space>
          <n-data-table :columns="lotColumns" :data="lots" :bordered="false" :loading="loading" />
          <n-pagination v-model:page="lotPage" :page-count="Math.ceil(lotTotal / 20)" style="margin-top: 12px; justify-content: flex-end" @update:page="fetchLots" />
        </n-tab-pane>

        <n-tab-pane name="doffings" tab="落纱记录">
          <n-space style="margin-bottom: 12px">
            <n-select v-model:value="doffLotId" :options="lotOptions" style="width: 200px" clearable placeholder="按批次筛选" />
          </n-space>
          <n-data-table :columns="doffColumns" :data="doffings" :bordered="false" />
          <n-pagination v-model:page="doffPage" :page-count="Math.ceil(doffTotal / 20)" style="margin-top: 12px; justify-content: flex-end" @update:page="fetchDoffings" />
        </n-tab-pane>

        <n-tab-pane name="monitor" tab="产线监控">
          <n-space style="margin-bottom: 12px" align="center">
            <span style="font-size: 13px; color: var(--steel)">选择产线:</span>
            <n-select v-model:value="monitorLine" :options="windingLineOpts" style="width: 180px" @update:value="fetchWinderStatus" />
            <n-button size="small" @click="fetchWinderStatus" :loading="monitorLoading">刷新</n-button>
          </n-space>
          <div class="winder-grid">
            <div v-for="w in winders" :key="w.code" class="winder-card"
                 :class="{ selected: selectedWinder?.code === w.code }" @click="selectWinder(w)">
              <div class="winder-header">
                <span class="winder-name">{{ w.name }}</span>
                <n-tag :type="winderTagType(w.status)" size="small">{{ tStatus(w.status) }}</n-tag>
              </div>
              <div class="winder-lot" v-if="w.active_lot_code">
                <span class="lot-label">批次:</span> {{ w.active_lot_code }}
                <span class="lot-spec">{{ w.active_lot_spec }}</span>
              </div>
              <div class="winder-lot" v-else>
                <span style="color: var(--steel); font-size: 12px">暂无生产</span>
              </div>
              <div class="winder-stats">
                <div><span class="stat-num">{{ w.today_doff_count }}</span><span class="stat-lbl">落纱</span></div>
                <div><span class="stat-num">{{ w.today_bobbin_count }}</span><span class="stat-lbl">丝饼</span></div>
              </div>
              <div class="winder-time">{{ w.last_doffing_time ? formatTime(w.last_doffing_time) : '暂无记录' }}</div>
            </div>
          </div>
          <n-empty v-if="!monitorLoading && !winders.length" description="该产线暂无卷绕机" style="margin-top: 40px" />
        </n-tab-pane>
      </n-tabs>
    </n-card>

    <!-- Create / Edit Lot Modal -->
    <n-modal v-model:show="showModal" preset="card" :title="editingId ? '编辑批次' : '新建批次'" style="width: 720px">
      <n-tabs type="line" v-model:value="formTab">
        <n-tab-pane name="basic" tab="基本信息">
          <n-form :model="lotForm" label-placement="left" label-width="100">
            <n-grid :cols="2" :x-gap="12">
              <n-gi><n-form-item label="批号"><n-input v-model:value="lotForm.code" /></n-form-item></n-gi>
              <n-gi><n-form-item label="名称"><n-input v-model:value="lotForm.name" /></n-form-item></n-gi>
              <n-gi><n-form-item label="前缀"><n-input v-model:value="lotForm.prefix" /></n-form-item></n-gi>
              <n-gi>
                <n-form-item label="产品类型">
                  <n-select v-model:value="lotForm.product_type" :options="productTypeOpts" />
                </n-form-item>
              </n-gi>
              <n-gi><n-form-item label="规格"><n-input v-model:value="lotForm.specification" placeholder="如 40D/34F" /></n-form-item></n-gi>
              <n-gi><n-form-item label="出口规格"><n-input v-model:value="lotForm.spec_export" /></n-form-item></n-gi>
              <n-gi :span="2"><n-form-item label="订单号"><n-input v-model:value="lotForm.order_code" /></n-form-item></n-gi>
            </n-grid>
          </n-form>
        </n-tab-pane>

        <n-tab-pane name="process" tab="工艺参数">
          <n-form :model="lotForm" label-placement="left" label-width="100">
            <n-grid :cols="2" :x-gap="12">
              <n-gi><n-form-item label="国内标准"><n-input v-model:value="lotForm.standard_china" /></n-form-item></n-gi>
              <n-gi><n-form-item label="出口标准"><n-input v-model:value="lotForm.standard_export" /></n-form-item></n-gi>
              <n-gi><n-form-item label="光泽"><n-input v-model:value="lotForm.lustre" /></n-form-item></n-gi>
              <n-gi><n-form-item label="捻度"><n-input v-model:value="lotForm.twist" /></n-form-item></n-gi>
              <n-gi><n-form-item label="纸管颜色"><n-input v-model:value="lotForm.paper_tube_color" /></n-form-item></n-gi>
              <n-gi><n-form-item label="箱重(kg)"><n-input-number v-model:value="lotForm.box_weight" :precision="3" clearable /></n-form-item></n-gi>
              <n-gi><n-form-item label="等待时间(min)"><n-input-number v-model:value="lotForm.wait_time_min" :min="0" /></n-form-item></n-gi>
            </n-grid>
          </n-form>
        </n-tab-pane>

        <n-tab-pane name="pallet" tab="码垛配置">
          <n-form :model="lotForm" label-placement="left" label-width="120">
            <n-grid :cols="2" :x-gap="12">
              <n-gi><n-form-item label="默认码垛层数"><n-input-number v-model:value="lotForm.default_pallet_layers" :min="1" :max="20" /></n-form-item></n-gi>
              <n-gi><n-form-item label="默认码垛重量"><n-input-number v-model:value="lotForm.default_pallet_size" :min="1" /></n-form-item></n-gi>
              <n-gi><n-form-item label="最小托盘数"><n-input-number v-model:value="lotForm.min_pallets_for_order" :min="0" /></n-form-item></n-gi>
              <n-gi><n-form-item label="最大托盘数"><n-input-number v-model:value="lotForm.max_pallets_for_order" :min="0" /></n-form-item></n-gi>
            </n-grid>
          </n-form>
        </n-tab-pane>

        <n-tab-pane name="weights" tab="重量规格">
          <n-data-table :columns="weightCols" :data="weightRows" :bordered="false" size="small" />
        </n-tab-pane>

        <n-tab-pane name="grades" tab="等级范围">
          <n-space style="margin-bottom: 8px">
            <n-button size="small" @click="addGradeRow">添加行</n-button>
          </n-space>
          <n-data-table :columns="gradeCols" :data="gradeRows" :bordered="false" size="small" />
        </n-tab-pane>
      </n-tabs>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showModal = false">取消</n-button>
          <n-button type="primary" @click="saveLot" :loading="saving">{{ editingId ? '保存' : '创建' }}</n-button>
        </n-space>
      </template>
    </n-modal>

    <!-- Detail Drawer -->
    <n-drawer v-model:show="showDetail" :width="500" placement="right">
      <n-drawer-content :title="detailLot?.code || '批次详情'">
        <template v-if="detailLot">
          <n-descriptions :column="2" size="small" bordered>
            <n-descriptions-item label="批号">{{ detailLot.code }}</n-descriptions-item>
            <n-descriptions-item label="名称">{{ detailLot.name || '-' }}</n-descriptions-item>
            <n-descriptions-item label="前缀">{{ detailLot.prefix || '-' }}</n-descriptions-item>
            <n-descriptions-item label="产品类型">{{ detailLot.product_type }}</n-descriptions-item>
            <n-descriptions-item label="规格">{{ detailLot.specification }}</n-descriptions-item>
            <n-descriptions-item label="出口规格">{{ detailLot.spec_export || '-' }}</n-descriptions-item>
            <n-descriptions-item label="订单号">{{ detailLot.order_code || '-' }}</n-descriptions-item>
            <n-descriptions-item label="来源">{{ tSource(detailLot.source) }}</n-descriptions-item>
            <n-descriptions-item label="国内标准">{{ detailLot.standard_china || '-' }}</n-descriptions-item>
            <n-descriptions-item label="出口标准">{{ detailLot.standard_export || '-' }}</n-descriptions-item>
            <n-descriptions-item label="光泽">{{ detailLot.lustre || '-' }}</n-descriptions-item>
            <n-descriptions-item label="捻度">{{ detailLot.twist || '-' }}</n-descriptions-item>
            <n-descriptions-item label="纸管颜色">{{ detailLot.paper_tube_color || '-' }}</n-descriptions-item>
            <n-descriptions-item label="箱重">{{ detailLot.box_weight ?? '-' }}</n-descriptions-item>
            <n-descriptions-item label="等待时间">{{ detailLot.wait_time_min }}min</n-descriptions-item>
            <n-descriptions-item label="码垛层数">{{ detailLot.default_pallet_layers }}</n-descriptions-item>
            <n-descriptions-item label="码垛重量">{{ detailLot.default_pallet_size }}</n-descriptions-item>
            <n-descriptions-item label="托盘数范围">{{ detailLot.min_pallets_for_order }} ~ {{ detailLot.max_pallets_for_order }}</n-descriptions-item>
            <n-descriptions-item label="状态">
              <n-tag :type="detailLot.status === 'active' ? 'success' : 'default'" size="small">{{ tStatus(detailLot.status) }}</n-tag>
            </n-descriptions-item>
            <n-descriptions-item label="结批">
              <n-tag :type="detailLot.end_lot ? 'error' : 'default'" size="small">{{ detailLot.end_lot ? '已结批' : '未结批' }}</n-tag>
            </n-descriptions-item>
            <n-descriptions-item label="包装锁定">
              <n-tag :type="detailLot.packing_lock ? 'warning' : 'default'" size="small">{{ detailLot.packing_lock ? '已锁定' : '未锁定' }}</n-tag>
            </n-descriptions-item>
          </n-descriptions>

          <n-divider>重量规格</n-divider>
          <n-data-table v-if="detailLot.weights?.length" :columns="detailWeightCols" :data="detailLot.weights" :bordered="false" size="small" />
          <n-empty v-else description="未配置" size="small" />

          <n-divider>等级范围</n-divider>
          <n-data-table v-if="detailLot.grade_ranges?.length" :columns="detailGradeCols" :data="detailLot.grade_ranges" :bordered="false" size="small" />
          <n-empty v-else description="未配置" size="small" />

          <n-divider />
          <n-space>
            <n-button size="small" @click="openEdit(detailLot)">编辑</n-button>
            <n-button size="small" type="error" :disabled="detailLot.end_lot" @click="endLot(detailLot.id)">结批</n-button>
            <n-button size="small" type="warning" @click="toggleLock(detailLot.id)">{{ detailLot.packing_lock ? '解除锁定' : '包装锁定' }}</n-button>
          </n-space>
        </template>
      </n-drawer-content>
    </n-drawer>
    <!-- Winder Detail Drawer -->
    <n-drawer v-model:show="showWinderDetail" :width="480" placement="right">
      <n-drawer-content :title="selectedWinder?.name || '卷绕机详情'">
        <template v-if="selectedWinder">
          <n-descriptions :column="2" size="small" bordered>
            <n-descriptions-item label="编号">{{ selectedWinder.code }}</n-descriptions-item>
            <n-descriptions-item label="状态">
              <n-tag :type="winderTagType(selectedWinder.status)" size="small">{{ tStatus(selectedWinder.status) }}</n-tag>
            </n-descriptions-item>
            <n-descriptions-item label="当前批次">{{ selectedWinder.active_lot_code || '-' }}</n-descriptions-item>
            <n-descriptions-item label="规格">{{ selectedWinder.active_lot_spec || '-' }}</n-descriptions-item>
            <n-descriptions-item label="今日落纱">{{ selectedWinder.today_doff_count }}</n-descriptions-item>
            <n-descriptions-item label="今日丝饼">{{ selectedWinder.today_bobbin_count }}</n-descriptions-item>
            <n-descriptions-item label="最后落纱时间" :span="2">{{ selectedWinder.last_doffing_time ? formatTime(selectedWinder.last_doffing_time) : '-' }}</n-descriptions-item>
          </n-descriptions>
          <n-divider>最近落纱记录</n-divider>
          <n-data-table :columns="winderDoffCols" :data="winderDoffings" :bordered="false" size="small" :loading="winderDoffLoading" />
        </template>
      </n-drawer-content>
    </n-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch, h, computed } from 'vue'
import { NTag, NButton, NInputNumber, NInput, NSelect, useMessage } from 'naive-ui'
import http from '../../api/http'
import { tStatus, tSource, formatTime } from '../../utils/status'

const msg = useMessage()
const activeTab = ref('lots')
const loading = ref(false)
const saving = ref(false)

// --- Batch source ---
const batchSource = ref('')
async function fetchBatchSource() {
  try {
    const res: any = await http.get('/admin/settings?category=production')
    const items = res.data?.items || res.data || []
    const s = items.find((i: any) => i.key === 'batch_source')
    batchSource.value = s?.value || 'manual'
  } catch { batchSource.value = 'manual' }
}

// --- Product types ---
const productTypes = ref<any[]>([])
const productTypeOpts = computed(() => productTypes.value.map(t => ({ label: `${t.code} - ${t.name}`, value: t.code })))
async function fetchProductTypes() {
  try {
    const res: any = await http.get('/admin/product-types')
    productTypes.value = res.data || []
  } catch { productTypes.value = [] }
}

// --- Lots ---
const lots = ref<any[]>([])
const lotTotal = ref(0)
const lotPage = ref(1)
const lotStatus = ref<string | null>(null)

const lotStatusOptions = [
  { label: '活跃', value: 'active' },
  { label: '已完成', value: 'completed' },
]

const lotColumns = [
  { title: 'ID', key: 'id', width: 60 },
  { title: '批号', key: 'code' },
  { title: '名称', key: 'name', ellipsis: { tooltip: true } },
  { title: '规格', key: 'specification' },
  { title: '类型', key: 'product_type', width: 60 },
  { title: '来源', key: 'source', width: 60, render: (r: any) => h(NTag, { type: r.source === 'manual' ? 'default' : 'info', size: 'small', bordered: false }, () => tSource(r.source || 'manual')) },
  { title: '结批', key: 'end_lot', width: 60, render: (r: any) => r.end_lot ? h(NTag, { type: 'error', size: 'small', bordered: false }, () => '是') : '-' },
  { title: '状态', key: 'status', width: 80, render: (r: any) => h(NTag, { type: r.status === 'active' ? 'success' : 'default', size: 'small' }, () => tStatus(r.status)) },
  { title: '操作', key: 'actions', width: 80, render: (r: any) => h(NButton, { text: true, type: 'info', size: 'small', onClick: () => openDetail(r.id) }, () => '详情') },
]

async function fetchLots() {
  loading.value = true
  const params = new URLSearchParams({ page: String(lotPage.value), page_size: '20' })
  if (lotStatus.value) params.set('status', lotStatus.value)
  const res: any = await http.get(`/lots?${params}`)
  lots.value = res.data?.items || []
  lotTotal.value = res.data?.total || 0
  loading.value = false
}

// --- Modal (create / edit) ---
const showModal = ref(false)
const formTab = ref('basic')
const editingId = ref<number | null>(null)

function emptyForm() {
  return {
    code: '', name: '', prefix: '', product_type: 'FDY', specification: '', spec_export: '',
    order_code: '', standard_china: '', standard_export: '', lustre: '', twist: '',
    paper_tube_color: '', box_weight: null as number | null, wait_time_min: 60,
    default_pallet_layers: 6, default_pallet_size: 8,
    min_pallets_for_order: 0, max_pallets_for_order: 0,
  }
}
const lotForm = ref(emptyForm())

const weightRows = ref([
  { level: 6, gross_weight: 0, net_weight: 0 },
  { level: 7, gross_weight: 0, net_weight: 0 },
  { level: 8, gross_weight: 0, net_weight: 0 },
  { level: 9, gross_weight: 0, net_weight: 0 },
])

const gradeRows = ref<{ grade_code: string; min_weight: number; max_weight: number }[]>([])

const gradeOptions = [
  { label: 'AAA', value: 'AAA' }, { label: 'AA', value: 'AA' },
  { label: 'AA1', value: 'AA1' }, { label: 'AA2', value: 'AA2' },
  { label: 'A1', value: 'A1' }, { label: 'A', value: 'A' },
]

const weightCols = [
  { title: '层数', key: 'level', width: 60 },
  { title: '毛重(kg)', key: 'gross_weight', render: (r: any, idx: number) => h(NInputNumber, { value: r.gross_weight, precision: 3, size: 'small', onUpdateValue: (v: number | null) => { weightRows.value[idx].gross_weight = v ?? 0 } }) },
  { title: '净重(kg)', key: 'net_weight', render: (r: any, idx: number) => h(NInputNumber, { value: r.net_weight, precision: 3, size: 'small', onUpdateValue: (v: number | null) => { weightRows.value[idx].net_weight = v ?? 0 } }) },
]

const gradeCols = [
  { title: '等级', key: 'grade_code', width: 100, render: (r: any, idx: number) => h(NSelect, { value: r.grade_code, options: gradeOptions, size: 'small', onUpdateValue: (v: string) => { gradeRows.value[idx].grade_code = v } }) },
  { title: '最小重量(kg)', key: 'min_weight', render: (r: any, idx: number) => h(NInputNumber, { value: r.min_weight, precision: 3, size: 'small', onUpdateValue: (v: number | null) => { gradeRows.value[idx].min_weight = v ?? 0 } }) },
  { title: '最大重量(kg)', key: 'max_weight', render: (r: any, idx: number) => h(NInputNumber, { value: r.max_weight, precision: 3, size: 'small', onUpdateValue: (v: number | null) => { gradeRows.value[idx].max_weight = v ?? 0 } }) },
  { title: '', key: 'op', width: 50, render: (_: any, idx: number) => h(NButton, { text: true, type: 'error', size: 'small', onClick: () => gradeRows.value.splice(idx, 1) }, () => '删除') },
]

function addGradeRow() {
  gradeRows.value.push({ grade_code: 'AA', min_weight: 0, max_weight: 0 })
}

function openCreate() {
  editingId.value = null
  lotForm.value = emptyForm()
  weightRows.value = [
    { level: 6, gross_weight: 0, net_weight: 0 },
    { level: 7, gross_weight: 0, net_weight: 0 },
    { level: 8, gross_weight: 0, net_weight: 0 },
    { level: 9, gross_weight: 0, net_weight: 0 },
  ]
  gradeRows.value = []
  formTab.value = 'basic'
  showModal.value = true
}

function openEdit(lot: any) {
  editingId.value = lot.id
  lotForm.value = {
    code: lot.code, name: lot.name || '', prefix: lot.prefix || '',
    product_type: lot.product_type, specification: lot.specification,
    spec_export: lot.spec_export || '', order_code: lot.order_code || '',
    standard_china: lot.standard_china || '', standard_export: lot.standard_export || '',
    lustre: lot.lustre || '', twist: lot.twist || '', paper_tube_color: lot.paper_tube_color || '',
    box_weight: lot.box_weight ?? null, wait_time_min: lot.wait_time_min || 60,
    default_pallet_layers: lot.default_pallet_layers || 6, default_pallet_size: lot.default_pallet_size || 8,
    min_pallets_for_order: lot.min_pallets_for_order || 0, max_pallets_for_order: lot.max_pallets_for_order || 0,
  }
  weightRows.value = [6, 7, 8, 9].map(level => {
    const w = (lot.weights || []).find((x: any) => x.level === level)
    return { level, gross_weight: w?.gross_weight || 0, net_weight: w?.net_weight || 0 }
  })
  gradeRows.value = (lot.grade_ranges || []).map((g: any) => ({ grade_code: g.grade_code, min_weight: g.min_weight, max_weight: g.max_weight }))
  formTab.value = 'basic'
  showModal.value = true
  showDetail.value = false
}

async function saveLot() {
  saving.value = true
  const weights = weightRows.value.filter(w => w.gross_weight > 0 || w.net_weight > 0)
  const body = { lot: lotForm.value, weights, grade_ranges: gradeRows.value }
  try {
    if (editingId.value) {
      await http.put(`/lots/${editingId.value}`, body)
      msg.success('更新成功')
    } else {
      await http.post('/lots', body)
      msg.success('创建成功')
    }
    showModal.value = false
    fetchLots()
  } catch (e: any) {
    msg.error(e?.message || '操作失败')
  }
  saving.value = false
}

// --- Detail Drawer ---
const showDetail = ref(false)
const detailLot = ref<any>(null)

const detailWeightCols = [
  { title: '层数', key: 'level', width: 60 },
  { title: '毛重(kg)', key: 'gross_weight' },
  { title: '净重(kg)', key: 'net_weight' },
]
const detailGradeCols = [
  { title: '等级', key: 'grade_code', width: 80 },
  { title: '最小重量(kg)', key: 'min_weight' },
  { title: '最大重量(kg)', key: 'max_weight' },
]

async function openDetail(id: number) {
  const res: any = await http.get(`/lots/${id}`)
  detailLot.value = res.data
  showDetail.value = true
}

async function endLot(id: number) {
  await http.post(`/lots/${id}/end`)
  msg.success('已结批')
  openDetail(id)
  fetchLots()
}

async function toggleLock(id: number) {
  await http.post(`/lots/${id}/packing-lock`)
  msg.success('已切换锁定状态')
  openDetail(id)
  fetchLots()
}

// --- Doffings ---
const doffings = ref<any[]>([])
const doffTotal = ref(0)
const doffPage = ref(1)
const doffLotId = ref<number | null>(null)
const lotOptions = computed(() => lots.value.map(l => ({ label: l.code, value: l.id })))

const doffColumns = [
  { title: 'ID', key: 'id', width: 60 },
  { title: '产线', key: 'line_id', width: 60 },
  { title: '卷绕机', key: 'winder_name' },
  { title: '落纱号', key: 'doff_no', width: 80 },
  { title: '丝饼数', key: 'bobbin_count', width: 80 },
  { title: '时间', key: 'created_at', width: 160, render: (r: any) => new Date(r.created_at).toLocaleString('zh-CN') },
]

async function fetchDoffings() {
  const params = new URLSearchParams({ page: String(doffPage.value), page_size: '20' })
  if (doffLotId.value) params.set('lot_id', String(doffLotId.value))
  const res: any = await http.get(`/doffings?${params}`)
  doffings.value = res.data?.items || []
  doffTotal.value = res.data?.total || 0
}

// --- Line Monitor ---
const windingLines = ref<any[]>([])
const monitorLine = ref<string | null>(null)
const monitorLoading = ref(false)
const winders = ref<any[]>([])
const selectedWinder = ref<any>(null)
const showWinderDetail = ref(false)
const winderDoffings = ref<any[]>([])
const winderDoffLoading = ref(false)

const windingLineOpts = computed(() => windingLines.value.map(l => ({ label: l.name, value: l.code })))

function winderTagType(status: string) {
  if (status === 'running') return 'success'
  if (status === 'fault') return 'error'
  if (status === 'offline') return 'default'
  return 'warning'
}

async function fetchWindingLines() {
  const res: any = await http.get('/admin/production-lines')
  windingLines.value = (res.data || []).filter((l: any) => l.station_type === 'winding')
  if (windingLines.value.length && !monitorLine.value) {
    monitorLine.value = windingLines.value[0].code
  }
}

async function fetchWinderStatus() {
  if (!monitorLine.value) return
  monitorLoading.value = true
  try {
    const res: any = await http.get(`/winder-status?line_code=${monitorLine.value}`)
    winders.value = res.data || []
  } catch { winders.value = [] }
  monitorLoading.value = false
}

async function selectWinder(w: any) {
  selectedWinder.value = w
  showWinderDetail.value = true
  winderDoffLoading.value = true
  try {
    const res: any = await http.get(`/winder-doffings?winder_name=${w.code}&limit=10`)
    winderDoffings.value = res.data || []
  } catch { winderDoffings.value = [] }
  winderDoffLoading.value = false
}

const winderDoffCols = [
  { title: '落纱号', key: 'doff_no', width: 70 },
  { title: '丝饼数', key: 'bobbin_count', width: 70 },
  { title: '时间', key: 'created_at', render: (r: any) => formatTime(r.created_at) },
]

watch(lotStatus, () => { lotPage.value = 1; fetchLots() })
watch(doffLotId, () => { doffPage.value = 1; fetchDoffings() })
watch(activeTab, (tab) => {
  if (tab === 'doffings') fetchDoffings()
  if (tab === 'monitor') fetchWindingLines().then(() => fetchWinderStatus())
})
onMounted(() => { fetchLots(); fetchBatchSource(); fetchProductTypes() })
</script>

<style scoped>
.winder-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  gap: 12px;
}
.winder-card {
  background: var(--plate);
  border: 1px solid var(--line);
  border-radius: 2px;
  padding: 14px;
  cursor: pointer;
  transition: border-color 0.2s;
}
.winder-card:hover, .winder-card.selected { border-color: var(--igh-red); }
.winder-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px; }
.winder-name { font-family: var(--font-display); font-weight: 600; font-size: 15px; }
.winder-lot { font-size: 12px; color: var(--graphite); margin-bottom: 8px; }
.lot-label { color: var(--steel); }
.lot-spec { color: var(--steel); margin-left: 8px; }
.winder-stats { display: flex; gap: 20px; margin-bottom: 6px; }
.stat-num { font-family: var(--font-display); font-size: 24px; font-weight: 600; color: var(--graphite); font-variant-numeric: tabular-nums; }
.stat-lbl { font-size: 11px; color: var(--steel); margin-left: 4px; }
.winder-time { font-size: 11px; color: var(--steel-light); }
</style>
