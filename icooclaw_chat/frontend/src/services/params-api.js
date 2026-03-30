import { request } from "./http";

export async function getExecEnv() {
  return request("/api/v1/params/exec-env/get");
}

export async function setExecEnv(env) {
  return request("/api/v1/params/exec-env/set", {
    method: "POST",
    body: JSON.stringify({ env }),
  });
}
