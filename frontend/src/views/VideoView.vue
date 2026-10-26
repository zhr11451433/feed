<script setup lang="ts">
import { onUnmounted, reactive, ref, watch } from 'vue'
import { RouterLink, useRouter } from 'vue-router'

import AppShell from '../components/AppShell.vue'
import { ApiError } from '../api/client'
import * as videoApi from '../api/video'
import type { Video } from '../api/types'
import { useAuthStore } from '../stores/auth'
import { useToastStore } from '../stores/toast'
import SparkMD5 from 'spark-md5'
const router = useRouter()
const auth = useAuthStore()
const toast = useToastStore()

const busy = ref(false)
const stage = ref('')
const published = ref<Video | null>(null)
const videoInput = ref<HTMLInputElement | null>(null)
const coverInput = ref<HTMLInputElement | null>(null)

const publishForm = reactive({
  title: '',
  description: '',
  //as 的作用：手动告诉 TS"按这个类型算"
  //这个值现在是 null，但请把它的类型当成 File | null 来对待
  video: null as File | null,
  cover: null as File | null,
})

const preview = reactive({ videoUrl: '', coverUrl: '' })

const uploadProgress = reactive({
  uploadedBytes: 0,
  totalBytes: 0,
  percent: 0,
})

function resetProgress() {
  uploadProgress.uploadedBytes = 0
  uploadProgress.totalBytes = 0
  uploadProgress.percent = 0
}

async function computeFileMD5(file: File) {
  const hashChunk = 2 << 20
  const spark = new SparkMD5.ArrayBuffer()
  for (let offset = 0; offset < file.size; offset += hashChunk) {
    const end = Math.min(offset + hashChunk, file.size)
    spark.append(await file.slice(offset, end).arrayBuffer())
  }
  return spark.end()
}

async function computeChunkMD5(blob: Blob) {
  const spark = new SparkMD5.ArrayBuffer()
  spark.append(await blob.arrayBuffer())
  return spark.end()
}

const CHUNK_SIZE = 5 << 20
const MAX_CONCURRENT = 3
const MAX_RETRIES = 3

async function uploadVideoChunked(file: File): Promise<videoApi.UploadResponse> {
  const totalChunks = Math.ceil(file.size / CHUNK_SIZE)
  stage.value = '计算文件摘要'
  const fileHash = await computeFileMD5(file)

  stage.value = '初始化上传'
  const initRes = await videoApi.initChunkUpload({
    filename: file.name,
    file_size: file.size,
    chunk_size: CHUNK_SIZE,
    total_chunks: totalChunks,
    file_hash: fileHash,
  })
  const uploadId = initRes.upload_id
  const uploadedSet = new Set(initRes.uploaded_chunks ?? [])

  uploadProgress.totalBytes = file.size
  let already = 0
  for (const index of uploadedSet) {
    already += index === totalChunks - 1 ? file.size - index * CHUNK_SIZE : CHUNK_SIZE
  }
  uploadProgress.uploadedBytes = already
  uploadProgress.percent = file.size > 0 ? Math.round((already / file.size) * 100) : 0

  const pending: number[] = []
  for (let i = 0; i < totalChunks; i++) {
    if (!uploadedSet.has(i)) pending.push(i)
  }
  if (pending.length === 0) {
    stage.value = '合并文件'
    return videoApi.completeChunkUpload(uploadId)
  }

  stage.value = '上传视频'
  const uploadOne = async (chunkIndex: number) => {
    const start = chunkIndex * CHUNK_SIZE
    const end = Math.min(start + CHUNK_SIZE, file.size)
    const blob = file.slice(start, end)
    const chunkHash = await computeChunkMD5(blob)
    let lastErr: unknown
    for (let attempt = 0; attempt < MAX_RETRIES; attempt++) {
      try {
        await videoApi.uploadChunk(uploadId, chunkIndex, chunkHash, blob)
        const n = end - start
        uploadProgress.uploadedBytes += n
        uploadProgress.percent = Math.round((uploadProgress.uploadedBytes / file.size) * 100)
        return
      } catch (e) {
        lastErr = e
      }
    }
    throw lastErr
  }

  await new Promise<void>((resolve, reject) => {
    let active = 0
    let cursor = 0
    let failed = false
    const next = () => {
      if (failed) return
      if (cursor >= pending.length && active === 0) {
        resolve()
        return
      }
      while (active < MAX_CONCURRENT && cursor < pending.length) {
        const chunkIndex = pending[cursor++]
        if (chunkIndex === undefined) continue
        active++
        uploadOne(chunkIndex)
            .then(() => {
              active--
              next()
            })
            .catch((e) => {
              failed = true
              reject(e)
            })
      }
    }
    next()
  })

  stage.value = '合并文件'
  return videoApi.completeChunkUpload(uploadId)
}

