import { createRouter, createWebHistory } from 'vue-router'
import { auth, setSession, clearSession, canView } from '../store/auth'
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
        meta: { title: '数据看板', module: 'dashboard' }
      },
      {
        path: 'majors',
        name: 'majors',
        component: () => import('../views/MajorView.vue'),
        meta: { title: '专业管理', module: 'majors' }
      },
      {
        path: 'courses',
        name: 'courses',
        component: () => import('../views/CourseView.vue'),
        meta: { title: '课程管理', module: 'courses' }
      },
      {
        path: 'courses/:id',
        name: 'course-edit',
        component: () => import('../views/CourseEditView.vue'),
        meta: { title: '课程编辑', module: 'courses', activeMenu: '/courses' }
      },
      {
        path: 'paths',
        name: 'paths',
        component: () => import('../views/PathView.vue'),
        meta: { title: '学习路线', module: 'paths' }
      },
      {
        path: 'paths/:id',
        name: 'path-edit',
        component: () => import('../views/PathEditView.vue'),
        meta: { title: '路线编排', module: 'paths', activeMenu: '/paths' }
      },
      {
        path: 'kb',
        name: 'kb',
        component: () => import('../views/KbView.vue'),
        meta: { title: 'AI 资料管理', module: 'kb' }
      },
      {
        path: 'ai/settings',
        name: 'ai-settings',
        component: () => import('../views/AiSettingView.vue'),
        meta: { title: 'AI 配置', module: 'ai', activeMenu: '/ai/settings' }
      },
      {
        path: 'ai/chat',
        name: 'ai-chat',
        component: () => import('../views/AiChatView.vue'),
        meta: { title: 'AI 调试台', module: 'ai', activeMenu: '/ai/chat' }
      },
      {
        path: 'learners',
        name: 'learners',
        component: () => import('../views/LearnerView.vue'),
        meta: { title: '学员与记录', module: 'learners' }
      },
      {
        path: 'learners/:id',
        name: 'learner-detail',
        component: () => import('../views/LearnerDetailView.vue'),
        meta: { title: '学员详情', module: 'learners', activeMenu: '/learners' }
      },
      {
        path: 'users',
        name: 'users',
        component: () => import('../views/UserView.vue'),
        meta: { title: '账号管理', module: 'users' }
      },
      {
        path: 'roles',
        name: 'roles',
        component: () => import('../views/RoleView.vue'),
        meta: { title: '角色权限', module: 'roles' }
      },
      {
        path: 'counters',
        name: 'counters',
        component: () => import('../views/CounterView.vue'),
        meta: { title: '计数管理', module: 'counters' }
      },
      {
        path: 'mini',
        name: 'mini',
        component: () => import('../views/MiniProgramView.vue'),
        meta: { title: '小程序管理', module: 'mp' }
      },
      {
        path: 'logs',
        name: 'logs',
        component: () => import('../views/LogView.vue'),
        meta: { title: '操作日志', module: 'logs' }
      },
      {
        path: 'settings',
        name: 'settings',
        component: () => import('../views/SettingView.vue'),
        meta: { title: '系统设置', module: 'configs' }
      },
      {
        path: 'system',
        name: 'system',
        component: () => import('../views/SystemView.vue'),
        meta: { title: '系统信息', module: 'system' }
      },
      {
        path: 'profile',
        name: 'profile',
        component: () => import('../views/ProfileView.vue'),
        meta: { title: '个人中心' }
      },
      {
        path: '403',
        name: 'forbidden',
        component: () => import('../views/ErrorView.vue'),
        meta: { title: '无访问权限', code: 403 }
      }
    ]
  },
  {
    path: '/:pathMatch(.*)*',
    name: 'notfound',
    component: () => import('../views/ErrorView.vue'),
    meta: { title: '页面不存在', code: 404, guest: true }
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes,
  scrollBehavior: () => ({ top: 0 })
})

router.beforeEach(async (to) => {
  if (to.meta.guest) {
    // 已登录时不再停留在登录页
    if (to.name === 'login' && auth.token) return { path: '/' }
    return true
  }

  if (!auth.token) {
    return { path: '/login', query: { redirect: to.fullPath } }
  }

  if (!auth.loaded) {
    try {
      const data = await api.profile()
      setSession({ user: data.user, role: data.role, permissions: data.permissions })
    } catch {
      clearSession()
      return { path: '/login', query: { redirect: to.fullPath } }
    }
  }

  if (to.meta.module && !canView(to.meta.module)) {
    return { path: '/403' }
  }
  return true
})

router.afterEach((to) => {
  document.title = to.meta.title ? `${to.meta.title} · 机匠学徒` : '机匠学徒 · 后台管理'
})

export default router
