import { postJson } from './client'
import type { ListNotificationsResponse, MessageResponse, UnreadCountResponse } from './types'

export function listNotifications() {
    return postJson<ListNotificationsResponse>('/notification/list', {}, { authRequired: true })
}

export function unreadCount() {
    return postJson<UnreadCountResponse>('/notification/unreadCount', {}, { authRequired: true })
}

// 不传 id：后端把当前用户的通知全部标已读
export function markRead(id?: number) {
    return postJson<MessageResponse>('/notification/markRead', id ? { id } : {}, { authRequired: true })
}

export function streamUrl(token: string) {
    const base = (import.meta.env.VITE_API_BASE as string | undefined) ?? '/api'
    return `${base}/notification/stream?token=${encodeURIComponent(token)}`
}

export function deleteNotification(id: number) {
    return postJson<MessageResponse>('/notification/delete', { id }, { authRequired: true })
}