<template>
  <div class="unknown">
    <n-result status="error" title="边端工位类型未配置或配置错误" :description="description">
      <template #footer>
        <n-space vertical size="small">
          <n-text depth="3">检查点：</n-text>
          <n-text depth="3">1. 配置文件 edge.yaml 里的 edge.type 是否为 winding / sorting / packaging / warehouse 之一</n-text>
          <n-text depth="3">2. 修改后需要重启边端服务</n-text>
          <n-text depth="3" v-if="edgeType">当前读到的值：{{ edgeType }}</n-text>
        </n-space>
      </template>
    </n-result>
  </div>
</template>

<script setup lang="ts">
const edgeType = (window as unknown as { __IGH_EDGE__?: { type: string } }).__IGH_EDGE__?.type || ''
const description = edgeType
  ? `未知的工位类型 "${edgeType}"，本机无法确定要显示哪个工位页面。`
  : '服务端没有注入工位类型，可能前端产物与边端二进制版本不匹配。'
</script>

<style scoped>
.unknown {
  height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #1a1a2e;
  padding: 24px;
}
</style>