function setPreviewVideo(file: File | null) {
  if (preview.videoUrl) URL.revokeObjectURL(preview.videoUrl) // ① 先回收旧的
  preview.videoUrl = file ? URL.createObjectURL(file) : '' // ② 再赋新值
}
function setPreviewCover(file: File | null) {
  if (preview.coverUrl) URL.revokeObjectURL(preview.coverUrl)
  preview.coverUrl = file ? URL.createObjectURL(file) : ''
}

watch(() => publishForm.video, setPreviewVideo)
watch(() => publishForm.cover, setPreviewCover)
onUnmounted(() => {
  setPreviewVideo(null)
  setPreviewCover(null)
})

function pickVideo(e: Event) {
  publishForm.video = (e.target as HTMLInputElement).files?.[0] ?? null
}
function pickCover(e: Event) {
  publishForm.cover = (e.target as HTMLInputElement).files?.[0] ?? null
}
function clearVideo() {
  publishForm.video = null
  if (videoInput.value) videoInput.value.value = ''
}
function clearCover() {
  publishForm.cover = null
  if (coverInput.value) coverInput.value.value = ''
}

async function onPublish() {
    //**防重复提交**：正在发布中直接返回，忽略重复点击。
  if (busy.value) return
  if (!auth.isLoggedIn) {
    toast.error('请先登录')
    await router.push('/account')
    return
  }
  const title = publishForm.title.trim()
  const description = publishForm.description.trim()
  if (!title) {
    toast.error('请输入 title')
    return
  }
  if (!publishForm.video) {
    toast.error('请选择视频文件（.mp4）')
    return
  }
  if (!publishForm.cover) {
    toast.error('请选择封面图片（jpg/png/webp）')
    return
  }

  busy.value = true
  resetProgress()
  published.value = null
  try {
    stage.value = '上传视频'
    const videoRes = await uploadVideoChunked(publishForm.video)
    stage.value = '上传封面'
    const coverRes = await videoApi.uploadCover(publishForm.cover)
    const playUrl = videoRes.play_url || videoRes.url
    const coverUrl = coverRes.cover_url || coverRes.url
    if (!playUrl || !coverUrl) {
      toast.error('上传成功但缺少 url')
      return
    }
    stage.value = '发布视频'
    published.value = await videoApi.publishVideo({
      title,
      description,
      play_url: playUrl,
      cover_url: coverUrl,
    })
    toast.success('已发布')
    publishForm.title = ''
    publishForm.description = ''
    clearVideo()
    clearCover()
  } catch (e) {
    toast.error(e instanceof ApiError ? e.message : String(e))
  } finally {
    busy.value = false
    stage.value = ''
  }
}
</script>

