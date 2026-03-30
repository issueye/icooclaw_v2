import { createPageRequest, request } from "./http";

export { createPageRequest, request };

export async function checkHealth() {
  return request("/api/v1/health");
}

export function createCrudApi(resourceName) {
  const base = `/api/v1/${resourceName}`;
  return {
    getPage: (params = {}) =>
      request(`${base}/page`, {
        method: "POST",
        body: JSON.stringify({
          ...createPageRequest(params.page, params.size),
          key_word: params.key_word || "",
          ...(params.extraFilters || {}),
        }),
      }),
    getAll: () => request(`${base}/all`),
    getEnabled: () => request(`${base}/enabled`),
    getById: (id) =>
      request(`${base}/get`, { method: "POST", body: JSON.stringify({ id }) }),
    create: (data) =>
      request(`${base}/create`, { method: "POST", body: JSON.stringify(data) }),
    update: (data) =>
      request(`${base}/update`, { method: "POST", body: JSON.stringify(data) }),
    remove: (id) =>
      request(`${base}/delete`, { method: "POST", body: JSON.stringify({ id }) }),
  };
}
