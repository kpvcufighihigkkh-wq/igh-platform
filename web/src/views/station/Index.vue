<template>
  <div class="station-page">
    <n-card :title="'工位: ' + stationCode">
      <template #header-extra>
        <n-tag :type="connected ? 'success' : 'error'" size="small">
          {{ connected ? '在线' : '离线' }}
        </n-tag>
      </template>

      <!-- Winding station -->
      <div v-if="stationType === 'winding'" class="station-content">
        <n-grid :cols="2" :x-gap="16" :y-gap="16">
          <n-gi>
            <n-card title="当前批号" size="small">
              <div class="big-text">{{ currentLot || '--' }}</div>
            </n-card>
          </n-gi>
          <n-gi>
            <n-card title="今日落纱" size="small">
              <div class="big-text">{{ todayDoffings }}</div>
            </n-card>
          </n-gi>
        </n-grid>
        <n-space style="margin-top: 16px">
          <n-button type="primary" size="large" @click="reprintLabel">标签补打</n-button>
          <n-button type="warning" size="large" @click="showOverride = true">手动干预</n-button>
        </n-space>
      </div>

      <!-- Sorting station -->
      <div v-else-if="stationType === 'sorting'" class="station-content">
        <n-space vertical size="large">
          <n-button-group>
            <n-button v-for="g in grades" :key="g" size="large" @click="submitGrade(g)">{{ g }}</n-button>
          </n-button-group>
          <n-button type="primary" size="large" @click="reprintLabel">标签补打</n-button>
        </n-space>
      </div>

      <!-- Packaging station -->
      <div v-else-if="stationType === 'packaging'" class="station-content">
        <n-grid :cols="2" :x-gap="16" :y-gap="16">
          <n-gi>
            <n-card title="当前订单" size="small">
              <div class="big-text">{{ currentOrder || '无' }}</div>
            </n-card>
          </n-gi>
          <n-gi>
            <n-card title="已打包" size="small">
              <div class="big-text">{{ packedCount }}/{{ targetCount }}</div>
            </n-card>
          </n-gi>
        </n-grid>
        <n-space style="margin-top: 16px">
          <n-button type="primary" size="large" @click="showCreateOrder = true">创建订单</n-button>
          <n-button type="info" size="large" @click="reprintLabel">标签补打</n-button>
        </n-space>
      </div>

      <!-- Warehouse station -->
      <div v-else-if="stationType === 'warehouse'" class="station-content">
        <n-space vertical size="large">
          <n-button type="primary" size="large">入库确认</n-button>
          <n-button type="info" size="large">手动出库</n-button>
          <n-button size="large" @click="reprintLabel">标签补打</n-button>
        </n-space>
      </div>

      <!-- Default -->
      <div v-else class="station-content">
        <n-alert type="info">未知工位类型: {{ stationType }}</n-alert>
      </div>
    </n-card>

    <!-- 待应用现场配置：中心下发后必须人工确认才生效 -->
    <n-card v-if="pendingConfig" title="现场配置待应用" style="margin-top: 16px">
      <template #header-extra>
        <n-tag type="warning" size="small">版本 {{ pendingConfig.version }}</n-tag>
      </template>

      <n-alert type="warning" :bordered="false" style="margin-bottom: 12px">
        中心下发了新的现场配置（{{ pendingSummary }}）。
        应用后边端会按新点表重新连接 PLC，产线运行中请谨慎操作。
      </n-alert>

      <n-space>
        <n-button type="primary" :loading="applyingConfig" @click="applyConfig">应用配置</n-button>
        <n-button @click="discardConfig">放弃</n-button>
      </n-space>
    </n-card>

    <!-- 配置校验问题 -->
    <n-card v-if="configProblems.length" title="配置校验未通过" style="margin-top: 16px">
      <n-alert type="error" :bordered="false">
        <ul class="config-problems">
          <li v-for="(p, i) in configProblems" :key="i">{{ p }}</li>
        </ul>
      </n-alert>
    </n-card>

    <!-- Active alarms -->
    <n-card title="当前异常" style="margin-top: 16px" v-if="activeAlarms.length > 0">
      <div v-for="a in activeAlarms" :key="a.id" class="alarm-item">
        <n-tag :type="a.level === 'urgent' ? 'error' : 'warning'" size="small">{{ a.level }}</n-tag>
        <span class="alarm-msg">{{ a.message }}</span>
        <n-button size="tiny" @click="acknowledgeAlarm(a.id)">确认</n-button>
      </div>
    </n-card>

    <!-- Reprint modal -->
    <n-modal v-model:show="showReprint" preset="card" title="标签补打" style="width: 400px">
      <n-form>
        <n-form-item label="目标ID">
          <n-input v-model:value="reprintTarget" placeholder="丝饼ID或托盘编码" />
        </n-form-item>
        <n-button type="primary" block @click="doReprint">打印</n-button>
      </n-form>
    </n-modal>

    <!-- Override modal -->
    <n-modal v-model:show="showOverride" preset="card" title="手动干预" style="width: 400px">
      <n-form>
        <n-form-item label="干预原因">
          <n-input v-model:value="overrideReason" type="textarea" placeholder="请输入原因" />
        </n-form-item>
        <n-button type="warning" block @click="doOverride">确认干预</n-button>
      </n-form>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useWebSocket } from '../../composables/useWebSocket'
