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

export async function listByAuthorId(authorId: number) {
  const videos = await postJson<Video[] | null>('/video/listByAuthorID', { author_id: authorId })
  return videos ?? []
}

export type InitChunkUploadResponse = {
    upload_id: string
    uploaded_chunks: number[]
}

export function initChunkUpload(input: {
    filename: string
    file_size: number
    chunk_size: number
    total_chunks: number
    file_hash: string
}) {
    return postJson<InitChunkUploadResponse>('/video/chunk/init', input, { authRequired: true })
}

export function uploadChunk(uploadId: string, chunkIndex: number, chunkHash: string, blob: Blob) {
    const fd = new FormData()
    fd.append('upload_id', uploadId)
    fd.append('chunk_index', String(chunkIndex))
    fd.append('chunk_hash', chunkHash)
    fd.append('file', blob, 'chunk')
    return postForm<{ chunk_index: number }>('/video/chunk/upload', fd, { authRequired: true })
}

export function completeChunkUpload(uploadId: string) {
    return postJson<UploadResponse>('/video/chunk/complete', { upload_id: uploadId }, { authRequired: true })
}