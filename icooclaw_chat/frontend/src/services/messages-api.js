import { createPageRequest, request } from "./http";

export async function getMessagesPage(params = {}) {
  return request("/api/v1/messages/page", {
    method: "POST",
    body: JSON.stringify({
      ...createPageRequest(params.page, params.size),
      session_id: params.session_id,
      role: params.role || "",
    }),
  });
}

export async function getSessionMessages(sessionId, params = {}) {
  const response = await getMessagesPage({
    session_id: sessionId,
    page: 1,
    size: 100,
    ...params,
  });
  const data = response.data || response;
  return data?.records || [];
}

export async function createMessage(data) {
  return request("/api/v1/messages/create", {
    method: "POST",
    body: JSON.stringify(data),
  });
}

export async function updateMessage(data) {
  return request("/api/v1/messages/update", {
    method: "POST",
    body: JSON.stringify(data),
  });
}

export async function deleteMessage(id) {
  return request("/api/v1/messages/delete", {
    method: "POST",
    body: JSON.stringify({ id }),
  });
}

export async function getMessage(id) {
  return request("/api/v1/messages/get", {
    method: "POST",
    body: JSON.stringify({ id }),
  });
}
