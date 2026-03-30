import { createPageRequest, request } from "./http";

export async function getChannelsPage(params = {}) {
  return request("/api/v1/channels/page", {
    method: "POST",
    body: JSON.stringify({
      ...createPageRequest(params.page, params.size),
      key_word: params.key_word || "",
      enabled: params.enabled,
    }),
  });
}

export async function getChannels() {
  return request("/api/v1/channels/all");
}

export async function getEnabledChannels() {
  return request("/api/v1/channels/enabled");
}

export async function getChannel(id) {
  return request("/api/v1/channels/get", {
    method: "POST",
    body: JSON.stringify({ id }),
  });
}

export async function createChannel(data) {
  return request("/api/v1/channels/create", {
    method: "POST",
    body: JSON.stringify(data),
  });
}

export async function updateChannel(data) {
  return request("/api/v1/channels/update", {
    method: "POST",
    body: JSON.stringify(data),
  });
}

export async function deleteChannel(id) {
  return request("/api/v1/channels/delete", {
    method: "POST",
    body: JSON.stringify({ id }),
  });
}
