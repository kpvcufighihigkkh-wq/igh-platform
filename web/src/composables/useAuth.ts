import { ref } from 'vue'
import http from '../api/http'

const token = ref(localStorage.getItem('token') || '')
const showLogin = ref(false)
const userName = ref(localStorage.getItem('userName') || '')
let pendingAction: (() => void) | null = null

export function useAuth() {
  function isLoggedIn() { return !!token.value }

  function requireAuth(action: () => void) {
    if (token.value) { action(); return }
    pendingAction = action
    showLogin.value = true
  }

  async function doLogin(username: string, password: string): Promise<string | null> {
    try {
      const res: any = await http.post('/auth/login', { username, password })
      token.value = res.data.token
      userName.value = res.data.user?.name || username
      localStorage.setItem('token', res.data.token)
      localStorage.setItem('userName', userName.value)
      showLogin.value = false
      if (pendingAction) { pendingAction(); pendingAction = null }
      return null
    } catch {
      return '用户名或密码错误'
    }
  }

  function logout() {
    token.value = ''
    userName.value = ''
    localStorage.removeItem('token')
    localStorage.removeItem('userName')
  }

  return { token, userName, showLogin, isLoggedIn, requireAuth, doLogin, logout }
}
