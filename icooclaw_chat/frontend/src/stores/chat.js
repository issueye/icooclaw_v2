import { defineStore } from "pinia";
import { ref, computed } from "vue";
import api from "../services/api";

function generateId() {
  return Date.now().toString(36) + Math.random().toString(36).slice(2, 8);
}

function buildTraceItems(thinking = "", toolCalls = []) {
  const traceItems = [];
  if (thinking) {
    traceItems.push({
      id: generateId(),
      type: "thinking",
      content: thinking,
    });
  }
  for (const toolCall of toolCalls || []) {
    traceItems.push({
      id: `tool-${toolCall.id || generateId()}`,
      type: "tool_call",
      toolCallId: toolCall.id,
    });
  }
  return traceItems;
}

function parseMessageMetadata(metadata) {
  if (!metadata) {
    return {
      reasoningContent: "",
      toolCalls: [],
      traceItems: [],
    };
  }

  try {
    const parsed = typeof metadata === "string" ? JSON.parse(metadata) : metadata;
    const traceItems = Array.isArray(parsed?.trace_items)
      ? parsed.trace_items.map((item, index) => ({
          id: item.tool_call_id ? `tool-${item.tool_call_id}` : `trace-${index}`,
          type: item.type,
          content: item.content || "",
          toolCallId: item.tool_call_id || "",
          toolName: item.tool_name || "",
          toolArgs: item.tool_args || "",
          toolResult: item.tool_result || "",
          iteration: item.iteration || 0,
        }))
      : [];

    const toolCalls = traceItems
      .filter((item) => item.type === "tool_call")
      .map((item) => ({
        id: item.toolCallId || item.id,
        toolName: item.toolName || "unknown_tool",
        arguments: item.toolArgs || "",
        status: item.toolResult ? "completed" : "running",
        content: item.toolResult || "",
        error: null,
        timestamp: Date.now(),
      }));

    return {
      reasoningContent: parsed?.reasoning_content || "",
      toolCalls,
      traceItems,
    };
  } catch {
    return {
      reasoningContent: "",
      toolCalls: [],
      traceItems: [],
    };
  }
}

function hasVisibleTrace(message) {
  return Boolean(
    (message.thinking && String(message.thinking).trim()) ||
      (Array.isArray(message.toolCalls) && message.toolCalls.length > 0) ||
      (Array.isArray(message.traceItems) && message.traceItems.length > 0),
  );
}

function normalizeLoadedMessage(message) {
  const normalized = {
    ...message,
    content: typeof message.content === "string" ? message.content : "",
    thinking: typeof message.thinking === "string" ? message.thinking : "",
    toolCalls: Array.isArray(message.toolCalls) ? message.toolCalls : [],
    traceItems: Array.isArray(message.traceItems) ? message.traceItems : [],
  };

  const hasContent = Boolean(normalized.content.trim());
  if (hasContent || hasVisibleTrace(normalized)) {
    return normalized;
  }

  return null;
}

function getDefaultWsHost() {
  return localStorage.getItem("icooclaw_ws_host") || "localhost";
}

function getDefaultWsPort() {
  return localStorage.getItem("icooclaw_ws_port") || "16789";
}

function getDefaultWsPath() {
  return localStorage.getItem("icooclaw_ws_path") || "/ws";
}

