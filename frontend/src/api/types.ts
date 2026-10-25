export type MessageResponse = { message: string }

export type TokenResponse = {
  token: string
  refresh_token?: string
  account_id?: number
  username?: string
}

export type Account = {
  id: number
  username: string
  avatar_url?: string
  bio?: string
}

export type Video = {
  id: number
  author_id: number
  username: string
  title: string
  description?: string
  play_url: string
  cover_url: string
  create_time: string
  likes_count: number
  popularity: number
}

export type Comment = {
  id: number
  username: string
  video_id: number
  author_id: number
  content: string
  created_at: string
}

export type FeedAuthor = {
  id: number
  username: string
}

export type FeedVideoItem = {
  id: number
  author: FeedAuthor
  title: string
  description?: string
  play_url: string
  cover_url: string
  create_time: number
  likes_count: number
  is_liked: boolean
}

export type ListLatestResponse = {
  video_list: FeedVideoItem[]
  next_time: number
  has_more: boolean
}

export type IsLikedResponse = { is_liked: boolean }

export type GetAllVloggersResponse = {
  vloggers: Account[]
  vlogger_count: number
}

export type GetAllFollowersResponse = {
  followers: Account[]
  follower_count: number
}

// next_time 是秒。ListLatestResponse 的 next_time 是毫秒。
export type ListByFollowingResponse = {
  video_list: FeedVideoItem[]
  next_time: number
  has_more: boolean
}

export type ProfileResponse = {
  account: Account
  video_count: number
  total_likes: number
  follower_count: number
  vlogger_count: number
}

export type DirectMessage = {
  id: number
  from_id: number
  to_id: number
  content: string
  is_read: boolean
  created_at: string
}

export type ListMessagesResponse = {
  messages: DirectMessage[]
}

export type AppNotification = {
  id: number
  recipient_id: number
  sender_id: number
  type: string
  target_id: number
  content: string
  is_read: boolean
  created_at: string
}

export type ListNotificationsResponse = { notifications: AppNotification[] }
export type UnreadCountResponse = { count: number }

export type ListByPopularityResponse = {
  video_list: FeedVideoItem[]
  as_of: number
  next_offset: number
  has_more: boolean
}