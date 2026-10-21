<script setup lang="ts">
import { onMounted, reactive } from 'vue'

import { ApiError } from '../api/client'
import * as commentApi from '../api/comment'
import type { Comment } from '../api/types'
import { useAuthStore } from '../stores/auth'
import { useToastStore } from '../stores/toast'

const props = defineProps<{ videoId: number; title: string }>()
const emit = defineEmits<{ close: [] }>()

const auth = useAuthStore()
const toast = useToastStore()
const drawer = reactive({
  loading: false,
  error: '',
  comments: [] as Comment[],
  content: '',
})

function needLogin() {
  toast.error('请先登录')
}

async function loadComments() {
  drawer.loading = true
  drawer.error = ''
  try {
    drawer.comments = await commentApi.listAll(props.videoId)
  } catch (e) {
    drawer.error = e instanceof ApiError ? e.message : String(e)
  } finally {
    drawer.loading = false
  }
}

async function publishComment() {
  if (!auth.isLoggedIn) return needLogin()
  const content = drawer.content.trim()
  if (!content) return
  drawer.loading = true
  drawer.error = ''
  try {
    await commentApi.publish(props.videoId, content)
    drawer.content = ''
    await loadComments()
    toast.success('评论已发布')
  } catch (e) {
    drawer.error = e instanceof ApiError ? e.message : String(e)
    toast.error(drawer.error)
  } finally {
    drawer.loading = false
  }
}

function canDelete(c: Comment) {
  const myId = auth.claims?.account_id
  return typeof myId === 'number' && myId === c.author_id
}

async function deleteComment(commentId: number) {
  if (!auth.isLoggedIn) return needLogin()
  if (!window.confirm('确认删除这条评论？')) return
  drawer.loading = true
  drawer.error = ''
  try {
    await commentApi.remove(commentId)
    await loadComments()
    toast.info('评论已删除')
  } catch (e) {
    drawer.error = e instanceof ApiError ? e.message : String(e)
    toast.error(drawer.error)
  } finally {
    drawer.loading = false
  }
}

onMounted(() => void loadComments())
</script>

<template>
  <div class="backdrop" @click.self="emit('close')">
    <aside class="drawer">
      <header class="head">
        <div class="title">{{ title }}</div>
        <button type="button" @click="emit('close')">×</button>
      </header>
      <div class="body">
        <p v-if="drawer.loading">加载中…</p>
        <p v-else-if="drawer.error">{{ drawer.error }}</p>
        <p v-else-if="drawer.comments.length === 0">暂无评论</p>
        <article v-for="c in drawer.comments" v-else :key="c.id" class="comment">
          <div class="subtle">{{ c.username }} · {{ new Date(c.created_at).toLocaleString() }}</div>
          <p>{{ c.content }}</p>
          <button v-if="canDelete(c)" type="button" :disabled="drawer.loading" @click="deleteComment(c.id)">删除</button>
        </article>
      </div>
      <footer class="foot">
        <textarea v-model="drawer.content" placeholder="说点什么…" :disabled="drawer.loading" />
        <button type="button" class="primary" :disabled="drawer.loading || !drawer.content.trim()" @click="publishComment">
          发送
        </button>
      </footer>
    </aside>
  </div>
</template>

<style scoped>
.backdrop {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.55);
  z-index: 20;
  display: grid;
  justify-items: end;
}
.drawer {
  width: min(420px, 100vw);
  height: 100vh;
  background: #141922;
  border-left: 1px solid var(--border);
  display: grid;
  grid-template-rows: auto 1fr auto;
}
.head,
.foot {
  padding: 12px 14px;
  border-bottom: 1px solid var(--border);
}
.foot {
  border-bottom: 0;
  border-top: 1px solid var(--border);
}
.body {
  overflow: auto;
  padding: 12px 14px;
  display: grid;
  gap: 10px;
  align-content: start;
}
.comment {
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 10px;
}
.title {
  font-weight: 800;
}
textarea {
  width: 100%;
  min-height: 72px;
  margin-bottom: 8px;
}
</style>