import { reactive } from 'vue'

const TOKEN_KEY = 'jjxt_token'
const USER_KEY = 'jjxt_user'

function readUser() {
  try {
    return JSON.parse(localStorage.getItem(USER_KEY) || 'null')
  } catch {
    return null
  }
}

export const auth = reactive({
  token: localStorage.getItem(TOKEN_KEY) || '',
  user: readUser(),
  role: null,
  permissions: [],
  loaded: false
})

export function setSession(payload) {
  auth.token = payload.token || auth.token
  auth.user = payload.user || auth.user
  auth.role = payload.role || auth.role
  auth.permissions = payload.permissions || []
  auth.loaded = true

  if (auth.token) localStorage.setItem(TOKEN_KEY, auth.token)
  if (auth.user) localStorage.setItem(USER_KEY, JSON.stringify(auth.user))
}

export function patchUser(user) {
  auth.user = { ...auth.user, ...user }
  localStorage.setItem(USER_KEY, JSON.stringify(auth.user))
}

export function clearSession() {
  auth.token = ''
  auth.user = null
  auth.role = null
  auth.permissions = []
  auth.loaded = false
  localStorage.removeItem(TOKEN_KEY)
  localStorage.removeItem(USER_KEY)
}

/** 拥有某个权限编码 */
export function can(code) {
  if (!code) return true
  const list = auth.permissions || []
  if (list.includes('*')) return true
  return list.includes(code)
}

/** 可查看某个模块：拥有模块全权限或 .view 权限 */
export function canView(module) {
  return can(module) || can(module + '.view')
}

/** 可写某个模块 */
export function canWrite(module) {
  return can(module)
}
