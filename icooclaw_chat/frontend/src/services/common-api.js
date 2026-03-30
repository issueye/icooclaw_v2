import { request } from "./http";

export async function checkHealth() {
  return request("/api/v1/health");
}
