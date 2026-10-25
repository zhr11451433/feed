import { defineStore } from 'pinia'
import { ref, watch } from 'vue'

import * as notificationApi from '../api/notification'
import type { AppNotification } from '../api/types'
import { useAuthStore } from './auth'

export const useNotificationStore = defineStore('notification', () => {
    const auth = useAuthStore()
    const items = ref<AppNotification[]>([])
    const unread = ref(0)
    const open = ref(false)
    let source: EventSource | null = null
    let started = false

    function closeStream() {
        source?.close()
        source = null
    }

    function openStream() {
        closeStream()
        if (!auth.token) return
        source = new EventSource(notificationApi.streamUrl(auth.token))
        source.onmessage = (ev) => {
            let n: AppNotification
            try {
                n = JSON.parse(ev.data) as AppNotification
            } catch {
                return
            }
            if (!n.id || items.value.some((item) => item.id === n.id)) return
            items.value = [n, ...items.value].slice(0, 50)
            if (!n.is_read) unread.value += 1
        }
        // 断线后浏览器会自己重连。这里关掉的话，重连就没了。
    }

    async function refresh() {
        if (!auth.token) {
            items.value = []
            unread.value = 0
            return
        }
        const [list, count] = await Promise.all([
            notificationApi.listNotifications(),
            notificationApi.unreadCount(),
        ])
        const fresh = list.notifications ?? []
        const seen = new Set(fresh.map((item) => item.id))
        const pending = items.value.filter((item) => !seen.has(item.id))
        items.value = [...pending, ...fresh].slice(0, 50)
        unread.value = count.count ?? 0
    }

    function start() {
        if (started) return
        started = true
        watch(
            () => auth.token,
            () => {
                open.value = false
                if (!auth.token) {
                    closeStream()
                    items.value = []
                    unread.value = 0
                    return
                }
                void refresh()
                openStream()
            },
            { immediate: true },
        )
    }

    async function markAll() {
        await notificationApi.markRead()
        items.value = items.value.map((item) => ({ ...item, is_read: true }))
        unread.value = 0
    }

    async function markOne(id: number) {
        await notificationApi.markRead(id)
        const item = items.value.find((n) => n.id === id)
        if (item && !item.is_read) {
            item.is_read = true
            unread.value = Math.max(0, unread.value - 1)
        }
    }

    async function remove(id: number) {
        await notificationApi.deleteNotification(id)
        const item = items.value.find((n) => n.id === id)
        if (item && !item.is_read) unread.value = Math.max(0, unread.value - 1)
        items.value = items.value.filter((n) => n.id !== id)
    }
    return { items, unread, open, start, refresh, markAll, markOne ,remove}
})

