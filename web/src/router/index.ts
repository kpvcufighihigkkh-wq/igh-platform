import { createRouter, createWebHistory } from 'vue-router'
import type { RouteRecordRaw } from 'vue-router'

// 边端二进制会往 index.html 注入 window.__IGH_EDGE__ = { type: "winding" }。
// 管理端（中心托管或 Vite 开发）没有这个变量，走完整路由表。
//
// 为什么要在路由层拦截，而不是只靠页面里提示：
// 边端没有 /api/v1/auth/login，也没有仓库/包装模块的接口。
// 让工人点进 /admin 或别的工位页，得到的是一堆 404，
// 既看不出原因，也会让人以为系统坏了。
interface EdgeMarker {
  type: string
}

const edge = (window as unknown as { __IGH_EDGE__?: EdgeMarker }).__IGH_EDGE__
const edgeType = edge?.type || ''

// 工位类型 → 组件。必须是静态映射：Vite 要在构建期分析出
// 哪些模块被打包，`import(`../views/station/${x}.vue`)` 这种拼出来的
// 路径在生产构建里会变成空实现，开发时正常、打包后白屏。
const stationPages: Record<string, { path: string; name: string; load: () => Promise<unknown> }> = {
  winding: { path: 'winding', name: 'StationWinding', load: () => import('../views/station/Winding.vue') },
  sorting: { path: 'sorting', name: 'StationSorting', load: () => import('../views/station/Sorting.vue') },
  packaging: { path: 'packaging', name: 'StationPackaging', load: () => import('../views/station/Packaging.vue') },
  warehouse: { path: 'warehouse', name: 'StationWarehouse', load: () => import('../views/station/WarehouseOp.vue') },
}

// 边端只需要本工位页面 + 一个兜底。
// 根路径直接进工位页，省掉 Entry.vue 那次识别请求（边端的工位类型是启动时
// 由配置决定的，不需要按 IP 去猜）。
const edgeRoutes: RouteRecordRaw[] = (() => {
  const page = stationPages[edgeType]
  if (!page) {
    // 配置里写了未知的边端类型：不要静默跳到某个页面，
    // 把错误显示出来，现场才知道是 edge.yaml 的 type 写错了。
    return [
      {
        path: '/:pathMatch(.*)*',
        name: 'UnknownStation',
        component: () => import('../views/station/UnknownType.vue'),
      },
    ]
  }
  return [
    { path: '/', redirect: `/station/${page.path}` },
    {
      path: '/station',
      children: [{ path: page.path, name: page.name, component: page.load }],
    },
    { path: '/:pathMatch(.*)*', redirect: `/station/${page.path}` },
  ]
})()

const adminRoutes: RouteRecordRaw[] = [
  { path: '/login', name: 'Login', component: () => import('../views/Login.vue') },
  {
    path: '/board',
    children: [
      { path: 'overview', name: 'BoardOverview', component: () => import('../views/board/Overview.vue') },
      { path: 'warehouse', name: 'BoardWarehouse', component: () => import('../views/board/Warehouse.vue') },
      { path: 'production', name: 'BoardProduction', component: () => import('../views/board/Production.vue') },
    ],
  },
  {
    path: '/station',
    children: [
      { path: 'winding', name: 'StationWinding', component: () => import('../views/station/Winding.vue') },
      { path: 'sorting', name: 'StationSorting', component: () => import('../views/station/Sorting.vue') },
      { path: 'packaging', name: 'StationPackaging', component: () => import('../views/station/Packaging.vue') },
      { path: 'warehouse', name: 'StationWarehouse', component: () => import('../views/station/WarehouseOp.vue') },
    ],
  },
  {
    path: '/admin',
    component: () => import('../views/admin/Layout.vue'),
    children: [
      { path: '', redirect: '/admin/dashboard' },
      { path: 'dashboard', name: 'AdminDashboard', component: () => import('../views/admin/Dashboard.vue') },
      { path: 'production', name: 'AdminProduction', component: () => import('../views/management/Production.vue') },
      { path: 'quality', name: 'AdminQuality', component: () => import('../views/management/Quality.vue') },
      { path: 'orders', name: 'AdminOrders', component: () => import('../views/management/Orders.vue') },
      { path: 'warehouse', name: 'AdminWarehouse', component: () => import('../views/management/Warehouse.vue') },
      { path: 'maintenance', name: 'AdminMaintenance', component: () => import('../views/management/Maintenance.vue') },
      { path: 'alarms', name: 'AdminAlarms', component: () => import('../views/management/Alarms.vue') },
      { path: 'trace', name: 'AdminTrace', component: () => import('../views/management/Trace.vue') },
      { path: 'statistics', name: 'AdminStatistics', component: () => import('../views/management/Statistics.vue') },
      { path: 'devices', name: 'AdminDevices', component: () => import('../views/management/Devices.vue') },
      { path: 'line-config', name: 'AdminLineConfig', component: () => import('../views/admin/LineConfig.vue') },
      { path: 'site-config', name: 'AdminSiteConfig', component: () => import('../views/admin/SiteConfig.vue') },
      { path: 'vehicles', name: 'AdminVehicles', component: () => import('../views/management/Vehicles.vue') },
      { path: 'settings', name: 'AdminSettings', component: () => import('../views/admin/Settings.vue') },
      { path: 'erp', name: 'AdminERP', component: () => import('../views/admin/ERP.vue') },
    ],
  },
  { path: '/', name: 'Root', component: () => import('../views/Entry.vue') },
]

const router = createRouter({
  history: createWebHistory(),
  routes: edgeType ? edgeRoutes : adminRoutes,
})

router.beforeEach((to, _from, next) => {
  // 边端不校验登录态：工位页面本就不需要 token
  if (edgeType) {
    next()
    return
  }
  const token = localStorage.getItem('token')
  if (to.path.startsWith('/admin') && !token) {
    next('/login')
  } else {
    next()
  }
})

export default router
