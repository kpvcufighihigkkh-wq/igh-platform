<template>
  <div>
    <n-card>
      <n-tabs type="line" v-model:value="activeTab">
        <n-tab-pane name="equipment" tab="设备台账">
          <n-space style="margin-bottom:12px">
            <n-select v-model:value="eqType" :options="typeOpts" style="width:140px" clearable placeholder="设备类型" />
          </n-space>
          <n-data-table :columns="eqColumns" :data="equipment" :bordered="false" :loading="loading" size="small" />
        </n-tab-pane>

        <n-tab-pane name="plans" tab="保养计划">
          <n-space justify="space-between" style="margin-bottom:12px">
            <n-space>
              <n-tag type="error" v-if="overdue.length">逾期: {{ overdue.length }}</n-tag>
              <n-tag type="warning" v-if="upcoming.length">即将到期: {{ upcoming.length }}</n-tag>
            </n-space>
            <n-button type="primary" size="small" @click="openPlanForm()">新增计划</n-button>
          </n-space>
          <n-data-table :columns="planColumns" :data="plans" :bordered="false" size="small" />
        </n-tab-pane>

        <n-tab-pane name="records" tab="保养记录">
          <n-space justify="space-between" style="margin-bottom:12px">
            <n-select v-model:value="recordEqId" :options="eqOpts" style="width:200px" clearable placeholder="按设备筛选" @update:value="fetchRecords" />
            <n-button type="primary" size="small" @click="openRecordForm()">录入记录</n-button>
          </n-space>
          <n-data-table :columns="recordColumns" :data="records" :bordered="false" size="small" />
          <n-pagination v-model:page="recordPage" :page-count="Math.ceil(recordTotal / 20)" style="margin-top:12px;justify-content:flex-end" @update:page="fetchRecords" />
        </n-tab-pane>
      </n-tabs>
    </n-card>

    <!-- 新增保养计划 -->
    <n-modal v-model:show="showPlanModal" preset="card" :title="planForm.id?'编辑保养计划':'新增保养计划'" style="width:480px">
      <n-form :model="planForm" label-placement="left" label-width="80" size="small">
        <n-form-item label="设备"><n-select v-model:value="planForm.equipment_id" :options="eqOpts" placeholder="选择设备" /></n-form-item>
        <n-form-item label="计划名称"><n-input v-model:value="planForm.name" placeholder="如：月度润滑保养" /></n-form-item>
        <n-form-item label="触发方式"><n-select v-model:value="planForm.trigger_type" :options="triggerOpts" /></n-form-item>
        <n-form-item label="间隔(小时)" v-if="planForm.trigger_type==='interval'"><n-input-number v-model:value="planForm.interval_hours" :min="1" style="width:100%" /></n-form-item>
        <n-form-item label="计数阈值" v-if="planForm.trigger_type==='counter'"><n-input-number v-model:value="planForm.counter_threshold" :min="1" style="width:100%" /></n-form-item>
        <n-form-item label="提前预警"><n-input-number v-model:value="planForm.warn_before_hours" :min="0" style="width:100%"><template #suffix>小时</template></n-input-number></n-form-item>
        <n-button type="primary" block @click="savePlan">保存</n-button>
      </n-form>
    </n-modal>

    <!-- 录入保养记录 -->
    <n-modal v-model:show="showRecordModal" preset="card" title="录入保养记录" style="width:480px">
      <n-form :model="recordForm" label-placement="left" label-width="80" size="small">
        <n-form-item label="关联计划"><n-select v-model:value="recordForm.plan_id" :options="planOpts" clearable placeholder="可选，关联后自动更新计划" /></n-form-item>
        <n-form-item label="设备"><n-select v-model:value="recordForm.equipment_id" :options="eqOpts" placeholder="选择设备" /></n-form-item>
        <n-form-item label="操作人"><n-input v-model:value="recordForm.operator_name" placeholder="操作人姓名" /></n-form-item>
        <n-form-item label="保养内容"><n-input v-model:value="recordForm.content" type="textarea" :rows="3" placeholder="描述本次保养工作内容" /></n-form-item>
        <n-form-item label="耗时"><n-input-number v-model:value="recordForm.duration_min" :min="0" style="width:100%"><template #suffix>分钟</template></n-input-number></n-form-item>
        <n-button type="primary" block @click="saveRecord">保存</n-button>
      </n-form>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch, h, computed } from 'vue'
import { NTag, NButton } from 'naive-ui'
import http from '../../api/http'
import { formatTime } from '../../utils/status'

const activeTab = ref('equipment')
const loading = ref(false)
const dictMap = ref<Record<string, any[]>>({})
const allEquipment = ref<any[]>([])

const triggerOpts = [
  { label: '定时', value: 'interval' },
  { label: '计数', value: 'counter' },
]
const triggerLabel = (t: string) => triggerOpts.find(o => o.value === t)?.label || t

const typeOpts = computed(() => (dictMap.value['device_type'] || []).map((d: any) => ({ label: d.name, value: d.code })))
const dictLabel = (category: string, code: string) => (dictMap.value[category] || []).find((d: any) => d.code === code)?.name || code || '-'
const eqOpts = computed(() => allEquipment.value.map(e => ({ label: `${e.code} - ${e.name}`, value: e.id })))
const eqName = (id: number) => { const e = allEquipment.value.find(x => x.id === id); return e ? `${e.code} - ${e.name}` : String(id) }

// --- Equipment ---
const equipment = ref<any[]>([])
const eqType = ref<string | null>(null)

