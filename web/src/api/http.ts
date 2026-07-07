import axios from 'axios'
import { ref } from 'vue'

export const moduleDisabled = ref('')

const http = axios.create({
  baseURL: '/api/v1',
  timeout: 10000,
})

http.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

http.interceptors.response.use(
  (res) => { moduleDisabled.value = ''; return res.data },
  (err) => {
    if (err.response?.status === 403 && err.response?.data?.message?.includes('模块未启用')) {
      moduleDisabled.value = err.response.data.message
      return Promise.reject(err)
    }
    if (err.response?.status === 401 && window.location.pathname.startsWith('/admin')) {
      localStorage.removeItem('token')
      window.location.href = '/login'
    }
    return Promise.reject(err)
  }
)

export default http
