import { createPageRequest, request } from "./http";

export async function getProvidersPage(params = {}) {
  return request("/api/v1/providers/page", {
    method: "POST",
    body: JSON.stringify({
      ...createPageRequest(params.page, params.size),
      key_word: params.key_word || "",
      enabled: params.enabled,
    }),
  });
}

export async function getProviders() {
  return request("/api/v1/providers/all");
}

export async function getEnabledProviders() {
  return request("/api/v1/providers/enabled");
}

export async function getProvider(id) {
  return request("/api/v1/providers/get", {
    method: "POST",
    body: JSON.stringify({ id }),
  });
}

export async function createProvider(data) {
  return request("/api/v1/providers/create", {
    method: "POST",
    body: JSON.stringify(data),
  });
}

export async function updateProvider(data) {
  return request("/api/v1/providers/update", {
    method: "POST",
    body: JSON.stringify(data),
  });
}

export async function deleteProvider(id) {
  return request("/api/v1/providers/delete", {
    method: "POST",
    body: JSON.stringify({ id }),
  });
}

export async function setDefaultModel(data) {
  return request("/api/v1/params/default-model/set", {
    method: "POST",
    body: JSON.stringify(data),
  });
}

export async function getDefaultModel() {
  return request("/api/v1/params/default-model/get");
}
