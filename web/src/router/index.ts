import { createRouter, createWebHistory } from 'vue-router'
import type { RouteRecordRaw } from 'vue-router'

const routes: RouteRecordRaw[] = [
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
      { path: 'statistics', name: 'AdminStatistics', component: () => import('../views/management/Statistics.vue') },
      { path: 'devices', name: 'AdminDevices', component: () => import('../views/management/Devices.vue') },
      { path: 'line-config', name: 'AdminLineConfig', component: () => import('../views/admin/LineConfig.vue') },
      { path: 'vehicles', name: 'AdminVehicles', component: () => import('../views/management/Vehicles.vue') },
      { path: 'settings', name: 'AdminSettings', component: () => import('../views/admin/Settings.vue') },
      { path: 'erp', name: 'AdminERP', component: () => import('../views/admin/ERP.vue') },
    ],
  },
  { path: '/', name: 'Root', component: () => import('../views/Entry.vue') },
]

const router = createRouter({ history: createWebHistory(), routes })

router.beforeEach((to, _from, next) => {
  const token = localStorage.getItem('token')
  if (to.path.startsWith('/admin') && !token) {
    next('/login')
  } else {
    next()
  }
})

export default router
