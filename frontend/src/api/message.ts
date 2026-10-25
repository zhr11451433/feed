import {postJson} from "@/api/client.ts";
import type {DirectMessage,ListMessagesResponse} from "@/api/types.ts";

export function sendMessage(toId:number,content:string){
    return postJson<DirectMessage>('/message/send',{to_id:toId,content:content},{authRequired:true});
}

export function listMessages(peerId: number) {
    return postJson<ListMessagesResponse>('/message/list', { peer_id: peerId }, { authRequired: true })
}