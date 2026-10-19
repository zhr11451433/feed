import { reactive } from 'vue'
import { useRouter } from 'vue-router'

import { ApiError } from '../api/client'
import * as likeApi from '../api/like'
import { useAuthStore } from '../stores/auth'
import { useToastStore } from '../stores/toast'

export type LikeTarget = {
  id: number
  is_liked: boolean
  likes_count: number
}

export function useLike() {
  const auth = useAuthStore()
  const toast = useToastStore()
  const router = useRouter()
  const busy = reactive<Record<string, boolean>>({})

  async function toggleLike(item: LikeTarget) {
    if (!auth.isLoggedIn) {
      toast.error('请先登录')
      await router.push('/account')
      return
    }
    const key = String(item.id)
    if (busy[key]) return
    busy[key] = true

    const prevLiked = item.is_liked
    const prevCount = item.likes_count
    item.is_liked = !prevLiked
    item.likes_count = Math.max(0, prevCount + (item.is_liked ? 1 : -1))
    try {
      if (item.is_liked) await likeApi.like(item.id)
      else await likeApi.unlike(item.id)
    } catch (e) {
      item.is_liked = prevLiked
      item.likes_count = prevCount
      toast.error(e instanceof ApiError ? e.message : String(e))
    } finally {
      busy[key] = false
    }
  }

  return { busy, toggleLike }
}