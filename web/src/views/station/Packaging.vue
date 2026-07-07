<template>
  <div class="station">
    <header class="station-header">
      <h1>包装工位</h1>
      <div class="header-info">
        <span class="user" v-if="isLoggedIn()">{{ userName }} <n-button text size="tiny" @click="logout">退出</n-button></span>
        <n-tag type="success" size="small">在线</n-tag>
        <span class="clock">{{ clock }}</span>
      </div>
    </header>
    <ModuleAlert />
    <n-tabs type="line" v-model:value="tab">
      <n-tab-pane name="orders" tab="订单队列">
        <div class="order-active" v-if="activeOrder">
          <h3>当前订单</h3>
          <n-descriptions :column="3" size="small" bordered>
            <n-descriptions-item label="批次">{{ activeOrder.lot_id }}</n-descriptions-item>
            <n-descriptions-item label="等级">{{ activeOrder.grade_code }}</n-descriptions-item>
            <n-descriptions-item label="进度">{{ activeOrder.packed_quantity }} / {{ activeOrder.target_quantity }}</n-descriptions-item>
          </n-descriptions>
          <n-progress :percentage="Math.round((activeOrder.packed_quantity/activeOrder.target_quantity)*100)" type="line" status="success" style="margin-top:8px" />
        </div>
        <n-space style="margin:12px 0">
          <n-button size="small" type="primary" @click="requireAuth(() => showCreateOrder=true)">新建订单</n-button>
        </n-space>
        <n-data-table :columns="orderCols" :data="orders" :bordered="false" size="small" />
      </n-tab-pane>
      <n-tab-pane name="pallets" tab="托盘记录">
        <n-data-table :columns="palletCols" :data="pallets" :bordered="false" size="small" />
      </n-tab-pane>
      <n-tab-pane name="print" tab="打印管理">
        <n-button size="small" @click="requireAuth(() => {})">补打标签</n-button>
        <n-empty description="打印记录将在标签打印后显示" style="margin-top:24px" />
      </n-tab-pane>
    </n-tabs>
    <LoginModal />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue'
import http from '../../api/http'
import { tStatus, formatTime } from '../../utils/status'
import { useAuth } from '../../composables/useAuth'
import LoginModal from '../../components/LoginModal.vue'
import ModuleAlert from '../../components/ModuleAlert.vue'

const { isLoggedIn, userName, requireAuth, logout } = useAuth()
const tab = ref('orders')
const clock = ref('')
const orders = ref<any[]>([])
const pallets = ref<any[]>([])
const showCreateOrder = ref(false)
let timer: any

const activeOrder = computed(() => orders.value.find(o => o.status === 'in_progress'))

const orderCols = [
  { title: '批次', key: 'lot_id', width: 60 },
  { title: '等级', key: 'grade_code', width: 60 },
  { title: '目标', key: 'target_quantity', width: 60 },
  { title: '已完成', key: 'packed_quantity', width: 70 },
  { title: '状态', key: 'status', width: 80, render: (r: any) => tStatus(r.status) },
  { title: '创建时间', key: 'created_at', width: 150, render: (r: any) => formatTime(r.created_at) },
]
const palletCols = [
  { title: '托盘号', key: 'code' },
  { title: '等级', key: 'grade_code', width: 60 },
  { title: '丝饼数', key: 'bobbin_count', width: 70 },
  { title: '层数', key: 'layers', width: 50 },
  { title: '重量(kg)', key: 'total_weight', width: 80 },
  { title: '时间', key: 'created_at', width: 150, render: (r: any) => formatTime(r.created_at) },
]

async function load() {
  const [oRes, pRes]: any[] = await Promise.all([
    http.get('/packing-orders?page=1&page_size=20'),
    http.get('/pallets?page=1&page_size=20'),
  ])
  orders.value = oRes.data?.items || []
  pallets.value = pRes.data?.items || []
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
.order-active { background:#161b22; border:1px solid #30363d; border-radius:8px; padding:12px; }
.order-active h3 { margin:0 0 8px; font-size:14px; color:#63e2b7; }
</style>
