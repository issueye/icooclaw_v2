import { createPageRequest, request } from "./http";

export async function getSessionsPage(params = {}) {
  return request("/api/v1/sessions/page", {
    method: "POST",
    body: JSON.stringify({
      ...createPageRequest(params.page, params.size),
      key_word: params.key_word || "",
      channel: params.channel || "",
      user_id: params.user_id || "",
    }),
  });
}

export async function getSessions(params = {}) {
  const response = await getSessionsPage({ page: 1, size: 100, ...params });
  const data = response.data || response;
  return data?.records || [];
}

export async function getSession(id) {
  return request("/api/v1/sessions/get", {
    method: "POST",
    body: JSON.stringify({ id }),
  });
}

export async function createSession(data) {
  return request("/api/v1/sessions/create", {
    method: "POST",
    body: JSON.stringify(data),
  });
}

export async function updateSession(data) {
  return request("/api/v1/sessions/save", {
    method: "POST",
    body: JSON.stringify(data),
  });
}

export async function deleteSession(id) {
  return request("/api/v1/sessions/delete", {
    method: "POST",
    body: JSON.stringify({ id }),
  });
}
