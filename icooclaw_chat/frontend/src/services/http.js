const API_BASE_KEY = "icooclaw_api_base";

export function isWailsEnv() {
  return typeof window !== "undefined" && window.go !== undefined;
}

function getApiBase() {
  const stored = localStorage.getItem(API_BASE_KEY);
  return stored || "http://localhost:16789";
}

function setApiBase(base) {
  localStorage.setItem(API_BASE_KEY, base);
}

function getHeaders() {
  return {
    "Content-Type": "application/json",
  };
}

function getProxyUrl(endpoint) {
  return `/proxy${endpoint}`;
}

export function buildApiUrl(endpoint) {
  if (isWailsEnv()) {
    return getProxyUrl(endpoint);
  }
  return `${getApiBase()}${endpoint}`;
}

export async function request(endpoint, options = {}) {
  const isFormData = typeof FormData !== "undefined" && options.body instanceof FormData;
  const config = {
    ...options,
    headers: {
      ...(isFormData ? {} : getHeaders()),
      ...options.headers,
    },
  };

  try {
    const response = await fetch(buildApiUrl(endpoint), config);
    if (!response.ok) {
      throw new Error(`HTTP ${response.status}: ${response.statusText}`);
    }
    return await response.json();
  } catch (error) {
    console.error(`API request failed: ${endpoint}`, error);
    throw error;
  }
}

export async function requestBlob(endpoint, options = {}) {
  const response = await fetch(buildApiUrl(endpoint), {
    ...options,
    headers: {
      ...getHeaders(),
      ...options.headers,
    },
  });

  if (!response.ok) {
    throw new Error(`HTTP ${response.status}: ${response.statusText}`);
  }

  return response.blob();
}

export async function requestSSE(endpoint, payload) {
  const response = await fetch(buildApiUrl(endpoint), {
    method: "POST",
    headers: getHeaders(),
    body: JSON.stringify(payload),
  });

  if (!response.ok) {
    throw new Error(`HTTP ${response.status}: ${response.statusText}`);
  }

  if (!response.body) {
    throw new Error("No response body");
  }

  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  let buffer = "";

  async function* streamReader() {
    while (true) {
      const { done, value } = await reader.read();
      if (done) break;

      buffer += decoder.decode(value, { stream: true });

      const lines = buffer.split("\n");
      buffer = lines.pop() || "";

      for (const line of lines) {
        if (line.startsWith("data: ")) {
          const data = line.slice(6).trim();
          if (data === "[DONE]") {
            return;
          }
          yield data;
        }
      }
    }
  }

  return streamReader();
}

export function createPageRequest(page = 1, size = 10) {
  return { page: { page, size } };
}

export function getApiBaseUrl() {
  return getApiBase();
}

export function setApiBaseUrl(base) {
  setApiBase(base);
}
