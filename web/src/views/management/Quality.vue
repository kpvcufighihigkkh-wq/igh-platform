<template>
  <div>
    <n-card>
      <n-tabs type="line" v-model:value="activeTab">
        <n-tab-pane name="bobbins" tab="丝饼追溯">
          <n-space style="margin-bottom: 12px">
            <n-select v-model:value="bobbinLotId" :options="lotOptions" style="width: 200px" clearable placeholder="按批次" />
            <n-select v-model:value="bobbinStatus" :options="bobbinStatusOpts" style="width: 120px" clearable placeholder="状态" />
          </n-space>
          <n-data-table :columns="bobbinColumns" :data="bobbins" :bordered="false" :loading="loading" />
          <n-pagination v-model:page="bobbinPage" :page-count="Math.ceil(bobbinTotal / 20)" style="margin-top: 12px; justify-content: flex-end" @update:page="fetchBobbins" />
        </n-tab-pane>

        <n-tab-pane name="grades" tab="等级配置">
          <n-data-table :columns="gradeColumns" :data="grades" :bordered="false" />
        </n-tab-pane>

        <n-tab-pane name="defects" tab="缺陷定义">
          <n-data-table :columns="defectColumns" :data="defects" :bordered="false" />
        </n-tab-pane>

        <n-tab-pane name="rules" tab="称重规则">
          <n-space justify="space-between" style="margin-bottom: 12px">
            <n-select v-model:value="ruleLotId" :options="lotOptions" style="width: 200px" clearable placeholder="按批次" />
            <n-button type="primary" @click="showRuleModal = true">新增规则</n-button>
          </n-space>
          <n-data-table :columns="ruleColumns" :data="rules" :bordered="false" />
        </n-tab-pane>
      </n-tabs>
    </n-card>

    <n-modal v-model:show="showRuleModal" preset="card" title="新增称重规则" style="width: 420px">
      <n-form :model="ruleForm" label-placement="left" label-width="80">
        <n-form-item label="最小重量"><n-input-number v-model:value="ruleForm.min_weight" :precision="3" style="width: 100%" /></n-form-item>
        <n-form-item label="最大重量"><n-input-number v-model:value="ruleForm.max_weight" :precision="3" style="width: 100%" /></n-form-item>
        <n-form-item label="等级"><n-input v-model:value="ruleForm.grade_code" placeholder="如 AA" /></n-form-item>
        <n-button type="primary" block @click="createRule">创建</n-button>
      </n-form>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch, h, computed } from 'vue'
import { NTag } from 'naive-ui'
import http from '../../api/http'
import { tStatus } from '../../utils/status'

const activeTab = ref('bobbins')
const loading = ref(false)

// --- Lots for filter ---
const lots = ref<any[]>([])
const lotOptions = computed(() => lots.value.map((l: any) => ({ label: l.code, value: l.id })))
async function fetchLots() {
  const res: any = await http.get('/lots?page=1&page_size=100')
  lots.value = res.data?.items || []
}

// --- Bobbins ---
const bobbins = ref<any[]>([])
const bobbinTotal = ref(0)
const bobbinPage = ref(1)
const bobbinLotId = ref<number | null>(null)
const bobbinStatus = ref<string | null>(null)
const bobbinStatusOpts = [
  { label: '在制', value: 'in_process' },
  { label: '完成', value: 'completed' },
]

const bobbinColumns = [
  { title: 'ID', key: 'id', width: 60 },
  { title: '批次', key: 'lot_id', width: 60 },
  { title: '落纱ID', key: 'doffing_id', width: 80 },
  { title: '位置', key: 'current_position', width: 80 },
  { title: '分拣等级', key: 'sorting_grade', width: 80, render: (r: any) => r.sorting_grade ? h(NTag, { size: 'small', type: 'info' }, () => r.sorting_grade) : '-' },
  { title: '重量', key: 'weight', width: 80, render: (r: any) => r.weight ? `${r.weight}g` : '-' },
  { title: '最终等级', key: 'final_grade', width: 80, render: (r: any) => r.final_grade ? h(NTag, { size: 'small', type: gradeType(r.final_grade) }, () => r.final_grade) : '-' },
  { title: '状态', key: 'status', width: 80, render: (r: any) => tStatus(r.status) },
]

function gradeType(g: string) {
  if (g === 'AAA' || g === 'AA') return 'success'
  if (g === 'A1' || g === 'A') return 'warning'
  return 'default'
}

async function fetchBobbins() {
  loading.value = true
  const params = new URLSearchParams({ page: String(bobbinPage.value), page_size: '20' })
  if (bobbinLotId.value) params.set('lot_id', String(bobbinLotId.value))
  if (bobbinStatus.value) params.set('status', bobbinStatus.value)
  const res: any = await http.get(`/bobbins?${params}`)
  bobbins.value = res.data?.items || []
  bobbinTotal.value = res.data?.total || 0
  loading.value = false
}

// --- Grades ---
const grades = ref<any[]>([])
const gradeColumns = [
  { title: 'ID', key: 'id', width: 60 },
  { title: '类别', key: 'category', width: 100 },
  { title: '代码', key: 'code', width: 80 },
  { title: '名称', key: 'name' },
  { title: '排序', key: 'sort_order', width: 60 },
]
async function fetchGrades() {
  const res: any = await http.get('/grades')
  grades.value = res.data || []
}

// --- Defects ---
const defects = ref<any[]>([])
const defectColumns = [
  { title: 'ID', key: 'id', width: 60 },
  { title: '代码', key: 'code' },
  { title: '名称', key: 'name' },
  { title: '类别', key: 'category' },
  { title: '自动等级', key: 'auto_grade', width: 80 },
]
async function fetchDefects() {
  const res: any = await http.get('/defects')
  defects.value = res.data || []
}

// --- Weighing Rules ---
const rules = ref<any[]>([])
const ruleLotId = ref<number | null>(null)
const showRuleModal = ref(false)
const ruleForm = ref({ min_weight: 3.0, max_weight: 4.0, grade_code: 'AA' })

const ruleColumns = [
  { title: 'ID', key: 'id', width: 60 },
  { title: '批次ID', key: 'lot_id', width: 80 },
  { title: '最小重量', key: 'min_weight' },
  { title: '最大重量', key: 'max_weight' },
  { title: '等级', key: 'grade_code', width: 80 },
]

async function fetchRules() {
  const params = new URLSearchParams()
  if (ruleLotId.value) params.set('lot_id', String(ruleLotId.value))
  const res: any = await http.get(`/weighing-rules?${params}`)
  rules.value = res.data || []
}

async function createRule() {
  await http.post('/weighing-rules', ruleForm.value)
  showRuleModal.value = false
  ruleForm.value = { min_weight: 3.0, max_weight: 4.0, grade_code: 'AA' }
  fetchRules()
}

watch(bobbinLotId, () => { bobbinPage.value = 1; fetchBobbins() })
watch(bobbinStatus, () => { bobbinPage.value = 1; fetchBobbins() })
watch(ruleLotId, fetchRules)
watch(activeTab, (tab) => {
  if (tab === 'grades') fetchGrades()
  if (tab === 'defects') fetchDefects()
  if (tab === 'rules') fetchRules()
})
onMounted(() => { fetchLots(); fetchBobbins() })
</script>
