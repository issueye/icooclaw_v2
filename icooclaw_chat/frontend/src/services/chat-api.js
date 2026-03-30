import { request, requestSSE } from "./http";

export async function sendChatMessage(content, chatId, userId) {
  return request("/api/v1/chat", {
    method: "POST",
    body: JSON.stringify({
      content,
      chat_id: chatId,
      user_id: userId,
    }),
  });
}

export async function sendChatStream(content, chatId, userId) {
  return requestSSE("/api/v1/chat/stream", {
    content,
    chat_id: chatId,
    user_id: userId,
  });
}
