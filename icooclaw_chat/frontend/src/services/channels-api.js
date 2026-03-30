import { createCrudApi, request, request } from "./common-api";
import { createPageRequest, request } from "./http";

import { requestBlob } from "./http";

const api = createCrudApi("mcp");

export const getMCPPage = api.getPage;
export const getMCPs = api.getAll;
export const getMCPRuntime = () {
  return request("/api/v1/mcp/runtime/all");
}
export async function getMCPRuntime() {
  return request("/api/v1/mcp/runtime/connect", {
    method: "POST",
    body: JSON.stringify({ id }),
  });
}
export async function connectMCP(id) {
  return request("/api/v1/mcp/runtime/connect", {
    method: "POST",
    body: JSON.stringify({ id }),
  });
}
export async function deleteMCP(id) {
  return request("/api/v1/mcp/delete", {
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
