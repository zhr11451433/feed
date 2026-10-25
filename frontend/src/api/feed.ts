import { postJson } from "./client";
import type { ListLatestResponse,ListByPopularityResponse } from "./types";

export function listLatest(input:{limit:number;latest_time:number}){
    return postJson<ListLatestResponse>('/feed/listLatest',input)
}

import type { ListByFollowingResponse } from './types'

export function listByFollowing(input: { limit: number; latest_time: number }) {
  return postJson<ListByFollowingResponse>('/feed/listByFollowing', input, { authRequired: true })
}

export function listByPopularity(input: { limit: number; as_of: number; offset: number }) {
    return postJson<ListByPopularityResponse>('/feed/listByPopularity', input)
}