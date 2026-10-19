import { postJson } from './client'
import type { IsLikedResponse, MessageResponse } from './types'

export function like(videoId: number) {
  return postJson<MessageResponse>('/like/like', { video_id: videoId }, { authRequired: true })
}

export function unlike(videoId: number) {
  return postJson<MessageResponse>('/like/unlike', { video_id: videoId }, { authRequired: true })
}

export function isLiked(videoId: number) {
    //<IsLikedResponse> 期望后端返回的数据类型
  return postJson<IsLikedResponse>('/like/isLiked', { video_id: videoId }, { authRequired: true })
}