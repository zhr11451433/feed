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