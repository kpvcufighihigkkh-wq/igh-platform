<template>
  <div class="login-container">
    <n-card class="login-card" :bordered="false">
      <h1 class="title">IGH 智能制造平台</h1>
      <n-form ref="formRef" :model="form" :rules="rules">
        <n-form-item path="username" label="用户名">
          <n-input v-model:value="form.username" placeholder="请输入用户名" />
        </n-form-item>
        <n-form-item path="password" label="密码">
          <n-input v-model:value="form.password" type="password" placeholder="请输入密码" @keyup.enter="handleLogin" />
        </n-form-item>
        <n-text type="error" v-if="err" style="display:block;margin-bottom:8px">{{ err }}</n-text>
        <n-button type="primary" block :loading="loading" @click="handleLogin">
          登录
        </n-button>
      </n-form>
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import http from '../api/http'

const router = useRouter()
const auth = useAuthStore()
const loading = ref(false)
const err = ref('')
const form = reactive({ username: '', password: '' })
const rules = {
  username: { required: true, message: '请输入用户名' },
  password: { required: true, message: '请输入密码' },
}

async function handleLogin() {
  if (!form.username || !form.password) return
  loading.value = true
  try {
    const res: any = await http.post('/auth/login', { username: form.username, password: form.password })
    auth.setToken(res.data.token)
    localStorage.setItem('role', res.data.role?.code || '')
    localStorage.setItem('userName', res.data.user?.name || '')
    const role = res.data.role?.code || 'admin'
    const roleRoutes: Record<string, string> = {
      admin: '/admin/dashboard',
      winding_op: '/station/winding',
      sorting_op: '/station/sorting',
      packaging_op: '/station/packaging',
      warehouse_op: '/station/warehouse',
      viewer: '/board/overview',
    }
    router.push(roleRoutes[role] || '/admin/dashboard')
  } catch {
    err.value = '用户名或密码错误'
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-container {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100vh;
  background: var(--bg-primary);
}
.login-card {
  width: 400px;
  background: var(--bg-card);
}
.title {
  text-align: center;
  margin-bottom: 24px;
  color: var(--color-primary);
  font-size: 24px;
}
</style>
