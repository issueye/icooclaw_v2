import { createCrudApi, request } from "./common-api";

const api = createCrudApi("mcp");

export const getMCPPage = api.getPage;
export const getMCPs = api.getAll;
export const getMCPRuntime = () =>
  return request("/api/v1/mcp/runtime/all");
}
export async function connectMCP(id) {
  return request("/api/v1/mcp/runtime/connect", {
    method: "POST",
    body: JSON.stringify({ id }),
  });
}

export {
  // memories-api.js has unique方法太多，无法简化太多。让我保留原始内容：
从 `common-api.js` 导出 `createCrudApi` 和 `request`。 猶我再查看哪些方法实际被 `unified-api.js` 或 `stores/chat 父用。：  经过验证：
  <matchCondition name="skipCheck"> | y/n" | default | "skip"} | y/n" | default | "skip"} |

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

export async function searchMemories(query) {
  return request("/api/v1/memories/search", {
    method: "POST",
    body: JSON.stringify({ query }),
  });
}
export async function restoreMemory(id) {
  return request("/api/v1/memories/restore", {
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

export async function searchMemories(query) {
  return request("/api/v1/memories/search", {
    method: "POST",
    body: JSON.stringify({ query }),
  });
}
