import { reactive } from 'vue'
import { ApiError } from '../api/client'
import * as feedApi from '../api/feed'
import type { FeedVideoItem } from '../api/types'

export function useVideoFeed(){
    const state = reactive({
        items: [] as FeedVideoItem[],
        loading: false,
        error: '',
        hasMore: false,
        nextTime: 0,
    })
    async function load(reset:boolean) {
        if (state.loading) return
        if (!reset && !state.hasMore) return
        state.loading = true
        state.error = ''
        try{
            const res = await feedApi.listLatest({
                limit:10,
                latest_time:reset ? 0 :state.nextTime,
            })
            const list = res.video_list ?? []
        state.nextTime = res.next_time
        // 空页当成没有更多，避免哨兵一直露着时无限请求
        state.hasMore = Boolean(res.has_more) && list.length > 0
        state.items = reset ? list : state.items.concat(list)
        }catch(e){
            state.error = e instanceof ApiError ? e.message : String(e)
        }finally{
            state.loading = false
        }
    }
    return {state,load}
}