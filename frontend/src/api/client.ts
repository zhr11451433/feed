import axios, { type AxiosError, type InternalAxiosRequestConfig } from 'axios'

import { useAuthStore } from '../stores/auth'
import { reportError } from '../utils/error-reporter'

export class ApiError extends Error {
  status: number
  //payload 这个 API 错误额外附带的自定义详情数据
  payload?: unknown
  constructor(message: string, status: number, payload?: unknown) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.payload = payload
  }
}

type AuthConfig = InternalAxiosRequestConfig & {
//   authRequired：调用方声明「这个请求必须有 token」。没 token 就本地直接拒掉，不浪费一次网络往返。
// _retried：标记「这个请求已经因 401 重放过一次了」，防止无限重试。
  authRequired?: boolean
  _retried?: boolean
}
//import.meta.env.VITE_API_BASE Vite 里面读取环境变量
// A ?? B
// 当 A 是 null 或者 undefined 的时候，才返回 B； 
// 如果 A 是空字符串 ""、0、false，不会走 B
const API_BASE = (import.meta.env.VITE_API_BASE as string | undefined) ?? '/api'
const http = axios.create({ baseURL: API_BASE })

// 多个请求一起 401 时，共用这一次刷新
let refreshPromise: Promise<string | null> | null = null

async function tryRefresh(): Promise<string | null> {
  const auth = useAuthStore()
  if (!auth.refreshToken) {
    auth.clearTokens()
    return null
  }
  if (refreshPromise) return refreshPromise
  refreshPromise = (async () => {
    try {
      // 用 axios 本体，不走下面的 http 实例，避免刷新接口的 401 再进拦截器
      const res = await axios.post(`${API_BASE}/account/refresh`, {
        refresh_token: auth.refreshToken,
      })
      const token = res.data?.token as string | undefined
      if (!token) {
        auth.clearTokens()
        return null
      }
      auth.setToken(token)
      return token
    } catch {
      auth.clearTokens()
      return null
    } finally {
      refreshPromise = null
    }
  })()
  return refreshPromise
}

async function goLogin() {
  // 动态 import，避免 client → router → 页面 → client 的循环依赖
  const { default: router } = await import('../router')
  const name = router.currentRoute.value.name
  if (name === 'account' || name === 'account-register') return
  await router.push('/account')
}

http.interceptors.request.use((config: AuthConfig) => {
  const auth = useAuthStore()
  if (config.authRequired && !auth.token) {
    return Promise.reject(new ApiError('需要先登录（缺少 token）', 401))
  }
  if (auth.token) config.headers.set('Authorization', `Bearer ${auth.token}`)
  return config
})

http.interceptors.response.use(
  (res) => res,
  async (error: AxiosError) => {
    const config = error.config as AuthConfig | undefined
    const status = error.response?.status
    const path = config?.url ?? ''
    const isCredentialCall =
      path.includes('/account/login') || path.includes('/account/register') || path.includes('/account/refresh')

    if (status === 401 && config && !config._retried && !isCredentialCall) {
      const newToken = await tryRefresh()
      if (newToken) {
        config._retried = true
        config.headers.set('Authorization', `Bearer ${newToken}`)
        return http.request(config)
      }
      await goLogin()
    } else if (status === 401 && !isCredentialCall) {
      useAuthStore().clearTokens()
      await goLogin()
    }

    const apiErr = toApiError(error)
    reportError(apiErr, { path, status })
    return Promise.reject(apiErr)
  },
)

function toApiError(error: unknown): ApiError {
  if (error instanceof ApiError) return error
  if (axios.isAxiosError(error)) {
    const status = error.response?.status ?? 0
    const data = error.response?.data
    const msg =
      data && typeof data === 'object' && 'error' in data && (data as { error?: unknown }).error
        ? String((data as { error: unknown }).error)
        : `请求失败 (${status || 'network'})`
    return new ApiError(msg, status, data)
  }
  return new ApiError(error instanceof Error ? error.message : String(error), 0)
}

//postJson 和 postForm 都是用来向后端发送 HTTP POST 请求的封装函数

export function postJson<T>(path: string, body: unknown, options?: { authRequired?: boolean }): Promise<T> {
  return http.post<T>(path, body ?? {}, { authRequired: options?.authRequired } as AuthConfig).then((r) => r.data)
}

// 上传时不要手写 Content-Type，否则会丢掉 boundary。这轮页面还用不到。
export function postForm<T>(path: string, body: FormData, options?: { authRequired?: boolean }): Promise<T> {
  return http.post<T>(path, body, { authRequired: options?.authRequired } as AuthConfig).then((r) => r.data)
}