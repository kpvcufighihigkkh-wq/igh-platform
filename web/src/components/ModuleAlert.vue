<template>
  <n-alert v-if="moduleDisabled" type="warning" style="margin-bottom:12px">
    {{ moduleDisabled }}，请联系管理员在系统设置中启用对应模块
  </n-alert>
</template>

<script setup lang="ts">
import { moduleDisabled } from '../api/http'
</script>