export const useChatStore = defineStore("chat", () => {
  const sessions = ref([]);
  const currentSessionId = ref(null);
  const isLoading = ref(false);
  const isLoadingSessions = ref(false);
  const apiBase = ref(api.getApiBaseUrl());
  const wsHost = ref(getDefaultWsHost());
  const wsPort = ref(getDefaultWsPort());
  const wsPath = ref(getDefaultWsPath());
  const userId = ref(localStorage.getItem("icooclaw_user_id") || "user-1");
  
  const wsUrl = computed(() => `ws://${wsHost.value}:${wsPort.value}${wsPath.value}`);
  const wsConnected = ref(false);
  const apiHealth = ref("checking");
  const wsRuntimeStatus = ref("disconnected");

  // WebSocket 会话ID映射: 前端sessionId -> 后端wsSessionId
  const wsSessionIdMap = ref({});

  const currentSession = computed(
    () => sessions.value.find((s) => s.id === currentSessionId.value) || null,
  );

  const currentMessages = computed(() => currentSession.value?.messages || []);

  // 获取当前会话的 WebSocket session_id
  const currentWsSessionId = computed(() => {
    if (!currentSessionId.value) return null;
    return wsSessionIdMap.value[currentSessionId.value] || null;
  });

  async function loadSessions() {
    isLoadingSessions.value = true;
    try {
      const data = await api.getSessions();
      sessions.value = (data || []).map((s) => ({
        id: String(s.id),
        chatId: s.chat_id,
        userId: s.user_id,
        title: s.metadata ? extractTitle(s.metadata) : "新对话",
        messages: [],
        createdAt: new Date(s.created_at).getTime(),
        updatedAt: new Date(s.updated_at).getTime(),
        raw: s,
      }));
      // 为每个会话设置 WebSocket 会话 ID 映射
      sessions.value.forEach((s) => {
        setWsSessionId(s.id, s.id);
      });
      if (sessions.value.length > 0 && !currentSessionId.value) {
        currentSessionId.value = sessions.value[0].id;
      }
    } catch (error) {
      console.error("加载会话列表失败:", error);
    } finally {
      isLoadingSessions.value = false;
    }
  }

  function extractTitle(metadata) {
    try {
      const meta = JSON.parse(metadata);
      return meta.title || "新对话";
    } catch {
      return "新对话";
    }
  }

  async function loadMessages(sessionId) {
    const session = sessions.value.find((s) => s.id === sessionId);
    if (!session) return;

    try {
      const messages = await api.getSessionMessages(sessionId);
      if (messages && Array.isArray(messages)) {
        session.messages = messages
          .map((m) => {
            const metadata = parseMessageMetadata(m.metadata);
            const thinking = m.thinking || metadata.reasoningContent || m.reasoning_content || "";
            return normalizeLoadedMessage({
            id: String(m.id),
            role: m.role,
            content: m.content,
            thinking,
            totalTokens: Number(m.total_tokens || 0),
            toolCalls: metadata.toolCalls,
            traceItems: metadata.traceItems.length > 0
              ? metadata.traceItems
              : buildTraceItems(thinking),
            created_at: new Date(m.created_at).getTime(),
            streaming: false,
            });
          })
          .filter(Boolean);
      }
    } catch (error) {
      console.error("加载消息失败:", error);
    }
  }

  async function createSession(title = "新对话") {
    try {
      const response = await api.createSession({
        user_id: userId.value,
        metadata: {
          title,
        },
      });
      // 后端返回格式: { code, message, data: { session_id, chat_id, user_id, key } }
      const data = response.data || response;
      console.log('data =>', data);
      const session = {
        id: String(data.session_id),
        chatId: data.chat_id,
        userId: data.user_id,
        title,
        messages: [],
        created_at: Date.now(),
        updated_at: Date.now(),
        raw: data,
      };
      sessions.value.unshift(session);
      currentSessionId.value = session.id;
      // REST API 创建的会话 ID 可以直接用于 WebSocket
      setWsSessionId(session.id, session.id);
      return session;
    } catch (error) {
      console.error("创建会话失败:", error);
      const session = {
        id: generateId(),
        chatId: generateId(),
        userId: userId.value,
        title,
        messages: [],
        createdAt: Date.now(),
        updatedAt: Date.now(),
      };
      sessions.value.unshift(session);
      currentSessionId.value = session.id;
      return session;
    }
  }

  async function switchSession(id) {
    currentSessionId.value = id;
    const session = sessions.value.find((s) => s.id === id);    
    if (session) {
      await loadMessages(id);
    }
  }

  async function deleteSession(id) {
    const idx = sessions.value.findIndex((s) => s.id === id);
    if (idx !== -1) {
      try {
        await api.deleteSession(id);
      } catch (error) {
        console.error("删除会话失败:", error);
      }
      sessions.value.splice(idx, 1);
      if (currentSessionId.value === id) {
        currentSessionId.value = sessions.value[0]?.id || null;
      }
    }
  }

  function updateSessionTitleLocal(id, title) {
    const session = sessions.value.find((s) => s.id === id);
    if (session) {
      session.title = title;
      session.updatedAt = Date.now();
    }
  }

  function addUserMessage(content) {
    if (!currentSession.value) {
      createSession();
      return null;
    }

    const msg = {
      id: generateId(),
      role: "user",
      content,
      created_at: Date.now(),
    };
    currentSession.value.messages.push(msg);
    currentSession.value.updatedAt = Date.now();

    if (currentSession.value.messages.length === 1) {
      const title = content.slice(0, 30) + (content.length > 30 ? "..." : "");
      updateSessionTitleLocal(currentSession.value.id, title);
    }

    return msg;
  }

  function addAIMessage() {
    if (!currentSession.value) return null;

    const msg = {
      id: generateId(),
      role: "assistant",
      content: "",
      thinking: "",
      totalTokens: 0,
      toolCalls: [],
      traceItems: [],
      created_at: Date.now(),
      streaming: true,
    };
    currentSession.value.messages.push(msg);
    currentSession.value.updatedAt = Date.now();
    return msg;
  }

  function appendToLastAI(content) {
    if (!currentSession.value) return;
    const msgs = currentSession.value.messages;
    const lastMsg = msgs[msgs.length - 1];
    if (lastMsg && lastMsg.role === "assistant") {
      lastMsg.content += content;
    }
  }

  function updateThinking(content) {
    if (!currentSession.value) return;
    const msgs = currentSession.value.messages;
    const lastMsg = msgs[msgs.length - 1];
    if (lastMsg && lastMsg.role === "assistant") {
      if (!content) return;

      let delta = content;
      if (lastMsg.thinking && content.startsWith(lastMsg.thinking)) {
        delta = content.slice(lastMsg.thinking.length);
      }
      if (!delta) return;

      lastMsg.thinking += delta;
      if (!Array.isArray(lastMsg.traceItems)) {
        lastMsg.traceItems = [];
      }

      const lastTraceItem = lastMsg.traceItems[lastMsg.traceItems.length - 1];
      if (lastTraceItem?.type === "thinking") {
        lastTraceItem.content += delta;
      } else {
        lastMsg.traceItems.push({
          id: generateId(),
          type: "thinking",
          content: delta,
        });
      }
    }
  }

  function finishLastAI(content) {
    if (!currentSession.value) return;
    const msgs = currentSession.value.messages;
    const lastMsg = msgs[msgs.length - 1];
    if (lastMsg && lastMsg.role === "assistant") {
      if (content !== undefined) lastMsg.content = content;
      lastMsg.streaming = false;
    }
  }

  function updateLastAITokens(totalTokens) {
    if (!currentSession.value) return;
    const msgs = currentSession.value.messages;
    const lastMsg = msgs[msgs.length - 1];
    if (lastMsg && lastMsg.role === "assistant") {
      lastMsg.totalTokens = Number(totalTokens || 0);
    }
  }

  function addToolCall(toolCall) {
    if (!currentSession.value) return;
    const msgs = currentSession.value.messages;
    const lastMsg = msgs[msgs.length - 1];
    if (lastMsg && lastMsg.role === "assistant") {
      if (!lastMsg.toolCalls) {
        lastMsg.toolCalls = [];
      }
      if (!Array.isArray(lastMsg.traceItems)) {
        lastMsg.traceItems = [];
      }
      const existing = lastMsg.toolCalls.find(tc => tc.id === toolCall.id);
      if (!existing) {
        const toolArguments = toolCall.arguments ?? toolCall.tool_args ?? "";
        lastMsg.toolCalls.push({
          id: toolCall.id,
          toolName: toolCall.tool_name,
          arguments: toolArguments,
          status: toolCall.status,
          content: "",
          error: null,
          timestamp: toolCall.timestamp,
        });
        lastMsg.traceItems.push({
          id: `tool-${toolCall.id}`,
          type: "tool_call",
          toolCallId: toolCall.id,
        });
      }
    }
  }

  function updateToolResult(toolResult) {
    if (!currentSession.value) return;
    const msgs = currentSession.value.messages;
    const lastMsg = msgs[msgs.length - 1];
    if (lastMsg && lastMsg.role === "assistant" && lastMsg.toolCalls) {
      const toolCall = lastMsg.toolCalls.find(tc => tc.id === toolResult.id);
      if (toolCall) {
        toolCall.status = toolResult.status;
        toolCall.content = toolResult.content;
        toolCall.error = toolResult.error;
      } else {
        lastMsg.toolCalls.push({
          id: toolResult.id,
          toolName: toolResult.tool_name || "unknown_tool",
          arguments: toolResult.arguments || "",
          status: toolResult.status,
          content: toolResult.content,
          error: toolResult.error,
          timestamp: toolResult.timestamp || Date.now(),
        });
      }

      if (!Array.isArray(lastMsg.traceItems)) {
        lastMsg.traceItems = [];
      }
      const traceExists = lastMsg.traceItems.some(
        (item) => item.type === "tool_call" && item.toolCallId === toolResult.id,
      );
      if (!traceExists) {
        lastMsg.traceItems.push({
          id: `tool-${toolResult.id}`,
          type: "tool_call",
          toolCallId: toolResult.id,
        });
      }
    }
  }

  function clearCurrentMessages() {
    if (!currentSession.value) return;
    currentSession.value.messages = [];
  }

  function ensureSession() {
    if (!currentSessionId.value || !currentSession.value) {
      createSession();
    }
    return currentSession.value;
  }

  function setApiBase(base) {
    apiBase.value = base;
    api.setApiBaseUrl(base);
    localStorage.setItem("icooclaw_api_base", base);
  }

  function setWsUrl(url) {
    try {
      const urlObj = new URL(url);
      wsHost.value = urlObj.hostname;
      wsPort.value = urlObj.port || (urlObj.protocol === 'wss:' ? '443' : '80');
      localStorage.setItem("icooclaw_ws_host", wsHost.value);
      localStorage.setItem("icooclaw_ws_port", wsPort.value);
    } catch (e) {
      console.error("Invalid URL:", e);
    }
  }

  function setWsHost(host) {
    wsHost.value = host;
    localStorage.setItem("icooclaw_ws_host", host);
  }

  function setWsPort(port) {
    wsPort.value = port;
    localStorage.setItem("icooclaw_ws_port", port);
  }

  function setWsPath(path) {
    wsPath.value = path;
    localStorage.setItem("icooclaw_ws_path", path);
  }

  function setWsConnected(connected) {
    wsConnected.value = connected;
  }

  function setApiHealth(status) {
    apiHealth.value = status;
  }

  function setWsRuntimeStatus(status) {
    wsRuntimeStatus.value = status;
  }

  function setUserId(id) {
    userId.value = id;
    localStorage.setItem("icooclaw_user_id", id);
  }

  // 设置 WebSocket 会话ID映射
  function setWsSessionId(frontendSessionId, wsSessionId) {
    wsSessionIdMap.value[frontendSessionId] = wsSessionId;
  }

  // 获取 WebSocket 会话ID
  function getWsSessionId(frontendSessionId) {
    return wsSessionIdMap.value[frontendSessionId] || null;
  }

  // 清除 WebSocket 会话ID
  function clearWsSessionId(frontendSessionId) {
    delete wsSessionIdMap.value[frontendSessionId];
  }

  return {
    sessions,
    currentSessionId,
    isLoading,
    isLoadingSessions,
    apiBase,
    wsUrl,
    wsHost,
    wsPort,
    wsPath,
    wsConnected,
    apiHealth,
    wsRuntimeStatus,
    userId,
    wsSessionIdMap,
    currentSession,
    currentMessages,
    currentWsSessionId,
    loadSessions,
    loadMessages,
    createSession,
    switchSession,
    deleteSession,
    updateSessionTitleLocal,
    addUserMessage,
    addAIMessage,
    appendToLastAI,
    updateThinking,
    finishLastAI,
    updateLastAITokens,
    addToolCall,
    updateToolResult,
    clearCurrentMessages,
    ensureSession,
    setApiBase,
    setWsUrl,
    setWsHost,
    setWsPort,
    setWsPath,
    setWsConnected,
    setApiHealth,
    setWsRuntimeStatus,
    setUserId,
    setWsSessionId,
    getWsSessionId,
    clearWsSessionId,
  };
});
