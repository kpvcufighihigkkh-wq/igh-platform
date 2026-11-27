<template>
  <div style="height:100vh;display:flex;align-items:center;justify-content:center;background:var(--paper)">
    <n-spin size="large" />
  </div>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useStation } from '../composables/useStation'

const router = useRouter()
const { identify } = useStation()

onMounted(async () => {
  const info = await identify()
  if (info.identified && info.station) {
    router.replace(`/station/${info.station.station_type}`)
  } else {
    router.replace('/board/overview')
  }
})
</script>
