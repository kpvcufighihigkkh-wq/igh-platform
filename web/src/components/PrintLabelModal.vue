<template>
  <n-modal v-model:show="visible" preset="card" title="补打标签" style="width:420px">
    <n-spin :show="loading">
      <n-empty v-if="!loading && printers.length === 0" description="本工位未配置打印机">
        <template #extra>
          <n-text depth="3" style="font-size:12px">在边端 edge.yaml 的 printers 里添加后再试</n-text>
        </template>
      </n-empty>

      <n-form v-else label-placement="left" label-width="80">
        <n-form-item label="打印机">
          <n-select v-model:value="printerCode" :options="printerOpts" placeholder="选择打印机" />
        </n-form-item>
        <n-form-item label="标签模板">
          <n-select v-model:value="templateName" :options="templateOpts" placeholder="选择模板" />
        </n-form-item>
        <n-form-item label="栈板号">
          <n-input v-model:value="form.palletCode" placeholder="扫描或输入栈板编码" />
        </n-form-item>
        <n-form-item label="批号">
          <n-input v-model:value="form.lotCode" placeholder="选填" />
        </n-form-item>
        <n-form-item label="等级">
          <n-input v-model:value="form.gradeCode" placeholder="选填" />
        </n-form-item>
        <n-form-item label="锭数/层数">
          <n-space>
            <n-input-number v-model:value="form.bobbinCount" :min="0" style="width:100px" />
            <n-input-number v-model:value="form.layers" :min="0" style="width:100px" />
          </n-space>
        </n-form-item>
        <n-form-item label="重量(kg)">
          <n-input-number v-model:value="form.totalWeight" :min="0" :precision="3" style="width:100%" />
        </n-form-item>

        <n-alert v-if="error" type="error" style="margin-bottom:12px">{{ error }}</n-alert>

        <n-button type="primary" block :loading="loading" :disabled="!printerCode" @click="submit">
          打印
        </n-button>
      </n-form>
    </n-spin>
  </n-modal>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import http from '../api/http'

// 打印机与模板都从边端接口取，不在前端写死。
// 之前 station/Index.vue 里把 IP 硬编码成 192.168.30.100，
// 换个现场就得改代码重新打包。
const visible = defineModel<boolean>('show', { default: false })

const props = defineProps<{
  // 打开弹窗时预填的数据（从表格行点进来时用）
  preset?: { palletCode?: string; lotCode?: string; gradeCode?: string
    bobbinCount?: number; layers?: number; totalWeight?: number }
}>()

const loading = ref(false)
const error = ref('')
const printers = ref<any[]>([])
const templates = ref<string[]>([])
const printerCode = ref('')
const templateName = ref('pallet')

const form = ref({ palletCode: '', lotCode: '', gradeCode: '',
  bobbinCount: 0, layers: 0, totalWeight: 0 })

const printerOpts = computed(() =>
  printers.value.map((p: any) => ({
    label: `${p.code} (${p.host}:${p.port})${p.online ? '' : ' 离线'}`,
    value: p.code,
  })))
const templateOpts = computed(() =>
  templates.value.map((t: string) => ({ label: t, value: t.replace(/\.zpl$/i, '') })))

async function loadMeta() {
  loading.value = true
  error.value = ''
  try {
    const [pr, tp]: any[] = await Promise.all([
      http.get('/edge/printers'),
      // 边端没有 /labels/templates 接口（模板是启动时从配置目录加载的），
      // 请求失败时退回到已知的内置模板名，避免下拉框空着没法打印。
      http.get('/labels/templates').catch(() => ({ data: ['pallet.zpl'] })),
    ])
    printers.value = pr.data || []
    templates.value = tp.data || []

    // 只有一台打印机时直接选中，省一次点击
    if (printers.value.length === 1) printerCode.value = printers.value[0].code
    // 默认模板优先 pallet
    if (templates.value.length && !templates.value.some(t => t.startsWith(templateName.value))) {
      templateName.value = templates.value[0].replace(/\.zpl$/i, '')
    }
  } catch (e: any) {
    error.value = e?.response?.data?.message || '读取打印机列表失败'
  } finally {
    loading.value = false
  }
}

watch(visible, (v) => {
  if (!v) return
  loadMeta()
  const p = props.preset || {}
  form.value = {
    palletCode: p.palletCode || '',
    lotCode: p.lotCode || '',
    gradeCode: p.gradeCode || '',
    bobbinCount: p.bobbinCount || 0,
    layers: p.layers || 0,
    totalWeight: p.totalWeight || 0,
  }
})

async function submit() {
  if (!form.value.palletCode.trim()) {
    error.value = '请输入栈板号'
    return
  }
  loading.value = true
  error.value = ''
  try {
    const now = new Date().toLocaleString('zh-CN', { hour12: false })
    await http.post('/edge/print', {
      printer_code: printerCode.value,
      template_name: templateName.value,
      data: {
        PalletCode: form.value.palletCode.trim(),
        LotCode: form.value.lotCode || '-',
        GradeCode: form.value.gradeCode || '-',
        BobbinCount: form.value.bobbinCount,
        Layers: form.value.layers,
        TotalWeight: form.value.totalWeight,
        ProductDate: new Date().toISOString().slice(0, 10),
        Palletizer: 0,
        Timestamp: now,
        // 条码内容默认与栈板号同源：现场扫码枪读的就是它
        BarCode: form.value.palletCode.trim(),
      },
    })
    visible.value = false
  } catch (e: any) {
    error.value = e?.response?.data?.message || '打印失败，检查打印机是否在线'
  } finally {
    loading.value = false
  }
}
</script>
