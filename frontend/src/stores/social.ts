import { defineStore } from 'pinia'
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ApiError } from '../api/client'
import * as socialApi from '../api/social'
import { useAuthStore } from './auth'
import { useToastStore } from './toast'

export const useSocialStore = defineStore('social', () => {
  const auth = useAuthStore()
  const toast = useToastStore()
  const router = useRouter()
  const ids = ref<number[]>([])
  const busy = reactive<Record<string, boolean>>({})
  function isFollowing(accountId: number) {
    return ids.value.includes(accountId)
  }
  async function refresh() {
    if (!auth.isLoggedIn) {
      ids.value = []
      return
    }
    const res = await socialApi.getAllVloggers()
    ids.value = (res.vloggers ?? []).map((a) => a.id)
  }
  async function toggle(vloggerId: number) {
    if (!auth.isLoggedIn) {
      toast.error('请先登录')
      await router.push('/account')
      return
    }
    const key = String(vloggerId)
    if (busy[key]) return
    busy[key] = true
    const was = isFollowing(vloggerId)
    try {
      if (was) await socialApi.unfollow(vloggerId)
      else await socialApi.follow(vloggerId)
      ids.value = was ? ids.value.filter((id) => id !== vloggerId) : ids.value.concat(vloggerId)
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : String(e))
      await refresh()
    } finally {
      busy[key] = false
    }
  }
  return { busy, isFollowing, refresh, toggle }
})

