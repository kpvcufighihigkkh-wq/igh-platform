<template>
  <div class="station">
    <header class="station-header">
      <h1>分拣工位</h1>
      <div class="header-info">
        <span class="user" v-if="isLoggedIn()">{{ userName }} <n-button text size="tiny" @click="logout">退出</n-button></span>
        <span class="clock">{{ clock }}</span>
      </div>
    </header>
    <ModuleAlert />
    <n-grid :cols="3" :x-gap="12">
      <n-gi :span="2">
        <n-card title="丝饼列表" size="small">
          <n-data-table :columns="bobbinCols" :data="bobbins" :bordered="false" size="small" :max-height="400" />
        </n-card>
      </n-gi>
      <n-gi>
        <n-card title="录入分拣结果" size="small">
          <n-form size="small">
            <n-form-item label="丝饼ID"><n-input-number v-model:value="sortForm.bobbin_id" style="width:100%" /></n-form-item>
            <n-form-item label="等级">
              <n-select v-model:value="sortForm.grade" :options="gradeOpts" />
            </n-form-item>
            <n-button type="primary" block size="small" @click="requireAuth(submitSorting)">提交</n-button>
          </n-form>
        </n-card>
        <n-card title="等级统计" size="small" style="margin-top:12px">
          <div v-for="d in dist" :key="d.grade" class="dist-item">
            <span>{{ d.grade }}</span><span>{{ d.count }}个</span>
          </div>
          <n-empty v-if="!dist.length" description="暂无数据" size="small" />
        </n-card>
      </n-gi>
    </n-grid>
    <LoginModal />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import http from '../../api/http'
import { tGrade, formatTime } from '../../utils/status'
import { useAuth } from '../../composables/useAuth'
import LoginModal from '../../components/LoginModal.vue'
import ModuleAlert from '../../components/ModuleAlert.vue'

const { isLoggedIn, userName, requireAuth, logout } = useAuth()
const clock = ref('')
const bobbins = ref<any[]>([])
const dist = ref<any[]>([])
const sortForm = ref({ bobbin_id: 1, grade: 'AA' })
let timer: any

const gradeOpts = [
  { label: 'AAA - 特等品', value: 'AAA' },
  { label: 'AA - 优等品', value: 'AA' },
  { label: 'AA1 - 一等品', value: 'AA1' },
  { label: 'AA2 - 二等品', value: 'AA2' },
  { label: 'A1 - 合格品', value: 'A1' },
  { label: 'A - 次品', value: 'A' },
]

const bobbinCols = [
  { title: 'ID', key: 'id', width: 50 },
  { title: '位置', key: 'place_in_winder', width: 40 },
  { title: '分拣等级', key: 'sorting_grade', width: 80, render: (r: any) => r.sorting_grade || '待分拣' },
  { title: '重量(g)', key: 'weight', width: 70, render: (r: any) => r.weight ? r.weight.toFixed(3) : '-' },
  { title: '最终等级', key: 'final_grade', width: 70, render: (r: any) => r.final_grade || '-' },
  { title: '时间', key: 'created_at', width: 140, render: (r: any) => formatTime(r.created_at) },
]

async function submitSorting() {
  await http.post('/quality/sorting', { bobbin_id: sortForm.value.bobbin_id, grade: sortForm.value.grade, defect_code: '' })
  load()
}

async function load() {
  const [bRes, dRes]: any[] = await Promise.all([
    http.get('/bobbins?page=1&page_size=30'),
    http.get('/stats/grade-distribution'),
  ])
  bobbins.value = bRes.data?.items || []
  dist.value = dRes.data || []
}

onMounted(() => {
  timer = setInterval(() => { clock.value = new Date().toLocaleTimeString('zh-CN', { hour12: false }); load() }, 20000)
  clock.value = new Date().toLocaleTimeString('zh-CN', { hour12: false })
  load()
})
onUnmounted(() => clearInterval(timer))
</script>

<style scoped>
.station { height:100vh; background:#0d1117; padding:16px; overflow:auto; }
.station-header { display:flex; justify-content:space-between; align-items:center; margin-bottom:12px; }
.station-header h1 { font-size:18px; color:#00d4ff; margin:0; }
.header-info { display:flex; align-items:center; gap:12px; }
.clock { font-family:monospace; color:#8b949e; }
.user { font-size:12px; color:#63e2b7; }
.dist-item { display:flex; justify-content:space-between; padding:3px 0; font-size:13px; color:#c9d1d9; }
</style>
