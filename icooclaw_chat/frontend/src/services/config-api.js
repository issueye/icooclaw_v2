import { request } from "./http";

export async function getConfig() {
  return request("/api/v1/config/");
}

export async function updateConfig(config) {
  return request("/api/v1/config/update", {
    method: "POST",
    body: JSON.stringify({ config }),
  });
}

export async function overwriteConfig(content) {
  return request("/api/v1/config/overwrite", {
    method: "POST",
    body: JSON.stringify({ content }),
  });
}

export async function getConfigFile() {
  return request("/api/v1/config/file");
}

export async function getConfigJSON() {
  return request("/api/v1/config/json");
}
