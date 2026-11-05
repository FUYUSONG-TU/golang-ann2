import { createRouter, createWebHistory } from 'vue-router'
import { auth, setToken } from '../store/auth'
import { api } from '../api'

const routes = [
  {
    path: '/login',
    name: 'login',
    component: () => import('../views/LoginView.vue'),
    meta: { guest: true, title: '登录' }
  },
  {
    path: '/',
    component: () => import('../layouts/MainLayout.vue'),
    children: [
      {
        path: '',
        name: 'dashboard',
        component: () => import('../views/DashboardView.vue'),
        meta: { title: '仪表盘', icon: 'Odometer' }
      },
      {
        path: 'counters',
        name: 'counters',
        component: () => import('../views/CounterView.vue'),
        meta: { title: '计数管理', icon: 'Tickets' }
      },
      {
        path: 'about',
        name: 'about',
        component: () => import('../views/AboutView.vue'),
        meta: { title: '系统信息', icon: 'InfoFilled' }
      }
    ]
  },
  { path: '/:pathMatch(.*)*', redirect: '/' }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

router.beforeEach(async (to) => {
  if (to.meta.guest) return true

  if (!auth.checked) {
    try {
      await api.probe()
      auth.authRequired = false
      auth.checked = true
      return true
    } catch {
      auth.authRequired = true
      auth.checked = true
      if (!auth.token) {
        return { path: '/login', query: { redirect: to.fullPath } }
      }
      try {
        await api.overview({ skipRedirect: true })
        return true
      } catch {
        setToken('')
        return { path: '/login', query: { redirect: to.fullPath } }
      }
    }
  }

  if (auth.authRequired && !auth.token) {
    return { path: '/login', query: { redirect: to.fullPath } }
  }
  return true
})

router.afterEach((to) => {
  document.title = to.meta.title ? `${to.meta.title} · 机匠学徒` : '机匠学徒 · 后台管理'
})

export default router
