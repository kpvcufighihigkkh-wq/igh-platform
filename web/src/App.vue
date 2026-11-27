<template>
  <n-config-provider :theme-overrides="themeOverrides">
    <n-message-provider>
      <n-notification-provider>
        <router-view />
      </n-notification-provider>
    </n-message-provider>
  </n-config-provider>
</template>

<script setup lang="ts">
import type { GlobalThemeOverrides } from 'naive-ui'

// IGH 工业主题：暖白工作面 + 石墨结构 + 品牌红作信号色。
// 边端离线运行，字体只用系统内置字体族，不加载网络字体。
const fontBody =
  "'Segoe UI', 'Noto Sans SC', 'PingFang SC', 'Microsoft YaHei', sans-serif"
const fontMono = "'Cascadia Mono', 'Consolas', 'SFMono-Regular', monospace"

const themeOverrides: GlobalThemeOverrides = {
  common: {
    primaryColor: '#e02118',
    primaryColorHover: '#f03a2f',
    primaryColorPressed: '#b2160f',
    primaryColorSuppl: '#e02118',
    infoColor: '#2f5c86',
    successColor: '#2f7d4f',
    warningColor: '#a56a00',
    errorColor: '#b72e24',
    bodyColor: '#f1f0ec',
    cardColor: '#ffffff',
    modalColor: '#ffffff',
    popoverColor: '#ffffff',
    tableColor: '#ffffff',
    inputColor: '#ffffff',
    textColorBase: '#25282b',
    textColor1: '#25282b',
    textColor2: '#3e4245',
    textColor3: '#5a5e62',
    borderColor: '#d9d6cf',
    dividerColor: '#e6e3dc',
    borderRadius: '3px',
    fontSize: '15px',
    fontFamily: fontBody,
    fontFamilyMono: fontMono,
  },
  Layout: {
    siderColor: '#25282b',
    color: '#f1f0ec',
    headerColor: '#ffffff',
    siderBorderColor: '#1b1d1f',
  },
  Menu: {
    itemTextColor: '#c9c7c1',
    itemIconColor: '#c9c7c1',
    itemTextColorHover: '#ffffff',
    itemIconColorHover: '#ffffff',
    itemTextColorActive: '#e02118',
    itemIconColorActive: '#e02118',
    itemTextColorChildActive: '#e02118',
    itemIconColorChildActive: '#e02118',
    itemTextColorActiveHover: '#f03a2f',
    itemIconColorActiveHover: '#f03a2f',
    itemColorHover: 'rgba(255, 255, 255, 0.06)',
    itemColorActive: 'rgba(224, 33, 24, 0.12)',
    itemColorActiveHover: 'rgba(224, 33, 24, 0.16)',
    itemColorActiveCollapsed: 'rgba(224, 33, 24, 0.12)',
    itemHeight: '42px',
  },
  Card: {
    borderColor: '#d9d6cf',
    color: '#ffffff',
    titleFontWeight: '600',
  },
  DataTable: {
    thColor: '#f3f2ee',
    thTextColor: '#5a5e62',
    thFontWeight: '600',
    borderColor: '#e6e3dc',
    tdColorHover: '#faf9f6',
    tdColorStriped: '#faf9f6',
  },
  Input: {
    color: '#ffffff',
    border: '1px solid #c4c0b8',
    borderHover: '1px solid #8a8d8f',
    borderFocus: '1px solid #e02118',
    boxShadowFocus: '0 0 0 2px rgba(224, 33, 24, 0.14)',
  },
  Button: {
    borderRadiusMedium: '3px',
    fontWeight: '600',
  },
  Tag: {
    borderRadius: '2px',
  },
  Descriptions: {
    thColor: '#f3f2ee',
  },
}
</script>
