import { createPageRequest, request } from "./http";

export async function getTasksPage(params = {}) {
  return request("/api/v1/tasks/page", {
    method: "POST",
    body: JSON.stringify({
      ...createPageRequest(params.page, params.size),
      key_word: params.key_word || "",
      enabled: params.enabled,
    }),
  });
}

export async function getTasks() {
  return request("/api/v1/tasks/all");
}

export async function getEnabledTasks() {
  return request("/api/v1/tasks/enabled");
}

export async function getTask(id) {
  return request("/api/v1/tasks/get", {
    method: "POST",
    body: JSON.stringify({ id }),
  });
}

export async function createTask(data) {
  return request("/api/v1/tasks/create", {
    method: "POST",
    body: JSON.stringify(data),
  });
}

export async function updateTask(data) {
  return request("/api/v1/tasks/update", {
    method: "POST",
    body: JSON.stringify(data),
  });
}

export async function deleteTask(id) {
  return request("/api/v1/tasks/delete", {
    method: "POST",
    body: JSON.stringify({ id }),
  });
}

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
