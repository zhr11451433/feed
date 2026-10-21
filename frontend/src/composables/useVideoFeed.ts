import { reactive, ref } from 'vue'

import { ApiError } from '../api/client'
import * as feedApi from '../api/feed'
import type { FeedVideoItem } from '../api/types'
import { useAuthStore } from '../stores/auth'

export type FeedTab = 'recommend' | 'following'

export function useVideoFeed() {
  const auth = useAuthStore()
  const tab = ref<FeedTab>('recommend')
  const cursors = { recommend: 0, following: 0 }
  const state = reactive({
    items: [] as FeedVideoItem[],
    loading: false,
    error: '',
    hasMore: false,
    nextTime: 0,
  })

  async function load(reset: boolean) {
    if (state.loading) return
    if (!reset && !state.hasMore) return
    const key = tab.value
    if (key === 'following' && !auth.isLoggedIn) {
      state.items = []
      state.hasMore = false
      state.nextTime = 0
      state.error = '登录后才能看关注流'
      return
    }
    state.loading = true
    state.error = ''
    try {
      const latest_time = reset ? 0 : cursors[key]
      const res =
        key === 'following'
          ? await feedApi.listByFollowing({ limit: 10, latest_time })
          : await feedApi.listLatest({ limit: 10, latest_time })
      const list = res.video_list ?? []
      cursors[key] = res.next_time
      state.nextTime = res.next_time
      state.hasMore = Boolean(res.has_more) && list.length > 0
      state.items = reset ? list : state.items.concat(list)
    } catch (e) {
      state.error = e instanceof ApiError ? e.message : String(e)
    } finally {
      state.loading = false
    }
  }

  return { tab, state, load }
}