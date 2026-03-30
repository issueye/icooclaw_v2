import { createPageRequest, request } from "./http";

export async function getMemoriesPage(params = {}) {
  return request("/api/v1/memories/page", {
    method: "POST",
    body: JSON.stringify({
      ...createPageRequest(params.page, params.size),
      type: params.type || "",
      key_word: params.key_word || "",
      user_id: params.user_id || "",
      session_id: params.session_id,
    }),
  });
}

export async function createMemory(data) {
  return request("/api/v1/memories/create", {
    method: "POST",
    body: JSON.stringify(data),
  });
}

export async function updateMemory(data) {
  return request("/api/v1/memories/update", {
    method: "POST",
    body: JSON.stringify(data),
  });
}

export async function deleteMemory(id) {
  return request("/api/v1/memories/delete", {
    method: "POST",
    body: JSON.stringify({ id }),
  });
}

export async function getMemory(id) {
  return request("/api/v1/memories/get", {
    method: "POST",
    body: JSON.stringify({ id }),
  });
}

export async function pinMemory(id) {
  return request("/api/v1/memories/pin", {
    method: "POST",
    body: JSON.stringify({ id }),
  });
}

export async function unpinMemory(id) {
  return request("/api/v1/memories/unpin", {
    method: "POST",
    body: JSON.stringify({ id }),
  });
}

export async function softDeleteMemory(id) {
  return request("/api/v1/memories/soft-delete", {
    method: "POST",
    body: JSON.stringify({ id }),
  });
}

export async function restoreMemory(id) {
  return request("/api/v1/memories/restore", {
    method: "POST",
    body: JSON.stringify({ id }),
  });
}

export async function searchMemories(query) {
  return request("/api/v1/memories/search", {
    method: "POST",
    body: JSON.stringify({ query }),
  });
}
