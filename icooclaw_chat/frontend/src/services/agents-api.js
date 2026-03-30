import { createCrudApi, createPageRequest, request } from "./common-api";

const api = createCrudApi("agents");

export const getAgentsPage = api.getPage;
export const getAgents = api.getAll;
export const getEnabledAgents = api.getEnabled;
export const getAgent = api.getById;
export const createAgent = api.create;
export const updateAgent = api.update;
export const deleteAgent = api.remove;

export async function getDefaultAgent() {
  return request("/api/v1/params/default-agent/get");
}

export async function setDefaultAgent(agent_id) {
  return request("/api/v1/params/default-agent/set", {
    method: "POST",
    body: JSON.stringify({ agent_id }),
  });
}
