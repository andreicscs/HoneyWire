import { createRouter, createWebHistory } from 'vue-router'

const routes = [
  {
    path: '/',
    redirect: '/dashboard'
  },
  {
    path: '/dashboard',
    name: 'dashboard',
    component: () => import('./views/Dashboard.vue')
  },
  {
    path: '/fleet',
    name: 'fleet',
    component: () => import('./views/FleetManagement.vue')
  },
  {
    path: '/fleet/node/:id',
    name: 'node-details',
    component: () => import('./views/NodeDetails.vue')
  },
  {
    path: '/settings',
    name: 'settings',
    component: () => import('./views/Settings.vue')
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

export default router