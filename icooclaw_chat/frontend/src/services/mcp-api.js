import { createCrudApi, request } from "./common-api";

const api = createCrudApi("mcp");

export const getMCPPage = api.getPage;
export const getMCPs = api.getAll;
export const getMCP = api.getById;
export const createMCP = api.create;
export const updateMCP = api.update;
export const deleteMCP = api.remove;

export async function getMCPRuntime() {
  return request("/api/v1/mcp/runtime/all");
}

export async function connectMCP(id) {
  return request("/api/v1/mcp/runtime/connect", {
    method: "POST",
    body: JSON.stringify({ id }),
  });
}
