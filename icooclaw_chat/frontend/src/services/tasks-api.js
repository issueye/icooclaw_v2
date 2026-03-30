import { createCrudApi, request } from "./common-api";

const api = createCrudApi("tasks");

export const getTasksPage = api.getPage;
export const getTasks = api.getAll;
export const getEnabledTasks = api.getEnabled;
export const getTask = api.getById;
export const createTask = api.create;
export const updateTask = api.update;
export const deleteTask = api.remove;

export async function toggleTask(id) {
  return request("/api/v1/tasks/toggle", {
    method: "POST",
    body: JSON.stringify({ id }),
  });
}

export async function executeTask(id) {
  return request("/api/v1/tasks/execute", {
    method: "POST",
    body: JSON.stringify({ id }),
  });
}
