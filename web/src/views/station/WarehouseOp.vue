<template>
  <div class="station">
    <header class="station-header">
      <h1>仓库工位</h1>
      <div class="header-info">
        <span class="user" v-if="isLoggedIn()">{{ userName }} <n-button text size="tiny" @click="logout">退出</n-button></span>
        <span class="clock">{{ clock }}</span>
      </div>
    </header>
    <ModuleAlert />
    <n-grid :cols="3" :x-gap="12">
      <n-gi :span="2">
        <n-card title="库位状态" size="small">
          <template #header-extra>
            <n-select v-model:value="selectedId" :options="whOpts" size="small" style="width:160px" @update:value="onWhChange" />
          </template>
          <WarehouseGrid
            v-if="warehouse"
            :rows="warehouse.rows"
            :cols="warehouse.cols"
            :layers="warehouse.layers"
            :slots="slots"
            @slot-click="onSlotClick"
          />
        </n-card>
      </n-gi>
      <n-gi>
        <n-card title="统计" size="small">
          <n-descriptions :column="1" size="small">
            <n-descriptions-item label="空闲">{{ wStats.empty || 0 }}</n-descriptions-item>
            <n-descriptions-item label="占用">{{ wStats.occupied || 0 }}</n-descriptions-item>
            <n-descriptions-item label="预留">{{ wStats.reserved || 0 }}</n-descriptions-item>
            <n-descriptions-item label="占用率">{{ usage }}%</n-descriptions-item>
          </n-descriptions>
        </n-card>
        <n-card title="操作" size="small" style="margin-top:12px">
          <n-space vertical>
            <n-text depth="3" style="font-size:12px">
              {{ selected ? `已选：行${selected.row_no} 列${selected.col_no} 层${selected.layer_no}（${selected.status}）` : '先在库位图上点选一个库位' }}
            </n-text>
            <n-button size="small" type="primary" block :disabled="!canInbound"
              @click="requireAuth(() => showInbound = true)">手动入库</n-button>
            <n-button size="small" block :disabled="!canOutbound"
              @click="requireAuth(doOutbound)">手动出库</n-button>
          </n-space>
        </n-card>
        <n-card title="待入库料车" size="small" style="margin-top:12px">
          <n-data-table :columns="trolleyCols" :data="trolleys" :bordered="false" size="small" :max-height="200" />
        </n-card>
      </n-gi>
    </n-grid>

    <!-- 手动入库：先在库位图上点选目标库位，再填占用类型与对象 -->
    <n-modal v-model:show="showInbound" preset="card" title="手动入库" style="width:420px">
      <n-form label-placement="left" label-width="80">
        <n-form-item label="库位">
          <n-text>{{ slotLabel }}</n-text>
        </n-form-item>
        <n-form-item label="类型">
          <n-select v-model:value="inbound.occupant_type" :options="occupantOpts" />
        </n-form-item>
        <n-form-item label="对象ID">
          <n-input-number v-model:value="inbound.occupant_id" :min="1" placeholder="栈板/料车 ID" style="width:100%" />
        </n-form-item>
        <n-form-item label="批次ID">
          <n-input-number v-model:value="inbound.lot_id" :min="1" placeholder="选填" style="width:100%" />
        </n-form-item>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showInbound = false">取消</n-button>
          <n-button type="primary" :loading="busy" @click="doInbound">确认入库</n-button>
        </n-space>
      </template>
    </n-modal>

    <LoginModal />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { useMessage } from 'naive-ui'
import http from '../../api/http'
import { tStatus } from '../../utils/status'
import { useAuth } from '../../composables/useAuth'
import LoginModal from '../../components/LoginModal.vue'
import ModuleAlert from '../../components/ModuleAlert.vue'
import WarehouseGrid from '../../components/warehouse/WarehouseGrid.vue'

const message = useMessage()
const { isLoggedIn, userName, requireAuth, logout } = useAuth()
const clock = ref('')
const warehouses = ref<any[]>([])
const warehouse = ref<any>(null)
const selectedId = ref<number | null>(null)
const slots = ref<any[]>([])
const wStats = ref<Record<string, number>>({ empty: 0, occupied: 0 })
const trolleys = ref<any[]>([])
let timer: any

