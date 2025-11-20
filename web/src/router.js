import { createRouter, createWebHistory } from 'vue-router'
import Dashboard from './views/Dashboard.vue'
import Positions from './views/Positions.vue'
import Accounts from './views/Accounts.vue'
import Channels from './views/Channels.vue'
import Settings from './views/Settings.vue'

const routes = [
  { path: '/', name: 'Dashboard', component: Dashboard },
  { path: '/positions', name: 'Positions', component: Positions },
  { path: '/accounts', name: 'Accounts', component: Accounts },
  { path: '/channels', name: 'Channels', component: Channels },
  { path: '/settings', name: 'Settings', component: Settings }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

export default router
