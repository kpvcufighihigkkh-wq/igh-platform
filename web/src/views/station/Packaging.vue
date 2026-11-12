<template>
  <div class="station">
    <header class="station-header">
      <h1>包装工位</h1>
      <div class="header-info">
        <span class="user" v-if="isLoggedIn()">{{ userName }} <n-button text size="tiny" @click="logout">退出</n-button></span>
        <n-tag :type="serverOnline ? 'success' : 'error'" size="small">
          {{ serverOnline ? '在线' : '离线' }}
        </n-tag>
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
        <n-button size="small" @click="requireAuth(() => { showPrint = true })">补打标签</n-button>
        <n-empty v-if="pallets.length === 0" description="暂无托盘，包装完成后可补打标签" style="margin-top:24px" />
        <n-data-table v-else :columns="printCols" :data="pallets" :bordered="false" size="small" style="margin-top:12px" />
      </n-tab-pane>
    </n-tabs>
    <n-modal v-model:show="showCreateOrder" preset="card" title="新建包装订单" style="width:420px">
      <n-form label-placement="left" label-width="80">
        <n-form-item label="批次">
          <n-select v-model:value="newOrder.lot_id" :options="lotOpts" filterable placeholder="选择批次" />
        </n-form-item>
        <n-form-item label="等级">
          <n-input v-model:value="newOrder.grade_code" placeholder="A / AA / B" />
        </n-form-item>
        <n-form-item label="目标数量">
          <n-input-number v-model:value="newOrder.target_quantity" :min="1" style="width:100%" />
        </n-form-item>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showCreateOrder = false">取消</n-button>
          <n-button type="primary" :loading="creating" @click="doCreateOrder">创建</n-button>
        </n-space>
      </template>
    </n-modal>

    <LoginModal />
    <PrintLabelModal v-model:show="showPrint" :preset="printPreset" />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed, h } from 'vue'
import { NButton } from 'naive-ui'
import http from '../../api/http'
import { tStatus, formatTime } from '../../utils/status'
import { useAuth } from '../../composables/useAuth'
import { useEdge } from '../../composables/useEdge'
import { useMessage } from 'naive-ui'
import LoginModal from '../../components/LoginModal.vue'
import ModuleAlert from '../../components/ModuleAlert.vue'
import PrintLabelModal from '../../components/PrintLabelModal.vue'

const { isLoggedIn, userName, requireAuth, logout } = useAuth()
const { serverOnline, checkServer } = useEdge()
const message = useMessage()
const tab = ref('orders')
const clock = ref('')
const orders = ref<any[]>([])
const pallets = ref<any[]>([])
const showCreateOrder = ref(false)
const showPrint = ref(false)
// 从托盘记录里点"补打"时带过去的数据
const printPreset = ref<any>({})

// --- 新建包装订单 ---
// 按钮原来只把 showCreateOrder 置 true，但没有对应弹窗，点了没反应。
const creating = ref(false)
const lots = ref<any[]>([])
const newOrder = ref<{ lot_id: number | null; grade_code: string; target_quantity: number }>({
  lot_id: null, grade_code: 'A', target_quantity: 48,
})
const lotOpts = computed(() => lots.value.map(l => ({ label: `${l.code} ${l.name || ''}`.trim(), value: l.id })))

async function doCreateOrder() {
  if (!newOrder.value.lot_id) { message.warning('请选择批次'); return }
  creating.value = true
  try {
    await http.post('/packing-orders', {
      lot_id: newOrder.value.lot_id,
      grade_code: newOrder.value.grade_code,
      target_quantity: newOrder.value.target_quantity,
    })
    message.success('订单已创建')
    showCreateOrder.value = false
    await load()
  } catch (e: any) {
    message.error(e?.response?.data?.message || '创建失败')
  } finally {
    creating.value = false
  }
}
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
const printCols = [
  { title: '托盘号', key: 'code' },
  { title: '等级', key: 'grade_code', width: 60 },
  { title: '丝饼数', key: 'bobbin_count', width: 70 },
  { title: '时间', key: 'created_at', width: 150, render: (r: any) => formatTime(r.created_at) },
  {
    title: '操作', key: 'a', width: 90,
    render: (r: any) => h(NButton, {
      size: 'tiny',
      onClick: () => requireAuth(() => {
        // 预填这一行的数据，避免人工重敲栈板号敲错
        printPreset.value = {
          palletCode: r.code,
          gradeCode: r.grade_code,
          bobbinCount: r.bobbin_count,
          layers: r.layers,
          totalWeight: r.total_weight || 0,
        }
        showPrint.value = true
      }),
    }, () => '补打'),
  },
]

async function load() {
  const [oRes, pRes, lRes]: any[] = await Promise.all([
    http.get('/packing-orders?page=1&page_size=20'),
    http.get('/pallets?page=1&page_size=20'),
    http.get('/lots?status=active&page=1&page_size=100'),
  ])
  orders.value = oRes.data?.items || []
  pallets.value = pRes.data?.items || []
  lots.value = lRes.data?.items || []
  checkServer()
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
