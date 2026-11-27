<template>
  <div class="login-shell">
    <!-- 左侧品牌面：石墨控制柜 + 白色铭牌上的 IGH 标识 -->
    <aside class="brand-panel">
      <div class="brand-plate">
        <img src="/igh-logo.jpg" alt="IGH S.r.l. — Irico Gualchierani Handling" />
      </div>

      <div class="brand-copy">
        <span class="signal-dot" aria-hidden="true"></span>
        <h1>智能制造平台</h1>
        <p>卷绕 · 外检 · 包装 · 仓储，全流程实时协同</p>
      </div>

      <div class="brand-foot">Irico Gualchierani Handling S.r.l.</div>
    </aside>

    <!-- 右侧操作面：暖白工作面，与工位页同一套视觉语言 -->
    <main class="form-panel">
      <div class="form-card">
        <div class="form-head">
          <h2>登录</h2>
          <p>华祥智纤（九江） · HX-JJ-001</p>
        </div>

        <n-form ref="formRef" :model="form" :rules="rules">
          <n-form-item path="username" label="用户名">
            <n-input v-model:value="form.username" placeholder="请输入用户名" @keyup.enter="handleLogin" />
          </n-form-item>
          <n-form-item path="password" label="密码">
            <n-input v-model:value="form.password" type="password" placeholder="请输入密码" @keyup.enter="handleLogin" />
          </n-form-item>
          <n-text type="error" v-if="err" style="display:block;margin-bottom:8px">{{ err }}</n-text>
          <n-button type="primary" block size="large" :loading="loading" @click="handleLogin">
            登录
          </n-button>
        </n-form>
      </div>
    </main>
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
  err.value = ''
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
.login-shell {
  display: grid;
  grid-template-columns: minmax(360px, 5fr) 6fr;
  height: 100vh;
  background: var(--paper);
}

/* ---- 品牌面 ---- */
.brand-panel {
  position: relative;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  padding: 56px 48px 40px;
  background: var(--graphite);
  color: #eceae5;
  overflow: hidden;
}

/* 标识下方的红色细线与圆点，引用 IGH 标识里的波形与圆点元素 */
.brand-panel::after {
  content: '';
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
  height: 6px;
  background: linear-gradient(
    90deg,
    var(--igh-red) 0,
    var(--igh-red) 38%,
    transparent 38%
  );
  opacity: 0.9;
}

.brand-plate {
  width: min(340px, 78%);
  background: #fff;
  padding: 18px 20px;
  border-radius: 2px;
}

.brand-plate img {
  display: block;
  width: 100%;
  height: auto;
}

.brand-copy {
  position: relative;
  padding-left: 20px;
  max-width: 420px;
}

/* 标题左侧的竖直红条：机台铭牌的刻线感 */
.brand-copy::before {
  content: '';
  position: absolute;
  left: 0;
  top: 4px;
  bottom: 4px;
  width: 3px;
  background: var(--igh-red);
}

.signal-dot {
  position: absolute;
  left: 20px;
  top: -26px;
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: var(--igh-red);
}

.brand-copy h1 {
  margin: 0 0 10px;
  font-family: var(--font-display);
  font-size: 40px;
  font-weight: 600;
  letter-spacing: 0.02em;
  line-height: 1.15;
}

.brand-copy p {
  margin: 0;
  font-size: 15px;
  color: #a9a6a0;
  line-height: 1.7;
}

.brand-foot {
  font-family: var(--font-mono);
  font-size: 12px;
  color: #7c7f82;
  letter-spacing: 0.04em;
}

/* ---- 操作面 ---- */
.form-panel {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 32px;
}

.form-card {
  width: min(400px, 100%);
  background: var(--plate);
  border: 1px solid var(--line);
  border-radius: 2px;
  padding: 36px 36px 32px;
}

.form-head {
  margin-bottom: 26px;
  padding-bottom: 18px;
  border-bottom: 1px solid var(--line-soft);
}

.form-head h2 {
  margin: 0 0 6px;
  font-family: var(--font-display);
  font-size: 26px;
  font-weight: 600;
  color: var(--graphite);
}

.form-head p {
  margin: 0;
  font-size: 13px;
  color: var(--steel);
}

@media (max-width: 860px) {
  .login-shell {
    grid-template-columns: 1fr;
    grid-template-rows: auto 1fr;
  }
  .brand-panel {
    padding: 24px;
  }
  .brand-plate {
    width: 200px;
  }
  .brand-copy h1 {
    font-size: 26px;
  }
  .brand-foot {
    display: none;
  }
}
</style>
