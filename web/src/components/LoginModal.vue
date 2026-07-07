<template>
  <n-modal v-model:show="showLogin" preset="card" title="操作需要登录" style="width:340px" :mask-closable="true">
    <n-form @submit.prevent="handleSubmit">
      <n-form-item label="用户名"><n-input v-model:value="form.username" placeholder="请输入用户名" /></n-form-item>
      <n-form-item label="密码"><n-input v-model:value="form.password" type="password" placeholder="请输入密码" @keyup.enter="handleSubmit" /></n-form-item>
      <n-text type="error" v-if="err" style="display:block;margin-bottom:8px">{{ err }}</n-text>
      <n-button type="primary" block :loading="loading" @click="handleSubmit">登录</n-button>
    </n-form>
  </n-modal>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { useAuth } from '../composables/useAuth'

const { showLogin, doLogin } = useAuth()
const form = reactive({ username: '', password: '' })
const loading = ref(false)
const err = ref('')

async function handleSubmit() {
  if (!form.username || !form.password) return
  loading.value = true
  err.value = ''
  const e = await doLogin(form.username, form.password)
  if (e) err.value = e
  loading.value = false
}
</script>
