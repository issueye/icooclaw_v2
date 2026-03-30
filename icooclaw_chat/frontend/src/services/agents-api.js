import { createPageRequest, request } from "./http";

export async function getAgentsPage(params = {}) {
  return request("/api/v1/agents/page", {
    method: "POST",
    body: JSON.stringify({
      ...createPageRequest(params.page, params.size),
      key_word: params.key_word || "",
      type: params.type || "",
      enabled: params.enabled,
    }),
  });
}

export async function getAgents() {
  return request("/api/v1/agents/all");
}

export async function getEnabledAgents() {
  return request("/api/v1/agents/enabled");
}

export async function getAgent(id) {
  return request("/api/v1/agents/get", {
    method: "POST",
    body: JSON.stringify({ id }),
  });
}

export async function createAgent(data) {
  return request("/api/v1/agents/create", {
    method: "POST",
    body: JSON.stringify(data),
  });
}

export async function updateAgent(data) {
  return request("/api/v1/agents/update", {
    method: "POST",
    body: JSON.stringify(data),
  });
}

export async function deleteAgent(id) {
  return request("/api/v1/agents/delete", {
    method: "POST",
    body: JSON.stringify({ id }),
  });
}

export async function getDefaultAgent() {
  return request("/api/v1/params/default-agent/get");
}

export async function setDefaultAgent(agent_id) {
  return request("/api/v1/params/default-agent/set", {
    method: "POST",
    body: JSON.stringify({ agent_id }),
  });
}
