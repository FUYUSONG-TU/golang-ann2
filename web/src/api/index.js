import axios from 'axios'
import { auth, clearSession } from '../store/auth'

const http = axios.create({
  baseURL: '/api/admin',
  timeout: 20000
})

http.interceptors.request.use((cfg) => {
  if (auth.token) {
    cfg.headers.Authorization = 'Bearer ' + auth.token
  }
  return cfg
})

http.interceptors.response.use(
  (res) => {
    const body = res.data
    if (body && typeof body.code === 'number' && body.code !== 0) {
      return Promise.reject(new Error(body.errorMsg || '请求失败'))
    }
    return body.data
  },
  (err) => {
    const status = err.response?.status
    const cfg = err.config || {}
    const msg = err.response?.data?.errorMsg || err.message || '网络异常，请稍后重试'
    if (status === 401 && !cfg.skipRedirect) {
      clearSession()
      const to = window.location.pathname + window.location.search
      if (!to.startsWith('/login')) {
        window.location.replace('/login?redirect=' + encodeURIComponent(to))
      }
    }
    return Promise.reject(new Error(msg))
  }
)

export const api = {
  // 登录态
  login: (username, password, remember) =>
    http.post('/auth/login', { username, password, remember }),
  logout: () => http.post('/auth/logout', {}),
  profile: () => http.get('/auth/profile', { skipRedirect: true }),
  changePassword: (oldPassword, newPassword) =>
    http.post('/auth/password', { oldPassword, newPassword }),

  // 看板与系统
  dashboard: () => http.get('/dashboard'),
  systemInfo: () => http.get('/system/info'),

  // 账号
  users: {
    list: (params) => http.get('/users', { params }),
    create: (data) => http.post('/users/create', data),
    update: (data) => http.post('/users/update', data),
    remove: (id) => http.post('/users/delete', { id }),
    toggle: (id, status) => http.post('/users/status', { id, status }),
    resetPassword: (id, password) => http.post('/users/reset-password', { id, password })
  },

  // 角色
  roles: {
    list: () => http.get('/roles'),
    permissions: () => http.get('/roles/permissions'),
    save: (data) => http.post('/roles/save', data),
    remove: (id) => http.post('/roles/delete', { id })
  },

  // 日志
  logs: {
    login: (params) => http.get('/logs/login', { params }),
    operation: (params) => http.get('/logs/operation', { params }),
    clear: (scope) => http.post('/logs/clear', { scope })
  },

  // 设置
  configs: {
    list: () => http.get('/configs'),
    save: (items) => http.post('/configs/save', { items })
  },

  // 小程序骨架
  mp: () => http.get('/mp/summary'),

  // 课程内容：专业
  majors: {
    list: (params) => http.get('/majors', { params }),
    save: (data) => http.post('/majors/save', data),
    remove: (id) => http.post('/majors/delete', { id }),
    sort: (ids) => http.post('/majors/sort', { ids })
  },

  // 课程内容：课程
  courses: {
    list: (params) => http.get('/courses', { params }),
    detail: (id) => http.get('/courses/detail', { params: { id } }),
    save: (data) => http.post('/courses/save', data),
    status: (id, status) => http.post('/courses/status', { id, status }),
    remove: (id) => http.post('/courses/delete', { id }),
    sort: (ids) => http.post('/courses/sort', { ids })
  },

  // 课程内容：章节
  chapters: {
    list: (courseId) => http.get('/chapters', { params: { courseId } }),
    save: (data) => http.post('/chapters/save', data),
    remove: (id) => http.post('/chapters/delete', { id }),
    sort: (ids) => http.post('/chapters/sort', { ids })
  },

  // 课程内容：学习路线
  paths: {
    list: (params) => http.get('/paths', { params }),
    detail: (id) => http.get('/paths/detail', { params: { id } }),
    save: (data) => http.post('/paths/save', data),
    status: (id, status) => http.post('/paths/status', { id, status }),
    remove: (id) => http.post('/paths/delete', { id }),
    sort: (ids) => http.post('/paths/sort', { ids }),
    stageSave: (data) => http.post('/path-stages/save', data),
    stageRemove: (id) => http.post('/path-stages/delete', { id }),
    stageSort: (ids) => http.post('/path-stages/sort', { ids }),
    addCourses: (pathId, stageId, courseIds) =>
      http.post('/path-items/add', { pathId, stageId, courseIds }),
    updateItem: (data) => http.post('/path-items/update', data),
    removeItem: (id) => http.post('/path-items/remove', { id }),
    sortItems: (ids) => http.post('/path-items/sort', { ids })
  },

  // AI 知识库：资料
  kb: {
    list: (params) => http.get('/kb', { params }),
    detail: (id) => http.get('/kb/detail', { params: { id } }),
    chunks: (docId) => http.get('/kb/chunks', { params: { docId } }),
    save: (data) => http.post('/kb/save', data),
    parse: (id) => http.post('/kb/parse', { id }),
    status: (id, status) => http.post('/kb/status', { id, status }),
    remove: (id) => http.post('/kb/delete', { id }),
    // 知识规则：领域准入与沉淀闸门
    rule: () => http.get('/kb/rule'),
    saveRule: (data) => http.post('/kb/rule/save', data),
    testRule: (question) => http.post('/kb/rule/test', { question })
  },

  // 问答沉淀：学员真实提问累积成的标准答案
  qa: {
    list: (params) => http.get('/qa', { params }),
    save: (data) => http.post('/qa/save', data),
    status: (id, status) => http.post('/qa/status', { id, status }),
    merge: (fromId, toId) => http.post('/qa/merge', { fromId, toId }),
    bulkPublish: () => http.post('/qa/bulk-publish', {}),
    autoReview: (data) => http.post('/qa/auto-review', data || {}),
    remove: (id) => http.post('/qa/delete', { id }),
    feedback: (id, helpful) => http.post('/qa/feedback', { id, helpful })
  },

  // AI 助手：配置、连通性测试、检索与问答
  ai: {
    configs: () => http.get('/ai/configs'),
    save: (data) => http.post('/ai/configs/save', data),
    test: () => http.post('/ai/test', {}),
    retrieve: (data) => http.post('/ai/retrieve', data),
    // 模型回答可能超过 20 秒，单独放宽超时
    chat: (data) => http.post('/ai/chat', data, { timeout: 90000 })
  },

  // 学员与学习记录
  learners: {
    list: (params) => http.get('/learners', { params }),
    detail: (id) => http.get('/learners/detail', { params: { id } }),
    save: (data) => http.post('/learners/save', data),
    status: (id, status) => http.post('/learners/status', { id, status }),
    remove: (id) => http.post('/learners/delete', { id }),
    cleanDemo: () => http.post('/learners/demo-clean', {})
  },

  // 计数业务
  counters: {
    list: () => http.get('/counters'),
    create: (count) => http.post('/counters/create', count === undefined ? {} : { count }),
    incr: (id, step = 1) => http.post('/counters/incr', { id, step }),
    set: (id, count) => http.post('/counters/set', { id, count }),
    remove: (id) => http.post('/counters/delete', { id })
  }
}
