import axios from 'axios'
import { auth, setToken } from '../store/auth'

const http = axios.create({
  baseURL: '/api',
  timeout: 20000
})

http.interceptors.request.use((cfg) => {
  if (!cfg.skipAuth && auth.token) {
    cfg.headers['X-Admin-Token'] = auth.token
  }
  return cfg
})

http.interceptors.response.use(
  (res) => {
    const body = res.data
    if (body && typeof body.code === 'number' && body.code !== 0) {
      return Promise.reject(new Error(body.errorMsg || '请求失败'))
    }
    return body
  },
  (err) => {
    const status = err.response?.status
    const cfg = err.config || {}
    const msg = err.response?.data?.errorMsg || err.message || '网络错误'
    if (status === 401 && !cfg.skipRedirect) {
      setToken('')
      if (!window.location.pathname.startsWith('/login')) {
        window.location.replace('/login')
      }
    }
    return Promise.reject(new Error(msg))
  }
)

export const api = {
  // 探测服务是否启用访问控制；不携带密钥、不触发跳转
  probe: () => http.get('/admin/overview', { skipAuth: true, skipRedirect: true }).then((r) => r.data),

  overview: (opts = {}) => http.get('/admin/overview', opts).then((r) => r.data),

  listCounters: () => http.get('/admin/counters').then((r) => r.data),

  incrCounter: (id, step = 1) =>
    http.post('/admin/counters/incr', { id, step }).then((r) => r.data),

  setCounter: (id, count) =>
    http.post('/admin/counters/set', { id, count }).then((r) => r.data),

  deleteCounter: (id) => http.post('/admin/counters/delete', { id }).then((r) => r.data),

  createCounter: (count) =>
    http.post('/admin/counters', count === undefined ? {} : { count }).then((r) => r.data)
}