const eqColumns = [
  { title: '编号', key: 'code', width: 110 },
  { title: '名称', key: 'name', width: 120 },
  { title: '类型', key: 'type', width: 90, render: (r: any) => dictLabel('device_type', r.type) },
  { title: '区域', key: 'station_type', width: 70, render: (r: any) => dictLabel('station_type', r.station_type) },
  { title: '产线', key: 'line_code', width: 60 },
  { title: '位置', key: 'location', width: 100 },
  { title: '状态', key: 'status', width: 70, render: (r: any) => h(NTag, { type: r.status === 'running' || r.status === 'in_use' ? 'success' : r.status === 'fault' ? 'error' : 'default', size: 'small' }, () => dictLabel('device_status', r.status)) },
  { title: '累计周期', key: 'total_cycles', width: 80 },
]

async function fetchEquipment() {
  loading.value = true
  const params = new URLSearchParams()
  if (eqType.value) params.set('type', eqType.value)
  const res: any = await http.get(`/equipment?${params}`)
  equipment.value = res.data || []
  loading.value = false
}

// --- Plans ---
const plans = ref<any[]>([])
const overdue = ref<any[]>([])
const upcoming = ref<any[]>([])
const showPlanModal = ref(false)
const planForm = ref({ id: 0, equipment_id: 0 as number, name: '', trigger_type: 'interval', interval_hours: 720, counter_threshold: 10000, warn_before_hours: 48 })

const planColumns = [
  { title: '设备', key: 'equipment_id', width: 160, render: (r: any) => eqName(r.equipment_id) },
  { title: '计划名称', key: 'name', width: 140 },
  { title: '触发方式', key: 'trigger_type', width: 70, render: (r: any) => triggerLabel(r.trigger_type) },
  { title: '间隔/阈值', key: '_interval', width: 90, render: (r: any) => r.trigger_type === 'interval' ? `${r.interval_hours}h` : r.counter_threshold ? `${r.counter_threshold}次` : '-' },
  { title: '上次完成', key: 'last_done_at', width: 140, render: (r: any) => formatTime(r.last_done_at) },
  { title: '下次到期', key: 'next_due_at', width: 140, render: (r: any) => {
    const text = formatTime(r.next_due_at)
    if (!r.next_due_at) return text
    const isOverdue = new Date(r.next_due_at) < new Date()
    return h(NTag, { type: isOverdue ? 'error' : 'default', size: 'small' }, () => text)
  } },
  { title: '操作', key: 'a', width: 80, render: (r: any) => h(NButton, { size: 'tiny', onClick: () => quickRecord(r) }, () => '执行') },
]

const planOpts = computed(() => plans.value.map(p => ({ label: `${eqName(p.equipment_id)} - ${p.name}`, value: p.id })))

function openPlanForm() {
  planForm.value = { id: 0, equipment_id: 0, name: '', trigger_type: 'interval', interval_hours: 720, counter_threshold: 10000, warn_before_hours: 48 }
  showPlanModal.value = true
}

async function fetchPlans() {
  const [plansRes, overdueRes, upcomingRes]: any[] = await Promise.all([
    http.get('/maintenance/plans'),
    http.get('/maintenance/overdue'),
    http.get('/maintenance/upcoming?hours=48'),
  ])
  plans.value = plansRes.data || []
  overdue.value = overdueRes.data || []
  upcoming.value = upcomingRes.data || []
}

async function savePlan() {
  await http.post('/maintenance/plans', planForm.value)
  showPlanModal.value = false
  fetchPlans()
}

// --- Records ---
const records = ref<any[]>([])
const recordTotal = ref(0)
const recordPage = ref(1)
const recordEqId = ref<number | null>(null)
const showRecordModal = ref(false)
const recordForm = ref({ plan_id: null as number | null, equipment_id: 0 as number, operator_id: 1, operator_name: '', content: '', duration_min: 0 })

const recordColumns = [
  { title: '设备', key: 'equipment_id', width: 160, render: (r: any) => eqName(r.equipment_id) },
  { title: '操作人', key: 'operator_id', width: 80 },
  { title: '保养内容', key: 'content' },
  { title: '耗时(分)', key: 'duration_min', width: 70 },
  { title: '完成时间', key: 'done_at', width: 140, render: (r: any) => formatTime(r.done_at) },
]

function openRecordForm() {
  recordForm.value = { plan_id: null, equipment_id: 0, operator_id: 1, operator_name: '', content: '', duration_min: 0 }
  showRecordModal.value = true
}

function quickRecord(plan: any) {
  recordForm.value = { plan_id: plan.id, equipment_id: plan.equipment_id, operator_id: 1, operator_name: '', content: plan.name, duration_min: 0 }
  showRecordModal.value = true
}

async function fetchRecords() {
  const params = new URLSearchParams({ page: String(recordPage.value), page_size: '20' })
  if (recordEqId.value) params.set('equipment_id', String(recordEqId.value))
  const res: any = await http.get(`/maintenance/records?${params}`)
  records.value = res.data?.items || []
  recordTotal.value = res.data?.total || 0
}

async function saveRecord() {
  await http.post('/maintenance/records', recordForm.value)
  showRecordModal.value = false
  fetchRecords()
  if (activeTab.value === 'plans' || recordForm.value.plan_id) fetchPlans()
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

async function loadAllEquipment() {
  const res: any = await http.get('/equipment')
  allEquipment.value = res.data || []
}

watch(eqType, fetchEquipment)
watch(activeTab, (tab) => {
  if (tab === 'plans') fetchPlans()
  if (tab === 'records') fetchRecords()
})
onMounted(async () => {
  await loadDict()
  await loadAllEquipment()
  fetchEquipment()
})
</script>
