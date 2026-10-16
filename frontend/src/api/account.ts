import { postJson } from './client'
import type { MessageResponse, TokenResponse } from './types'

export function register(username: string, password: string) {
  return postJson<MessageResponse>('/account/register', { username, password })
}

export function login(username: string, password: string) {
  return postJson<TokenResponse>('/account/login', { username, password })
}