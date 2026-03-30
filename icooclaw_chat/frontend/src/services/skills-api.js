import { createPageRequest, request, requestBlob } from "./http";

export async function getSkillsPage(params = {}) {
  return request("/api/v1/skills/page", {
    method: "POST",
    body: JSON.stringify({
      ...createPageRequest(params.page, params.size),
      key_word: params.key_word || "",
      enabled: params.enabled,
      source: params.source || "",
    }),
  });
}

export async function getSkills() {
  return request("/api/v1/skills/all");
}

export async function getEnabledSkills() {
  return request("/api/v1/skills/enabled");
}

export async function getSkill(id) {
  return request("/api/v1/skills/get", {
    method: "POST",
    body: JSON.stringify({ id }),
  });
}

export async function getSkillByName(name) {
  return request("/api/v1/skills/get-by-name", {
    method: "POST",
    body: JSON.stringify({ name }),
  });
}

export async function createSkill(data) {
  return request("/api/v1/skills/create", {
    method: "POST",
    body: JSON.stringify(data),
  });
}

export async function updateSkill(data) {
  return request("/api/v1/skills/update", {
    method: "POST",
    body: JSON.stringify(data),
  });
}

export async function upsertSkill(data) {
  return request("/api/v1/skills/upsert", {
    method: "POST",
    body: JSON.stringify(data),
  });
}

export async function deleteSkill(id) {
  return request("/api/v1/skills/delete", {
    method: "POST",
    body: JSON.stringify({ id }),
  });
}

export async function batchDeleteSkills(ids) {
  return request("/api/v1/skills/batch-delete", {
    method: "POST",
    body: JSON.stringify({ ids }),
  });
}

export async function batchUpdateSkillsEnabled(ids, enabled) {
  return request("/api/v1/skills/batch-enabled", {
    method: "POST",
    body: JSON.stringify({ ids, enabled }),
  });
}

export async function batchUpdateSkillsAlwaysLoad(ids, alwaysLoad) {
  return request("/api/v1/skills/batch-always-load", {
    method: "POST",
    body: JSON.stringify({ ids, always_load: alwaysLoad }),
  });
}

export async function getSkillTags() {
  return request("/api/v1/skills/tags");
}

export async function getSkillsByTag(tag) {
  return request(`/api/v1/skills/by-tag?tag=${encodeURIComponent(tag)}`);
}

export async function exportSkills() {
  return requestBlob("/api/v1/skills/export", {
    method: "GET",
  });
}

export async function importSkills(file, overwrite = false) {
  const form = new FormData();
  form.append("file", file);
  form.append("overwrite", overwrite ? "true" : "false");
  return request("/api/v1/skills/import", {
    method: "POST",
    body: form,
  });
}

export async function installSkill(slug, version = "") {
  return request("/api/v1/skills/install", {
    method: "POST",
    body: JSON.stringify({ slug, version }),
  });
}
