<script setup lang="ts">
import { computed, onMounted, reactive, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'

import AppShell from '../components/AppShell.vue'
import UserAvatar from '../components/UserAvatar.vue'
import { ApiError } from '../api/client'
import * as accountApi from '../api/account'
import * as socialApi from '../api/social'
import type { Account, Video } from '../api/types'
import * as videoApi from '../api/video'
import { useAuthStore } from '../stores/auth'
import { useSocialStore } from '../stores/social'
import { useToastStore } from '../stores/toast'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const social = useSocialStore()
const toast = useToastStore()

const userId = computed(() => Number(route.params.id))
const myId = computed(() => auth.claims?.account_id ?? 0)
const isMe = computed(() => myId.value > 0 && myId.value === userId.value)

const state = reactive({
  loading: false,
  error: '',
  user: null as Account | null,
  videoCount: 0,
  totalLikes: 0,
  followerCount: 0,
  vloggerCount: 0,
  videos: [] as Video[],
  followers: [] as Account[],
  vloggers: [] as Account[],
  socialLoading: false,
  socialError: '',
})

const drawer = reactive({ open: false, tab: 'followers' as 'followers' | 'following' })
const listTitle = computed(() => (drawer.tab === 'followers' ? '粉丝' : '关注'))
const listItems = computed(() => (drawer.tab === 'followers' ? state.followers : state.vloggers))

async function loadProfile() {
  if (!Number.isFinite(userId.value) || userId.value <= 0) {
    state.error = '无效的用户 id'
    return
  }
  state.loading = true
  state.error = ''
  try {
    const [profile, videos] = await Promise.all([
      accountApi.getProfile(userId.value),
      videoApi.listByAuthorId(userId.value),
    ])
    state.user = profile.account
    state.videoCount = profile.video_count
    state.totalLikes = profile.total_likes
    state.followerCount = profile.follower_count
    state.vloggerCount = profile.vlogger_count
    state.videos = videos
  } catch (e) {
    state.error = e instanceof ApiError ? e.message : String(e)
    state.user = null
    state.videos = []
  } finally {
    state.loading = false
  }
}

async function openList(tab: 'followers' | 'following') {
  if (!auth.isLoggedIn) {
    toast.error('请先登录')
    await router.push('/account')
    return
  }
  drawer.tab = tab
  drawer.open = true
  state.socialLoading = true
  state.socialError = ''
  try {
    if (tab === 'followers') {
      const res = await socialApi.getAllFollowers(userId.value)
      state.followers = res.followers ?? []
    } else {
      const res = await socialApi.getAllVloggers(userId.value)
      state.vloggers = res.vloggers ?? []
    }
  } catch (e) {
    state.socialError = e instanceof ApiError ? e.message : String(e)
  } finally {
    state.socialLoading = false
  }
}

async function toggleFollow() {
  if (isMe.value) return
  await social.toggle(userId.value)
  await loadProfile()
}

watch(() => route.params.id, () => {
  drawer.open = false
  void loadProfile()
})

onMounted(async () => {
  await social.refresh()
  await loadProfile()
})
</script>

<template>
  <AppShell>
    <div class="card">
      <div class="row" style="justify-content: space-between; align-items: flex-start">
        <div class="row" style="gap: 12px; align-items: center">
          <UserAvatar :username="state.user?.username ?? 'User'" :id="state.user?.id ?? userId" :src="state.user?.avatar_url" :size="64" />
          <div>
            <div class="title" style="margin: 0">@{{ state.user?.username ?? '-' }}</div>
            <div class="subtle">{{ state.user?.bio || '这个人很懒，什么都没写' }}</div>
          </div>
        </div>
        <div class="row">
          <button v-if="isMe" class="ghost" type="button" @click="router.push('/settings')">编辑资料</button>
          <button v-else class="primary" type="button" :disabled="!state.user || state.loading" @click="toggleFollow">
            {{ social.isFollowing(userId) ? '已关注' : '关注' }}
          </button>
        </div>
      </div>
      <div v-if="state.loading" class="hint">加载中…</div>
      <div v-else-if="state.error" class="hint bad">{{ state.error }}</div>
      <div v-else class="row" style="margin-top: 14px">
        <button class="metric" type="button" @click="openList('followers')">
          <div class="metric-num">{{ state.followerCount }}</div>
          <div class="metric-label">粉丝</div>
        </button>
        <button class="metric" type="button" @click="openList('following')">
          <div class="metric-num">{{ state.vloggerCount }}</div>
          <div class="metric-label">关注</div>
        </button>
        <div class="metric static">
          <div class="metric-num">{{ state.videoCount }}</div>
          <div class="metric-label">作品</div>
        </div>
        <div class="metric static">
          <div class="metric-num">{{ state.totalLikes }}</div>
          <div class="metric-label">获赞</div>
        </div>
      </div>
    </div>

    <div class="card" style="margin-top: 14px">
      <p class="title" style="margin: 0">作品</p>
      <div v-if="state.videos.length === 0" class="hint">暂无作品</div>
      <div v-else class="video-grid">
        <button v-for="v in state.videos" :key="v.id" class="video-card" type="button" @click="router.push(`/video/${v.id}`)">
          <img class="video-cover" :src="v.cover_url" :alt="v.title" />
          <div class="video-meta">
            <div class="video-title">{{ v.title }}</div>
            <div class="subtle">❤ {{ v.likes_count }}</div>
          </div>
        </button>
      </div>
    </div>

    <div v-if="drawer.open" class="drawer-backdrop" @click.self="drawer.open = false">
      <div class="drawer">
        <div class="drawer-head">
          <div class="drawer-title">{{ listTitle }}</div>
          <button class="drawer-x" type="button" @click="drawer.open = false">×</button>
        </div>
        <div class="drawer-body">
          <div v-if="state.socialLoading" class="hint">加载中…</div>
          <div v-else-if="state.socialError" class="hint bad">{{ state.socialError }}</div>
          <div v-else-if="listItems.length === 0" class="hint">暂无</div>
          <button v-for="u in listItems" :key="u.id" class="user-row" type="button" @click="drawer.open = false; router.push(`/u/${u.id}`)">
            <UserAvatar :username="u.username" :id="u.id" :src="u.avatar_url" :size="40" />
            <span>@{{ u.username }}</span>
          </button>
        </div>
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

.ghost:hover {
  background: rgba(255, 255, 255, 0.1);
}

.metric {
  border: 1px solid var(--border);
  background: rgba(255, 255, 255, 0.06);
  border-radius: 8px;
  padding: 12px 14px;
  min-width: 120px;
  cursor: pointer;
  display: grid;
  gap: 4px;
  text-align: left;
}

.metric.static {
  cursor: default;
}

.metric:hover {
  background: rgba(255, 255, 255, 0.1);
}

.metric:disabled {
  opacity: 0.55;
  cursor: not-allowed;
}

.metric-num {
  font-size: 18px;
  font-weight: 900;
  letter-spacing: 0.2px;
}

.metric-label {
  font-size: 12px;
  color: var(--muted);
}

.hint {
  color: var(--muted);
}

.hint.bad {
  color: var(--danger);
}

.video-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
}

