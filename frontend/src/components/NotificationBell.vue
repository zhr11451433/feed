<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'

import type { AppNotification } from '../api/types'
import { useNotificationStore } from '../stores/notification'

const notices = useNotificationStore()
const router = useRouter()
const badge = computed(() => (notices.unread > 99 ? '99+' : String(notices.unread)))

function typeLabel(type: string) {
  if (type === 'like') return '点赞'
  if (type === 'comment') return '评论'
  if (type === 'follow') return '关注'
  return type
}

function formatTime(value: string) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  return date.toLocaleString([], { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
}

async function openOne(item: AppNotification) {
  await notices.markOne(item.id)
  notices.open = false
  if ((item.type === 'like' || item.type === 'comment') && item.target_id) {
    await router.push(`/video/${item.target_id}`)
    return
  }
  if (item.type === 'follow' && item.sender_id) await router.push(`/u/${item.sender_id}`)
}
</script>

<template>
  <div class="bell">
    <button class="dy-btn bell-btn" type="button" @click="notices.open = !notices.open">
      通知
      <span v-if="notices.unread > 0" class="badge">{{ badge }}</span>
    </button>

    <Teleport to="body">
      <template v-if="notices.open">
        <button class="backdrop" type="button" aria-label="关闭通知" @click="notices.open = false" />
        <section class="panel">
          <header class="panel-head">
            <span>通知</span>
            <button class="small" type="button" :disabled="notices.unread === 0" @click="notices.markAll">全部已读</button>
          </header>
          <div v-if="notices.items.length === 0" class="empty">暂无通知</div>
          <div v-for="item in notices.items" :key="item.id" class="row" :class="{ unread: !item.is_read }">
            <button class="row-main" type="button" @click="openOne(item)">
              <span class="kind">{{ typeLabel(item.type) }}</span>
              <span class="text">{{ item.content }}</span>
              <span class="time">{{ formatTime(item.created_at) }}</span>
            </button>
            <button class="del" type="button" @click.stop="notices.remove(item.id)">删除</button>
          </div>
        </section>
      </template>
    </Teleport>
  </div>
</template>

<style scoped>
.bell { position: relative; }
.bell-btn { position: relative; }
.badge {
  position: absolute;
  top: -6px;
  right: -6px;
  min-width: 18px;
  height: 18px;
  padding: 0 5px;
  border-radius: 9px;
  background: var(--danger);
  color: white;
  font-size: 11px;
  line-height: 18px;
}
.backdrop { position: fixed; inset: 0; border: 0; background: transparent; padding: 0; }
.panel {
  position: fixed;
  top: 64px;
  right: 16px;
  z-index: 40;
  width: min(360px, 80vw);
  max-height: 420px;
  overflow: auto;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: #121822;
  display: grid;
  gap: 6px;
  padding: 10px;
}
.panel-head { display: flex; justify-content: space-between; align-items: center; }
.row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 8px;
  align-items: center;
}
.row-main {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  gap: 2px 8px;
  min-width: 0;
  text-align: left;
  background: transparent;
  border: 0;
}
.del {
  background: var(--danger);
  border-color: transparent;
  color: white;
  padding: 6px 10px;
}
.row.unread { border-color: rgba(43, 161, 255, 0.45); }
.kind { color: rgba(190, 223, 255, 0.92); font-size: 12px; }
.text { grid-column: 2; }
.time { grid-column: 2; color: var(--muted); font-size: 11px; }
.empty { color: var(--muted); padding: 12px 4px; }
button.small { padding: 6px 10px; }
</style>