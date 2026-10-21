import { postJson } from './client'
import type { ProfileResponse,Account,MessageResponse, TokenResponse } from './types'
import { postForm } from './client'
export function register(username: string, password: string) {
  return postJson<MessageResponse>('/account/register', { username, password })
}

export function login(username: string, password: string) {
  return postJson<TokenResponse>('/account/login', { username, password })
}

export function findByID(id: number) {
  return postJson<Account>('/account/findByID', { id })
}

export function logout() {
  return postJson<MessageResponse>('/account/logout', {}, { authRequired: true })
}

export function rename(newUsername: string) {
  return postJson<TokenResponse>('/account/rename', { new_username: newUsername }, { authRequired: true })
}

export function changePassword(username: string, oldPassword: string, newPassword: string) {
  return postJson<MessageResponse>('/account/changePassword', {
    username,
    old_password: oldPassword,
    new_password: newPassword,
  })
}

export function getProfile(accountId: number) {
  return postJson<ProfileResponse>('/account/getProfile', { account_id: accountId })
}

export function uploadAvatar(file: File) {
  const fd = new FormData()
  fd.append('file', file)
  return postForm<{ avatar_url: string }>('/account/uploadAvatar', fd, { authRequired: true })
}

export function updateProfile(data: { avatar_url?: string; bio?: string }) {
  return postJson<MessageResponse>('/account/updateProfile', data, { authRequired: true })
}