// --- 手动入库/出库 ---
//
// 这两个按钮原来绑的是 `requireAuth(() => {})` —— 点了什么都不发生。
// 现场会以为"已经手动入库了"，实际库里没变，账面与实物直接错位。
// 现在接真实接口，并且：
//   入库只在空位/预留位可用，出库只在占用位可用（与后端校验一致）
//   未选中库位时按钮置灰，而不是点了没反应
const selected = ref<any>(null)
const showInbound = ref(false)
const busy = ref(false)
const inbound = ref<{ occupant_type: string; occupant_id: number | null; lot_id: number | null }>({
  occupant_type: 'pallet', occupant_id: null, lot_id: null,
})
const occupantOpts = [
  { label: '栈板', value: 'pallet' },
  { label: '料车', value: 'trolley' },
]
const slotLabel = computed(() =>
  selected.value ? `行${selected.value.row_no} 列${selected.value.col_no} 层${selected.value.layer_no}` : '')
const canInbound = computed(() =>
  !!selected.value && ['empty', 'reserved'].includes(selected.value.status))
const canOutbound = computed(() => !!selected.value && selected.value.status === 'occupied')

function onSlotClick(slot: any) {
  selected.value = slot
}

async function doInbound() {
  if (!selected.value || !warehouse.value) return
  if (!inbound.value.occupant_id) {
    message.warning('请填写对象 ID')
    return
  }
  busy.value = true
  try {
    await http.post(`/warehouses/${warehouse.value.id}/inbound`, {
      row: selected.value.row_no,
      col: selected.value.col_no,
      layer: selected.value.layer_no,
      occupant_type: inbound.value.occupant_type,
      occupant_id: inbound.value.occupant_id,
      lot_id: inbound.value.lot_id || undefined,
    })
    message.success('已入库')
    showInbound.value = false
    inbound.value.occupant_id = null
    inbound.value.lot_id = null
    selected.value = null
    await loadSlots(warehouse.value.id)
  } catch (e: any) {
    // 后端会区分 404 库位不存在 / 409 库位不可用，原样展示：
    // 笼统的"操作失败"会让现场反复重试同一个错误。
    message.error(e?.response?.data?.message || '入库失败')
  } finally {
    busy.value = false
  }
}

async function doOutbound() {
  if (!selected.value || !warehouse.value) return
  busy.value = true
  try {
    await http.post(`/warehouses/${warehouse.value.id}/outbound`, {
      row: selected.value.row_no,
      col: selected.value.col_no,
      layer: selected.value.layer_no,
    })
    message.success('已出库')
    selected.value = null
    await loadSlots(warehouse.value.id)
  } catch (e: any) {
    message.error(e?.response?.data?.message || '出库失败')
  } finally {
    busy.value = false
  }
}

const whOpts = computed(() => warehouses.value.map(w => ({ label: w.name, value: w.id })))
const usage = computed(() => { const t = (wStats.value.empty || 0) + (wStats.value.occupied || 0) + (wStats.value.reserved || 0); return t > 0 ? Math.round(((wStats.value.occupied || 0) / t) * 100) : 0 })

const trolleyCols = [
  { title: '编号', key: 'number', width: 60 },
  { title: '状态', key: 'status', width: 80, render: (r: any) => tStatus(r.status) },
]

async function onWhChange(id: number) {
  warehouse.value = warehouses.value.find(w => w.id === id)
  await loadSlots(id)
}

async function loadSlots(id: number) {
  const [sRes, stRes]: any[] = await Promise.all([
    http.get(`/warehouses/${id}/slots`),
    http.get(`/warehouses/${id}/stats`),
  ])
  slots.value = sRes.data?.items || []
  wStats.value = stRes.data || { empty: 0, occupied: 0 }
}

async function load() {
  const [whRes, tRes]: any[] = await Promise.all([http.get('/warehouses'), http.get('/trolleys?page=1&page_size=10')])
  trolleys.value = tRes.data?.items || []
  warehouses.value = whRes.data?.items || []
  if (warehouses.value.length && !selectedId.value) {
    selectedId.value = warehouses.value[0].id
    warehouse.value = warehouses.value[0]
    await loadSlots(warehouses.value[0].id)
  } else if (selectedId.value) {
    await loadSlots(selectedId.value)
  }
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
