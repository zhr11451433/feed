<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'

import AppShell from '../components/AppShell.vue'
import UserAvatar from '../components/UserAvatar.vue'
import { ApiError } from '../api/client'
import * as accountApi from '../api/account'
import { useAuthStore } from '../stores/auth'
import { useToastStore } from '../stores/toast'

const router = useRouter()
const auth = useAuthStore()
const toast = useToastStore()

const busy = ref(false)
const loginForm = reactive({ username: '', password: '' })

const me = computed(() => ({
  id: auth.claims?.account_id ?? 0,
  username: auth.claims?.username ?? '',
}))

async function onLogin() {
  if (busy.value) return
  const username = loginForm.username.trim()
  const password = loginForm.password.trim()
  if (!username || !password) {
    toast.error('请输入用户名和密码')
    return
  }
  busy.value = true
  try {
    const res = await accountApi.login(username, password)
    auth.setTokens(res.token, res.refresh_token ?? '')
    toast.success('登录成功')
  } catch (e) {
    const msg = e instanceof ApiError ? e.message : String(e)
    toast.error(msg)
  } finally {
    busy.value = false
  }
}

async function goRegister() {
  await router.push('/account/register')
}
</script>

<template>
  <AppShell>
    <div v-if="!auth.isLoggedIn" class="login-wrap">
      <div class="card login-card">
        <p class="title">登录</p>
        <div class="grid" style="margin-top: 10px">
          <div>
            <label>username</label>
            <input v-model.trim="loginForm.username" autocomplete="username" />
          </div>
          <div>
            <label>password</label>
            <input v-model.trim="loginForm.password" type="password" autocomplete="current-password" @keydown.enter="onLogin" />
          </div>
          <button class="primary" type="button" :disabled="busy" @click="onLogin">登录</button>
        </div>
        <div class="row" style="justify-content: flex-end; margin-top: 14px">
          <button class="ghost" type="button" :disabled="busy" @click="goRegister">注册账号</button>
        </div>
      </div>
    </div>

    <div v-else class="card">
      <div class="row" style="gap: 12px; align-items: center">
        <UserAvatar :username="me.username" :id="me.id" :size="64" />
        <div>
          <div class="title" style="margin: 0">@{{ me.username }}</div>
          <div class="subtle mono">#{{ me.id }}</div>
        </div>
      </div>
      <div class="row" style="margin-top: 12px">
  <button class="primary" type="button" @click="router.push(`/u/${me.id}`)">我的主页</button>
  <button class="ghost" type="button" @click="router.push('/settings')">设置</button>
</div>
    </div>
  </AppShell>
</template>

<style scoped>
.login-wrap {
  display: grid;
  justify-items: center;
  align-content: start;
  padding: clamp(56px, 14vh, 160px) 16px 40px;
}
.login-card {
  width: min(420px, 100%);
}
.ghost {
  border: 1px solid var(--border);
  background: rgba(255, 255, 255, 0.055);
  color: var(--text);
  border-radius: 8px;
  padding: 10px 12px;
  cursor: pointer;
}
</style>