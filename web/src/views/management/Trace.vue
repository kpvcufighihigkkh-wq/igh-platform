<template>
  <n-card title="全链追溯">
    <template #header-extra>
      <n-space>
        <n-radio-group v-model:value="mode" size="small">
          <n-radio-button value="bobbin">按丝锭编码</n-radio-button>
          <n-radio-button value="pallet">按栈板编码</n-radio-button>
        </n-radio-group>
      </n-space>
    </template>

    <!-- 扫码枪直接扫标签即可输入，回车触发查询 -->
    <n-space style="margin-bottom:16px">
      <n-input
        v-model:value="code"
        :placeholder="mode === 'bobbin' ? '扫丝锭标签或输入编码' : '扫栈板标签或输入编码'"
        style="width:480px"
        clearable
        @keyup.enter="query"
      />
      <n-button type="primary" :loading="loading" @click="query">查询</n-button>
    </n-space>

    <n-alert v-if="notFound" type="warning" :show-icon="true" style="margin-bottom:16px">
      查无此编码。确认标签是否属于本厂，或该丝锭尚未入库。
    </n-alert>

    <!-- 链路缺环必须显式提示：静默省略会让人误以为这一环正常 -->
    <n-alert v-if="missing.length" type="error" title="追溯链不完整" style="margin-bottom:16px">
      <ul style="margin:0;padding-left:18px">
        <li v-for="m in missing" :key="m">{{ m }}</li>
      </ul>
    </n-alert>

    <template v-if="result">
      <n-descriptions bordered :column="2" size="small" title="丝锭">
        <n-descriptions-item label="编码">{{ result.bobbin.code || '—' }}</n-descriptions-item>
        <n-descriptions-item label="状态">{{ result.bobbin.status }}</n-descriptions-item>
        <n-descriptions-item label="锭位">{{ result.bobbin.place_in_winder }}</n-descriptions-item>
        <n-descriptions-item label="当前位置">{{ result.bobbin.current_position || '—' }}</n-descriptions-item>
        <n-descriptions-item label="等级">
          {{ result.bobbin.final_grade || result.bobbin.sorting_grade || '—' }}
        </n-descriptions-item>
        <n-descriptions-item label="缺陷">{{ result.bobbin.defect_code || '—' }}</n-descriptions-item>
      </n-descriptions>

      <n-descriptions bordered :column="2" size="small" title="落筒" style="margin-top:16px">
        <n-descriptions-item label="产线">
          {{ result.line?.code || '—' }} {{ result.line?.name || '' }}
        </n-descriptions-item>
        <n-descriptions-item label="机位">{{ result.doffing?.winder_name || '—' }}</n-descriptions-item>
        <n-descriptions-item label="落筒号">{{ result.doffing?.doff_no ?? '—' }}</n-descriptions-item>
        <n-descriptions-item label="落筒时间">{{ fmt(result.doffing?.end_time) }}</n-descriptions-item>
      </n-descriptions>

      <n-descriptions bordered :column="2" size="small" title="批次" style="margin-top:16px">
        <n-descriptions-item label="批号">{{ result.lot?.code || '—' }}</n-descriptions-item>
        <n-descriptions-item label="规格">{{ result.lot?.specification || '—' }}</n-descriptions-item>
        <n-descriptions-item label="工单">{{ result.lot?.order_code || '—' }}</n-descriptions-item>
        <n-descriptions-item label="产品类型">{{ result.lot?.product_type || '—' }}</n-descriptions-item>
      </n-descriptions>

      <n-descriptions v-if="result.pallet" bordered :column="2" size="small" title="栈板" style="margin-top:16px">
        <n-descriptions-item label="栈板编码">{{ result.pallet.pallet.code }}</n-descriptions-item>
        <n-descriptions-item label="等级">{{ result.pallet.pallet.grade_code || '—' }}</n-descriptions-item>
        <n-descriptions-item label="层/位">
          {{ result.pallet.layer || '—' }} / {{ result.pallet.position || '—' }}
        </n-descriptions-item>
        <n-descriptions-item label="整垛锭数">{{ result.pallet.pallet.bobbin_count }}</n-descriptions-item>
        <n-descriptions-item label="包装单">
          {{ result.pallet.packing_order ? '#' + result.pallet.packing_order.id : '—' }}
        </n-descriptions-item>
        <n-descriptions-item label="拣货单">
          {{ result.pallet.picking_order ? '#' + result.pallet.picking_order.id : '—' }}
        </n-descriptions-item>
      </n-descriptions>

      <!-- 栈板模式：列出这一垛上的全部丝锭 -->
      <template v-if="palletResult">
        <n-descriptions bordered :column="2" size="small" title="栈板" style="margin-top:16px">
          <n-descriptions-item label="栈板编码">{{ palletResult.pallet.code }}</n-descriptions-item>
          <n-descriptions-item label="批次">{{ palletResult.lot?.code || '—' }}</n-descriptions-item>
          <n-descriptions-item label="等级">{{ palletResult.pallet.grade_code || '—' }}</n-descriptions-item>
          <n-descriptions-item label="记录锭数">{{ palletResult.pallet.bobbin_count }}</n-descriptions-item>
        </n-descriptions>
        <n-data-table
          :columns="bobbinCols"
          :data="palletResult.bobbins"
          :bordered="false"
          size="small"
          style="margin-top:12px"
          :max-height="360"
        />
      </template>
    </template>

    <n-empty v-if="!result && !palletResult && !notFound" description="扫描或输入编码后查询" style="margin-top:48px" />
  </n-card>
</template>

<script setup lang="ts">
import { ref, computed, h } from 'vue'
import { NTag } from 'naive-ui'
import http from '../../api/http'

const mode = ref<'bobbin' | 'pallet'>('bobbin')
const code = ref('')
const loading = ref(false)
const notFound = ref(false)
const result = ref<any>(null)
const palletResult = ref<any>(null)

const missing = computed(() => {
  if (palletResult.value) return palletResult.value.missing || []
  if (result.value) return result.value.missing || []
  return []
})

function fmt(t?: string) {
  return t ? new Date(t).toLocaleString('zh-CN') : '—'
}

const bobbinCols = [
  { title: '编码', key: 'code', ellipsis: { tooltip: true } },
  { title: '锭位', key: 'place_in_winder', width: 70 },
  {
    title: '等级', key: 'final_grade', width: 80,
    render: (r: any) => h(NTag, { size: 'small' }, () => r.final_grade || r.sorting_grade || '—'),
  },
  { title: '状态', key: 'status', width: 110 },
]

async function query() {
  const c = code.value.trim()
  if (!c) return
  loading.value = true
  result.value = null
  palletResult.value = null
  notFound.value = false
  try {
    const res: any = await http.get(`/trace/${mode.value}/${encodeURIComponent(c)}`)
    if (mode.value === 'bobbin') result.value = res.data
    else palletResult.value = res.data
  } catch (e: any) {
    // 404 = 编码不存在（扫错/贴错），和系统故障区分开提示
    if (e?.response?.status === 404) notFound.value = true
    else notFound.value = false
  } finally {
    loading.value = false
  }
}
</script>
