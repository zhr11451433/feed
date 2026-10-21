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

const me = computed(() => ({
  id: auth.claims?.account_id ?? 0,
  username: auth.claims?.username ?? '',
}))

const rename = reactive({ open: false, newUsername: '' })
const bio = ref('')
const avatarUrl = ref('')

async function loadMe() {
  if (!me.value.id) return
  const profile = await accountApi.getProfile(me.value.id)
  bio.value = profile.account.bio ?? ''
  avatarUrl.value = profile.account.avatar_url ?? ''
}

async function submitRename() {
  const newUsername = rename.newUsername.trim()
  if (!newUsername) {
    toast.error('请输入新用户名')
    return
  }
  busy.value = true
  try {
    const res = await accountApi.rename(newUsername)
    auth.setToken(res.token)
    rename.open = false
    toast.success('改名成功')
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : String(e))
  } finally {
    busy.value = false
  }
}

async function onAvatar(ev: Event) {
  const file = (ev.target as HTMLInputElement).files?.[0]
  if (!file) return
  busy.value = true
  try {
    const res = await accountApi.uploadAvatar(file)
    avatarUrl.value = res.avatar_url
    toast.success('头像已更新')
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : String(e))
  } finally {
    busy.value = false
  }
}

async function saveBio() {
  const text = bio.value.trim()
  if (!text) {
    toast.error('简介不能为空')
    return
  }
  busy.value = true
  try {
    await accountApi.updateProfile({ bio: text })
    toast.success('简介已更新')
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : String(e))
  } finally {
    busy.value = false
  }
}

async function onLogout() {
  if (!window.confirm('确认退出登录？')) return
  busy.value = true
  try {
    await accountApi.logout()
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : String(e))
  } finally {
    auth.clearTokens()
    busy.value = false
    await router.push('/')
  }
}

void loadMe()
</script>

<template>
  <AppShell>
    <div class="card">
      <div class="row" style="gap: 12px; align-items: center">
        <UserAvatar :username="me.username" :id="me.id" :src="avatarUrl" :size="64" />
        <div>
          <div class="title" style="margin: 0">@{{ me.username }}</div>
          <button class="ghost" type="button" @click="router.push(`/u/${me.id}`)">我的主页</button>
        </div>
      </div>

      <div style="margin-top: 14px">
        <label>头像</label>
        <input type="file" accept=".jpg,.jpeg,.png,.webp" :disabled="busy" @change="onAvatar" />
      </div>

      <div style="margin-top: 14px">
        <label>简介</label>
        <input v-model="bio" :disabled="busy" />
        <button class="primary" type="button" :disabled="busy" @click="saveBio">保存简介</button>
      </div>

      <div style="margin-top: 14px">
        <button class="ghost" type="button" :disabled="busy" @click="rename.open = true; rename.newUsername = me.username">改名</button>
        <div v-if="rename.open" class="row">
          <input v-model.trim="rename.newUsername" @keydown.enter="submitRename" />
          <button class="primary" type="button" :disabled="busy" @click="submitRename">提交</button>
        </div>
      </div>

      <div class="row" style="margin-top: 14px">
        <button class="ghost" type="button" @click="router.push('/account/change-password')">修改密码</button>
        <button class="danger" type="button" :disabled="busy" @click="onLogout">退出登录</button>
      </div>
    </div>
  </AppShell>
</template>

<style scoped>
.ghost {
  border: 1px solid var(--border);
  background: rgba(255, 255, 255, 0.055);
  color: var(--text);
  border-radius: 8px;
  padding: 10px 12px;
  cursor: pointer;
}
</style>