import http from '../../api/http'

const route = useRoute()
const stationCode = computed(() => route.params.stationCode as string)
const stationType = computed(() => {
  const code = stationCode.value
  if (code.startsWith('winding')) return 'winding'
  if (code.startsWith('sorting')) return 'sorting'
  if (code.startsWith('packaging')) return 'packaging'
  if (code.startsWith('warehouse')) return 'warehouse'
  return 'unknown'
})

const { connected } = useWebSocket(['production', 'alarm'])

const grades = ['AAA', 'AA', 'AA1', 'AA2', 'A1', 'A']
const currentLot = ref('')
const todayDoffings = ref(0)
const currentOrder = ref('')
const packedCount = ref(0)
const targetCount = ref(0)
const activeAlarms = ref<any[]>([])

const showReprint = ref(false)
const showOverride = ref(false)
const showCreateOrder = ref(false)
const reprintTarget = ref('')
const overrideReason = ref('')

// 现场配置：中心下发后只暂存，必须人工确认才生效
const pendingConfig = ref<any>(null)
const configProblems = ref<string[]>([])
const applyingConfig = ref(false)

const pendingSummary = computed(() => {
  const p = pendingConfig.value
  if (!p) return ''
  const parts: string[] = []
  if (p.plc?.length) parts.push(`${p.plc.length} 个 PLC 点表`)
  if (p.loading?.patterns?.length) parts.push(`${p.loading.patterns.length} 套装载查表`)
  if (p.heads?.head_maps?.length) parts.push(`${p.heads.head_maps.length} 套锭位映射`)
  return parts.join('、') || '无实际内容'
})

async function loadConfig() {
  try {
    const res: any = await http.get('/edge/config')
    pendingConfig.value = res.data?.pending || null
    configProblems.value = res.data?.problems || []
  } catch {
    // 旧版本边端没有该接口：静默忽略，不影响工位主流程
  }
}

async function applyConfig() {
  applyingConfig.value = true
  try {
    await http.post('/edge/config/apply', {})
    pendingConfig.value = null
    configProblems.value = []
  } catch (e: any) {
    configProblems.value = [e?.response?.data?.message || e?.message || String(e)]
  } finally {
    applyingConfig.value = false
  }
}

async function discardConfig() {
  try {
    await http.post('/edge/config/discard', {})
  } finally {
    pendingConfig.value = null
    configProblems.value = []
  }
}

onMounted(async () => {
  try {
    const res: any = await http.get('/alarms?status=active&page_size=5')
    activeAlarms.value = res.data?.items || []
  } catch {}
  loadConfig()
})

function reprintLabel() { showReprint.value = true }

async function doReprint() {
  await http.post('/labels/print', {
    template: 'bobbin.zpl',
    variables: { id: reprintTarget.value },
    printer_host: '192.168.30.100',
  })
  showReprint.value = false
  reprintTarget.value = ''
}

async function doOverride() {
  showOverride.value = false
  overrideReason.value = ''
}

async function submitGrade(grade: string) {
  // In real usage, bobbin_id comes from scanner/PLC
}

async function acknowledgeAlarm(id: number) {
  await http.put(`/alarms/${id}/acknowledge`)
  activeAlarms.value = activeAlarms.value.filter(a => a.id !== id)
}
</script>

<style scoped>
.station-page { height: 100%; }
.station-content { padding: 8px 0; }
.big-text { font-size: 28px; font-weight: bold; color: var(--color-primary); }
.alarm-item { display: flex; align-items: center; gap: 8px; padding: 8px 0; border-bottom: 1px solid rgba(255,255,255,0.05); }
.alarm-msg { flex: 1; font-size: 14px; }
.config-problems { margin: 4px 0 0 18px; padding: 0; }
.config-problems li { font-size: 13px; line-height: 1.7; }
</style>
