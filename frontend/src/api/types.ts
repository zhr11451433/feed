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