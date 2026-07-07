<template>
  <div class="station">
    <header class="station-header">
      <h1>卷绕工位</h1>
      <div class="header-info">
        <span class="user" v-if="isLoggedIn()">{{ userName }} <n-button text size="tiny" @click="logout">退出</n-button></span>
        <span class="clock">{{ clock }}</span>
      </div>
    </header>
    <ModuleAlert />
    <n-grid :cols="2" :x-gap="12" :y-gap="12">
      <n-gi>
        <n-card title="当前批次" size="small">
          <template #header-extra>
            <n-button size="tiny" type="primary" @click="requireAuth(() => showLotEdit=true)">维护批次</n-button>
          </template>
          <n-descriptions :column="1" size="small" v-if="lot">
            <n-descriptions-item label="批号">{{ lot.code }}</n-descriptions-item>
            <n-descriptions-item label="规格">{{ lot.specification }}</n-descriptions-item>
            <n-descriptions-item label="类型">{{ lot.product_type }}</n-descriptions-item>
          </n-descriptions>
          <n-empty v-else description="暂无活跃批次" />
        </n-card>
      </n-gi>
      <n-gi>
        <n-card title="今日统计" size="small">
          <n-statistic label="落纱次数" :value="todayDoff" />
          <n-statistic label="丝饼数量" :value="todayBobbin" style="margin-top:8px" />
        </n-card>
      </n-gi>
    </n-grid>
    <n-card title="最近落纱" size="small" style="margin-top:12px">
      <n-data-table :columns="doffCols" :data="doffings" :bordered="false" size="small" :max-height="200" />
    </n-card>
    <n-card title="料车状态" size="small" style="margin-top:12px">
      <template #header-extra>
        <n-button size="tiny" @click="requireAuth(() => {})">补打标签</n-button>
      </template>
      <n-data-table :columns="trolleyCols" :data="trolleys" :bordered="false" size="small" :max-height="150" />
    </n-card>
    <LoginModal />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import http from '../../api/http'
import { tStatus, formatTime } from '../../utils/status'
import { useAuth } from '../../composables/useAuth'
import LoginModal from '../../components/LoginModal.vue'
import ModuleAlert from '../../components/ModuleAlert.vue'

const { isLoggedIn, userName, requireAuth, logout } = useAuth()
const clock = ref('')
const lot = ref<any>(null)
const todayDoff = ref(0)
const todayBobbin = ref(0)
const doffings = ref<any[]>([])
const trolleys = ref<any[]>([])
const showLotEdit = ref(false)
let timer: any

const doffCols = [
  { title: '卷绕机', key: 'winder_name' },
  { title: '落纱号', key: 'doff_no', width: 60 },
  { title: '丝饼数', key: 'bobbin_count', width: 60 },
  { title: '时间', key: 'created_at', width: 150, render: (r: any) => formatTime(r.created_at) },
]
const trolleyCols = [
  { title: '编号', key: 'number', width: 60 },
  { title: '状态', key: 'status', width: 80, render: (r: any) => tStatus(r.status) },
  { title: '装载时间', key: 'loading_time', width: 150, render: (r: any) => formatTime(r.loading_time) },
]

async function load() {
  const [lotRes, doffRes, trolleyRes, statsRes]: any[] = await Promise.all([
    http.get('/lots?status=active&page=1&page_size=1'),
    http.get('/doffings?page=1&page_size=10'),
    http.get('/trolleys?page=1&page_size=10'),
    http.get('/stats/daily-output?days=1'),
  ])
  lot.value = (lotRes.data?.items || [])[0] || null
  doffings.value = doffRes.data?.items || []
  trolleys.value = trolleyRes.data?.items || []
  const today = (statsRes.data || [])[0]
  if (today) { todayDoff.value = today.doff_count; todayBobbin.value = today.bobbin_count }
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
</style>
