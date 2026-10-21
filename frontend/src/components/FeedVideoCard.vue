<script setup lang="ts">
import { RouterLink } from 'vue-router'

import UserAvatar from './UserAvatar.vue'
import type { FeedVideoItem } from '../api/types'
import { useAuthStore } from '../stores/auth'
import { useSocialStore } from '../stores/social'

const auth = useAuthStore()
const social = useSocialStore()
const props = defineProps<{ item: FeedVideoItem; busy?: boolean }>()
const emit = defineEmits<{ toggleLike: [item: FeedVideoItem] }>()
</script>

<template>
  <article class="feed-card">
    <RouterLink class="cover" :to="`/video/${item.id}`">
      <img :src="item.cover_url" :alt="item.title" loading="lazy" />
    </RouterLink>
    <div class="content">
      <RouterLink class="title" :to="`/video/${item.id}`">{{ item.title }}</RouterLink>
      <div class="subtle author">
        <UserAvatar :username="item.author.username" :id="item.author.id" :size="28" />
        <RouterLink :to="`/u/${item.author.id}`">@{{ item.author.username }} · #{{ item.author.id }}</RouterLink>
      </div>
      <p v-if="item.description" class="subtle">{{ item.description }}</p>
      <div class="row">
        <button class="pill" type="button" :disabled="props.busy" @click="emit('toggleLike', props.item)">
  {{ item.is_liked ? '已赞' : '点赞' }} {{ item.likes_count }}
</button>
        <button
  v-if="auth.claims?.account_id !== item.author.id"
  class="pill"
  type="button"
  :disabled="!!social.busy[String(item.author.id)]"
  @click="social.toggle(item.author.id)"
>
  {{ social.isFollowing(item.author.id) ? '已关注' : '关注' }}
</button>
        <RouterLink class="pill" :to="`/video/${item.id}`">播放</RouterLink>
      </div>
    </div>
  </article>
</template>

<style scoped>
.feed-card {
  display: grid;
  grid-template-columns: 240px minmax(0, 1fr);
  gap: 14px;
  border: 1px solid var(--border);
  background: var(--panel);
  border-radius: 12px;
  overflow: hidden;
}
.cover {
  background: #000;
  aspect-ratio: 16 / 9;
}
.cover img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}
.content {
  padding: 12px 14px 14px;
}
.author {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 6px 0;
}
@media (max-width: 900px) {
  .feed-card {
    grid-template-columns: 1fr;
  }
}
</style>