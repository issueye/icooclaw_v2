import { createPageRequest, request } from "./http";

export async function getMCPPage(params = {}) {
  return request("/api/v1/mcp/page", {
    method: "POST",
    body: JSON.stringify({
      ...createPageRequest(params.page, params.size),
      key_word: params.key_word || "",
    }),
  });
}

export async function getMCPs() {
  return request("/api/v1/mcp/all");
}

export async function getMCPRuntime() {
  return request("/api/v1/mcp/runtime/all");
}

export async function connectMCP(id) {
  return request("/api/v1/mcp/runtime/connect", {
    method: "POST",
    body: JSON.stringify({ id }),
  });
}

export async function getMCP(id) {
  return request("/api/v1/mcp/get", {
    method: "POST",
    body: JSON.stringify({ id }),
  });
}

export async function createMCP(data) {
  return request("/api/v1/mcp/create", {
    method: "POST",
    body: JSON.stringify(data),
  });
}

export async function updateMCP(data) {
  return request("/api/v1/mcp/update", {
    method: "POST",
    body: JSON.stringify(data),
  });
}

export async function deleteMCP(id) {
  return request("/api/v1/mcp/delete", {
    method: "POST",
    body: JSON.stringify({ id }),
  });
}
