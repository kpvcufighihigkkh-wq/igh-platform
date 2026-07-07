<template>
  <n-config-provider :theme="darkTheme" :theme-overrides="themeOverrides">
    <n-message-provider>
      <n-notification-provider>
        <router-view />
      </n-notification-provider>
    </n-message-provider>
  </n-config-provider>
</template>

<script setup lang="ts">
import { darkTheme } from 'naive-ui'
import type { GlobalThemeOverrides } from 'naive-ui'

const themeOverrides: GlobalThemeOverrides = {
  common: {
    primaryColor: '#00d4ff',
    primaryColorHover: '#33ddff',
    primaryColorPressed: '#00a8cc',
    successColor: '#00e676',
    warningColor: '#ffab00',
    errorColor: '#ff1744',
    bodyColor: '#1a1a2e',
    cardColor: '#0f3460',
    textColorBase: '#e0e0e0',
    borderRadius: '6px',
    fontSize: '16px',
  },
}
</script>
