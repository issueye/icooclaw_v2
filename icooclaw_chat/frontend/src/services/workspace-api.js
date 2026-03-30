import { request } from "./http";

export async function getWorkspacePrompt(name) {
  return request("/api/v1/workspace/prompt/get", {
    method: "POST",
    body: JSON.stringify({ name }),
  });
}

export async function saveWorkspacePrompt(name, content) {
  return request("/api/v1/workspace/prompt/save", {
    method: "POST",
    body: JSON.stringify({ name, content }),
  });
}

export async function generateWorkspacePrompt({ name, instruction, current = "", mode = "modify" }) {
  return request("/api/v1/workspace/prompt/generate", {
    method: "POST",
    body: JSON.stringify({ name, instruction, current, mode }),
  });
}
