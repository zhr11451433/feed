<script setup lang="ts">
import { computed, reactive, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'

import AppShell from '../components/AppShell.vue'
import UserAvatar from '../components/UserAvatar.vue'
import { ApiError } from '../api/client'
import * as accountApi from '../api/account'
import * as socialApi from '../api/social'
import type { Account } from '../api/types'

const route = useRoute()
const router = useRouter()
const userId = computed(() => Number(route.params.id))
const tab = computed(() => (route.query.tab === 'following' ? 'following' : 'followers'))

const state = reactive({
  loading: false,
  error: '',
  username: '',
  followers: [] as Account[],
  vloggers: [] as Account[],
  followerCount: 0,
  vloggerCount: 0,
})

const shown = computed(() => (tab.value === 'followers' ? state.followers : state.vloggers))

async function load() {
  if (!Number.isFinite(userId.value) || userId.value <= 0) {
    state.error = '无效的用户 id'
    return
  }
  state.loading = true
  state.error = ''
  try {
    const [user, followersRes, vloggersRes] = await Promise.all([
      accountApi.findByID(userId.value),
      socialApi.getAllFollowers(userId.value),
      socialApi.getAllVloggers(userId.value),
    ])
    state.username = user.username
    state.followers = followersRes.followers ?? []
    state.vloggers = vloggersRes.vloggers ?? []
    state.followerCount = followersRes.follower_count
    state.vloggerCount = vloggersRes.vlogger_count
  } catch (e) {
    state.error = e instanceof ApiError ? e.message : String(e)
  } finally {
    state.loading = false
  }
}

function openTab(next: 'followers' | 'following') {
  void router.replace({ query: { tab: next } })
}

watch(userId, () => void load(), { immediate: true })
</script>

<template>
  <AppShell>
    <div class="card">
      <p class="title">@{{ state.username || '…' }} · #{{ userId }}</p>
      <div class="row">
        <button type="button" class="pill" :class="{ ok: tab === 'followers' }" @click="openTab('followers')">
          粉丝 {{ state.followerCount }}
        </button>
        <button type="button" class="pill" :class="{ ok: tab === 'following' }" @click="openTab('following')">
          关注 {{ state.vloggerCount }}
        </button>
      </div>
    </div>

    <div v-if="state.loading" class="card">加载中…</div>
    <div v-else-if="state.error" class="card">{{ state.error }}</div>
    <div v-else-if="shown.length === 0" class="card">暂无</div>
    <div v-else class="list">
      <RouterLink v-for="u in shown" :key="u.id" class="card user" :to="`/u/${u.id}`">
        <UserAvatar :username="u.username" :id="u.id" :size="40" />
        <span>@{{ u.username }} · #{{ u.id }}</span>
      </RouterLink>
    </div>
  </AppShell>
</template>

<style scoped>
.list {
  display: grid;
  gap: 10px;
  margin-top: 12px;
}
.user {
  display: flex;
  align-items: center;
  gap: 10px;
}
</style>