<template>
  <AppShell>
    <div class="publish-wrap">
      <div class="card publish-card">
        <div class="row" style="justify-content: space-between; align-items: baseline">
          <p class="title" style="margin: 0">发布视频</p>
          <div v-if="busy" class="pill">进行中：{{ stage || '…' }}</div>
        </div>
        <p class="subtle" style="margin-top: 10px">选一个 mp4 和一张封面，上传后写入 `/video/publish`。</p>

        <div class="grid form-grid" style="margin-top: 16px">
          <div>
            <label>title</label>
            <input v-model.trim="publishForm.title" class="big-input" :disabled="busy" />
          </div>
          <div>
            <label>description</label>
            <textarea v-model.trim="publishForm.description" class="big-input" :disabled="busy" />
          </div>
          <div class="grid two">
            <div>
              <label>video (.mp4)</label>
              <input ref="videoInput" class="file-native" type="file" accept="video/mp4,.mp4" :disabled="busy" @change="pickVideo" />
              <div class="file-box">
                <button type="button" :disabled="busy" @click="videoInput?.click()">选择视频</button>
                <div class="file-name" :class="publishForm.video ? '' : 'muted'">
                  {{ publishForm.video ? publishForm.video.name : '未选择文件' }}
                </div>
                <button v-if="publishForm.video" type="button" :disabled="busy" @click="clearVideo">清除</button>
              </div>
            </div>
            <div>
              <label>cover (jpg/png/webp)</label>
              <input ref="coverInput" class="file-native" type="file" accept="image/jpeg,image/png,image/webp" :disabled="busy" @change="pickCover" />
              <div class="file-box">
                <button type="button" :disabled="busy" @click="coverInput?.click()">选择封面</button>
                <div class="file-name" :class="publishForm.cover ? '' : 'muted'">
                  {{ publishForm.cover ? publishForm.cover.name : '未选择文件' }}
                </div>
                <button v-if="publishForm.cover" type="button" :disabled="busy" @click="clearCover">清除</button>
              </div>
            </div>
          </div>
          <div v-if="busy && uploadProgress.totalBytes > 0" class="progress-wrap">
            <div class="progress-bar">
              <div class="progress-fill" :style="{ width: uploadProgress.percent + '%' }"></div>
            </div>
            <div class="progress-text">
              {{ (uploadProgress.uploadedBytes / 1024 / 1024).toFixed(1) }} MB /
              {{ (uploadProgress.totalBytes / 1024 / 1024).toFixed(1) }} MB
              ({{ uploadProgress.percent }}%)
            </div>
          </div>
          <div v-if="preview.coverUrl || preview.videoUrl" class="grid two">
            <div v-if="preview.videoUrl" class="preview-card">
              <div class="subtle">视频预览</div>
              <video class="video" :src="preview.videoUrl" controls playsinline />
            </div>
            <div v-if="preview.coverUrl" class="preview-card">
              <div class="subtle">封面预览</div>
              <img class="cover" :src="preview.coverUrl" alt="" />
            </div>
          </div>

          <div class="row" style="justify-content: flex-end">
            <button class="primary big-btn" type="button" :disabled="busy" @click="onPublish">发布</button>
          </div>
        </div>

        <div v-if="published" class="card" style="margin-top: 14px">
          <p class="title">已发布</p>
          <div class="row" style="justify-content: space-between">
            <div>
              <div class="title" style="margin: 0">{{ published.title }}</div>
              <div class="subtle mono">#{{ published.id }}</div>
            </div>
            <RouterLink class="pill" :to="`/video/${published.id}`">去播放</RouterLink>
          </div>
        </div>
      </div>
    </div>
  </AppShell>
</template>

<style scoped>
.publish-wrap { display: grid; justify-items: center; }
.publish-card { width: min(980px, 100%); padding: 22px; }
.form-grid { gap: 16px; }
.file-native {
  position: absolute; width: 1px; height: 1px; padding: 0; margin: -1px;
  overflow: hidden; clip: rect(0, 0, 0, 0); white-space: nowrap; border: 0;
}
.file-box {
  display: flex; align-items: center; gap: 10px; padding: 10px 12px;
  border: 1px solid rgba(255, 255, 255, 0.12); border-radius: 14px;
}
.file-name { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.big-input { width: 100%; padding: 12px 14px; border-radius: 14px; }
.big-btn { padding: 12px 18px; border-radius: 14px; }
.preview-card { border: 1px solid rgba(255, 255, 255, 0.12); border-radius: 16px; padding: 12px; }
.cover, .video { width: 100%; border-radius: 14px; background: rgba(0, 0, 0, 0.35); }
.cover { aspect-ratio: 9/12; object-fit: cover; }
.progress-wrap { display: grid; gap: 6px; }
.progress-bar { height: 8px; background: rgba(255, 255, 255, 0.1); border-radius: 4px; overflow: hidden; }
.progress-fill { height: 100%; background: #4a9eff; border-radius: 4px; transition: width 0.2s ease; }
.progress-text { font-size: 13px; color: rgba(255, 255, 255, 0.7); }
</style>