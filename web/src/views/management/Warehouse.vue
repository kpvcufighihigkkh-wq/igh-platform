<template>
  <div>
    <n-card title="仓库管理">
      <template #header-extra>
        <n-select
          v-model:value="selectedWarehouse"
          :options="warehouseOptions"
          style="width: 200px"
          placeholder="选择仓库"
        />
      </template>

      <div v-if="currentWarehouse" style="margin-bottom: 16px">
        <n-space>
          <n-tag type="info">{{ currentWarehouse.type }}</n-tag>
          <n-tag>{{ currentWarehouse.rows }}行 × {{ currentWarehouse.cols }}列 × {{ currentWarehouse.layers }}层</n-tag>
          <n-tag type="success">空闲: {{ slotStats.empty || 0 }}</n-tag>
          <n-tag type="warning">占用: {{ slotStats.occupied || 0 }}</n-tag>
        </n-space>
      </div>

      <WarehouseGrid
        v-if="currentWarehouse"
        :rows="currentWarehouse.rows"
        :cols="currentWarehouse.cols"
        :layers="currentWarehouse.layers"
        :slots="slots"
        @slot-click="handleSlotClick"
      />
    </n-card>

    <n-modal v-model:show="showSlotDetail" preset="card" title="库位详情" style="width: 360px">
      <div v-if="selectedSlot">
        <p>位置: {{ selectedSlot.row_no }}-{{ selectedSlot.col_no }}-{{ selectedSlot.layer_no }}</p>
        <p>状态: {{ selectedSlot.status }}</p>
        <p v-if="selectedSlot.occupant_type">类型: {{ selectedSlot.occupant_type }}</p>
        <p v-if="selectedSlot.occupant_id">ID: {{ selectedSlot.occupant_id }}</p>
      </div>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import http from '../../api/http'
import WarehouseGrid from '../../components/warehouse/WarehouseGrid.vue'

const warehouses = ref<any[]>([])
const selectedWarehouse = ref<number | null>(null)
const slots = ref<any[]>([])
const slotStats = ref<Record<string, number>>({})
const showSlotDetail = ref(false)
const selectedSlot = ref<any>(null)

const warehouseOptions = computed(() =>
  warehouses.value.map(w => ({ label: w.name, value: w.id }))
)

const currentWarehouse = computed(() =>
  warehouses.value.find(w => w.id === selectedWarehouse.value)
)

onMounted(async () => {
  const res: any = await http.get('/warehouses')
  warehouses.value = res.data || []
  if (warehouses.value.length > 0) {
    selectedWarehouse.value = warehouses.value[0].id
  }
})

watch(selectedWarehouse, async (id) => {
  if (!id) return
  const [slotsRes, statsRes]: any[] = await Promise.all([
    http.get(`/warehouses/${id}/slots`),
    http.get(`/warehouses/${id}/stats`),
  ])
  slots.value = slotsRes.data || []
  slotStats.value = statsRes.data || {}
})

function handleSlotClick(slot: any) {
  selectedSlot.value = slot
  showSlotDetail.value = true
}
</script>
