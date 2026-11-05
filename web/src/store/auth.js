import { reactive } from 'vue'

const TOKEN_KEY = 'jjxt_admin_token'

export const auth = reactive({
  token: localStorage.getItem(TOKEN_KEY) || '',
  // null 表示尚未探测服务端是否启用了访问控制
  authRequired: null,
  checked: false
})

export function setToken(token) {
  auth.token = token || ''
  if (token) {
    localStorage.setItem(TOKEN_KEY, token)
  } else {
    localStorage.removeItem(TOKEN_KEY)
  }
}

export function resetAuth() {
  setToken('')
  auth.checked = false
}
