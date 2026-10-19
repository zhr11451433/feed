<script setup lang="ts">
import { computed, reactive, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import * as likeApi from '../api/like'
import { useLike } from '../composables/useLike'
import AppShell from '../components/AppShell.vue'
import UserAvatar from '../components/UserAvatar.vue'
import { ApiError } from '../api/client'
import * as videoApi from '../api/video'
import type { Video } from '../api/types'

const route = useRoute()
const id = computed(() => Number(route.params.id))
const auth = useAuthStore()
const { busy, toggleLike } = useLike()
const likeState = reactive({ id: 0, is_liked: false, likes_count: 0 })
const state = reactive({
  loading: false,
  error: '',
  video: null as Video | null,
})

async function loadVideo() {
  if (!Number.isFinite(id.value) || id.value <= 0) {
    state.error = '无效的 video id'
    state.video = null
    return
  }
  state.loading = true
  state.error = ''
  try {
    state.video = await videoApi.getDetail(id.value)
    likeState.id = state.video.id
    likeState.likes_count = state.video.likes_count
    likeState.is_liked = false
    if (auth.isLoggedIn) {
      const res = await likeApi.isLiked(state.video.id)
      likeState.is_liked = res.is_liked
}
  } catch (e) {
    state.video = null
    state.error = e instanceof ApiError ? e.message : String(e)
  } finally {
    state.loading = false
  }
}

watch(id, () => void loadVideo(), { immediate: true })
</script>

<template>
  <AppShell>
    <div v-if="state.loading" class="card">加载中…</div>
    <div v-else-if="state.error" class="card">{{ state.error }}</div>
    <div v-else-if="state.video" class="card">
      <video class="player" :src="state.video.play_url" :poster="state.video.cover_url" controls playsinline />
      <div class="row" style="margin-top: 12px; gap: 12px; align-items: center">
        <UserAvatar :username="state.video.username" :id="state.video.author_id" :size="40" />
        <div>
          <div class="title" style="margin: 0">{{ state.video.title }}</div>
          <div class="subtle">@{{ state.video.username }} · #{{ state.video.id }}</div>
        </div>
      </div>
      <p v-if="state.video.description" class="subtle">{{ state.video.description }}</p>
      <button class="pill" type="button" :disabled="!!busy[String(likeState.id)]" @click="toggleLike(likeState)">
  {{ likeState.is_liked ? '已赞' : '点赞' }} {{ likeState.likes_count }}
</button>
    </div>
  </AppShell>
</template>

<style scoped>
.player {
  width: min(420px, 100%);
  max-height: 70vh;
  background: #000;
  border-radius: 12px;
}
</style>