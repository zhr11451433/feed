import type {Video} from "./types"
import { postForm, postJson } from './client'
export type UploadResponse = {
    url:string
    play_url?:string
    cover_url?:string
}

export function uploadVideo(file: File) {
  const fd = new FormData()
  fd.append('file', file)
  //`<T>` 代表「本次请求成功后，后端返回的数据类型」
  return postForm<UploadResponse>('/video/uploadVideo', fd, { authRequired: true })
}

export function uploadCover(file: File) {
  const fd = new FormData()
  fd.append('file', file)
  return postForm<UploadResponse>('/video/uploadCover', fd, { authRequired: true })
}

export function publishVideo(input: { title: string; description: string; play_url: string; cover_url: string }) {
  return postJson<Video>('/video/publish', input, { authRequired: true })
}
export function getDetail(id: number) {
  return postJson<Video>('/video/getDetail', { id })
}