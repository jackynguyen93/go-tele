import { createRouter, createWebHistory } from 'vue-router'
import Accounts from './views/Accounts.vue'
import Channels from './views/Channels.vue'
import Settings from './views/Settings.vue'

const routes = [
  { path: '/', redirect: '/accounts' },
  { path: '/accounts', name: 'Accounts', component: Accounts },
  { path: '/channels', name: 'Channels', component: Channels },
  { path: '/settings', name: 'Settings', component: Settings }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

export default router