@media (max-width: 1100px) {
  .video-grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}

@media (max-width: 800px) {
  .video-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

.video-card {
  border: 1px solid var(--border);
  background: rgba(255, 255, 255, 0.05);
  border-radius: 8px;
  overflow: hidden;
  cursor: pointer;
  padding: 0;
  text-align: left;
}

.video-card:hover {
  background: rgba(255, 255, 255, 0.08);
}

.video-cover {
  width: 100%;
  aspect-ratio: 9/12;
  object-fit: cover;
  display: block;
  background: rgba(0, 0, 0, 0.35);
}

.video-meta {
  padding: 10px 10px;
}

.video-title {
  font-weight: 800;
  font-size: 13px;
  overflow: hidden;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
}

.video-sub {
  margin-top: 6px;
  font-size: 12px;
}

.drawer-backdrop {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.55);
  backdrop-filter: blur(10px);
  z-index: 120;
  display: grid;
  justify-items: center;
  align-items: center;
  padding: 16px;
}

.drawer {
  width: min(520px, calc(100vw - 18px));
  max-height: min(78vh, 720px);
  background: rgba(13, 18, 29, 0.92);
  border: 1px solid var(--border);
  border-radius: 8px;
  overflow: hidden;
  display: grid;
  grid-template-rows: auto 1fr;
}

.drawer-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 14px;
  border-bottom: 1px solid var(--border);
}

.drawer-title {
  font-weight: 900;
}

.drawer-x {
  width: 34px;
  height: 34px;
  border-radius: 8px;
  border: 1px solid var(--border);
  background: rgba(255, 255, 255, 0.06);
  color: rgba(255, 255, 255, 0.9);
  cursor: pointer;
  font-size: 20px;
  line-height: 1;
}

.drawer-body {
  overflow: auto;
  padding: 12px 14px;
  display: grid;
  gap: 10px;
}

.drawer-hint {
  color: var(--muted);
  padding: 12px 0;
}

.drawer-hint.bad {
  color: var(--danger);
}

.user-row {
  text-align: left;
  display: grid;
  grid-template-columns: auto 1fr;
  gap: 12px;
  align-items: center;
  padding: 10px 10px;
  border-radius: 8px;
  border: 1px solid var(--border);
  background: rgba(255, 255, 255, 0.05);
  cursor: pointer;
}

.user-row:hover {
  background: rgba(255, 255, 255, 0.08);
}

.user-meta {
  min-width: 0;
}

.user-name {
  font-weight: 800;
}

.user-id {
  font-size: 12px;
  color: var(--muted);
}

.mono {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New', monospace;
}
</style>