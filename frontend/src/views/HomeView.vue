<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useLike } from '../composables/useLike'
import AppShell from '../components/AppShell.vue'
import FeedVideoCard from '../components/FeedVideoCard.vue'
import { useVideoFeed } from '../composables/useVideoFeed'

const route = useRoute()
import { useSocialStore } from '../stores/social'

const social = useSocialStore()
const { tab, state, load } = useVideoFeed()

watch(tab, () => void load(true))
const { busy, toggleLike } = useLike()
//我要创建一个响应式引用，它里面存放的值应该是 HTMLElement 或 null
//ref<HTMLElement | null>(null) 里面的泛型不是在约定 HTML 元素有哪些属性，
// 而是在告诉 TypeScript：这个 ref 内部的值允许是什么类型
const sentinel = ref<HTMLElement | null>(null)
let observer: IntersectionObserver | null = null
///videos?q=cat q.value=cat
const q = computed(() => (typeof route.query.q === 'string' ? route.query.q.trim().toLowerCase() : ''))
//如果没有关键词，就显示全部视频；有关键词，就筛选标题或作者用户名包含该关键词的视频
const shown = computed(() => {
  if (!q.value) return state.items
  return state.items.filter(
    (v) => v.title.toLowerCase().includes(q.value) || v.author.username.toLowerCase().includes(q.value),
  )
})
//判断哨兵是否可见
function sentinelVisible() {
  const el = sentinel.value
  if (!el) return false
  const root = el.closest('.dy-content')
  const er = el.getBoundingClientRect()
  const rr = root ? root.getBoundingClientRect() : { top: 0, bottom: window.innerHeight }
  return er.top < rr.bottom && er.bottom > rr.top
}
//bindObserver：开始监视
// 这里的流程是：
// 1. 如果旧观察器存在，就断开它。
// 2. 清空旧的观察器变量。
// 3. 获取当前哨兵 DOM。
// 4. 如果 DOM 不存在，就直接退出。
// 5. 创建一个新的观察器。
// 6. 观察哨兵元素。
// observer.observe(el) 才是正式告诉浏览器：“请监视这个元素。”
function bindObserver() {
  observer?.disconnect()
  observer = null
  const el = sentinel.value
  if (!el) return
  observer = new IntersectionObserver(
    (entries) => {
      if (entries.some((e) => e.isIntersecting)) void load(false)
    },
    {
      // 【未学】root = 真正滚动的祖先；rootMargin 让哨兵还差 160px 进视野时就提前请求
      root: el.closest('.dy-content'),
      rootMargin: '160px',
    },
  )
  observer.observe(el)
}
//组件挂载以后做什么？
onMounted(async () => {
  await social.refresh()
  await load(true)
  await nextTick()
  bindObserver()
})

watch(sentinel, async () => {
  await nextTick()
  bindObserver()
})

// 第一页不满一屏时，回调可能撞上 loading 被丢掉，这里补拉
watch(
  () => state.loading,
  async (loading, wasLoading) => {
    if (!wasLoading || loading || !state.hasMore) return
    await nextTick()
    if (sentinelVisible()) void load(false)
  },
)

onBeforeUnmount(() => observer?.disconnect())
</script>

<template>
  <AppShell>
    <div class="card">
  <div class="row">
    <button type="button" class="pill" :class="{ ok: tab === 'recommend' }" @click="tab = 'recommend'">推荐</button>
    <button type="button" class="pill" :class="{ ok: tab === 'following' }" @click="tab = 'following'">关注</button>
  </div>
  <p class="subtle">
    {{ tab === 'following' ? '只看我关注的作者。翻页用返回的 next_time，单位是秒。' : '按发布时间从新到旧。' }}
  </p>
</div>

    <div v-if="state.loading && state.items.length === 0" class="card">加载中…</div>
    <div v-else-if="state.error && state.items.length === 0" class="card">{{ state.error }}</div>
    <div v-else-if="shown.length === 0" class="card">
  {{ tab === 'following' ? '关注的人还没有视频' : '没有视频。先去「发布」传一条。' }}
</div>

    <div class="list">
      <FeedVideoCard
  v-for="item in shown"
  :key="item.id"
  :item="item"
  :busy="!!busy[String(item.id)]"
  @toggle-like="toggleLike"
/>
    </div>

    <div ref="sentinel" class="sentinel">
      <span v-if="state.loading && state.items.length > 0">加载更多…</span>
      <span v-else-if="state.error && state.items.length > 0">{{ state.error }}</span>
      <span v-else-if="!state.hasMore && state.items.length > 0">没有更多了</span>
    </div>
  </AppShell>
</template>

<style scoped>
.list {
  display: grid;
  gap: 12px;
  margin-top: 12px;
}
.sentinel {
  height: 48px;
  display: grid;
  place-items: center;
  color: var(--muted);
  font-size: 13px;
}
</style>