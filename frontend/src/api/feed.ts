import { postJson } from "./client";
import type { ListLatestResponse } from "./types";

export function listLatest(input:{limit:number;latest_time:number}){
    return postJson<ListLatestResponse>('/feed/listLatest',input)
}

import type { ListByFollowingResponse } from './types'

export function listByFollowing(input: { limit: number; latest_time: number }) {
  return postJson<ListByFollowingResponse>('/feed/listByFollowing', input, { authRequired: true })
}