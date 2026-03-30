import { createCrudApi, request } from "./common-api";

const api = createCrudApi("providers");

export const getProvidersPage = api.getPage;
export const getProviders = api.getAll;
export const getEnabledProviders = api.getEnabled;
export const getProvider = api.getById;
export const createProvider = api.create;
export const updateProvider = api.update;
export const deleteProvider = api.remove;

export async function setDefaultModel(data) {
  return request("/api/v1/params/default-model/set", {
    method: "POST",
    body: JSON.stringify(data),
  });
}

export async function getDefaultModel() {
  return request("/api/v1/params/default-model/get");
}
