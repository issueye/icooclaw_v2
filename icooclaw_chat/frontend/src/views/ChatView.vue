<template>
  <div class="page-shell">
    <div class="page-frame">
      <ChatSidebar
        :sessions="chatStore.sessions"
        :current-session-id="chatStore.currentSessionId"
        :ws-status="wsStatus"
        :collapsed="sidebarCollapsed"
        @new="handleNewChat"
        @select="handleSelectSession"
        @delete="handleDeleteSession"
        @toggle="sidebarCollapsed = !sidebarCollapsed"
      />

      <section class="chat-workspace surface-panel page-panel subtle-grid">
        <div class="chat-workspace-inner">
          <ChatHeader
            :title="chatStore.currentSession?.title"
            :session-id="chatStore.currentSessionId"
            :sidebar-collapsed="sidebarCollapsed"
            @toggle-sidebar="sidebarCollapsed = !sidebarCollapsed"
            @new-chat="handleNewChat"
            @open-settings="router.push('/settings')"
          />

          <ChatConversationPane
            ref="conversationPaneRef"
            :messages="chatStore.currentMessages"
            :hints="hints"
            :show-scroll-button="showScrollButton"
            @hint="sendMessage"
            @scroll-bottom="scrollToBottom"
            @scroll-state="showScrollButton = $event"
          />

          <div class="chat-footer">
            <ChatInput
              ref="chatInputRef"
              :disabled="chatStore.isLoading"
              @send="sendMessage"
            />
          </div>
        </div>
      </section>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, watch } from "vue";
import { useRouter } from "vue-router";
import {
  BotIcon,
  SparklesIcon,
  CodeIcon,
  LightbulbIcon,
} from "lucide-vue-next";

import ChatConversationPane from "@/components/ChatConversationPane.vue";
import ChatSidebar from "@/components/ChatSidebar.vue";
import ChatHeader from "@/components/ChatHeader.vue";
import ChatInput from "@/components/ChatInput.vue";

import { useChatStore } from "@/stores/chat";
import { useWebSocket } from "@/composables/useWebSocket";

const router = useRouter();
const chatStore = useChatStore();

// ===== WebSocket =====
const {
  status: wsStatus,
  connect,
  send,
  onMessage,
} = useWebSocket();

function connectWs() {
  const url = buildSessionWebSocketUrl(
    chatStore.wsUrl,
    chatStore.currentSessionId,
  );
  connect(url);
}

function buildSessionWebSocketUrl(baseUrl, sessionId) {
  if (!sessionId) {
    return baseUrl;
  }

  try {
    const url = new URL(baseUrl);
    const basePath = url.pathname.replace(/\/+$/, "");
    url.pathname = `${basePath}/${encodeURIComponent(sessionId)}`;
    url.searchParams.delete("session_id");
    return url.toString();
  } catch (error) {
    console.warn("构建 WebSocket 会话地址失败，回退到原始地址", error);
    return `${baseUrl.replace(/\/+$/, "")}/${encodeURIComponent(sessionId)}`;
  }
}

// ===== 消息处理 =====
onMessage((msg) => {
  // 处理后端返回的消息
  switch (msg.type) {
    case "session_created":
      // 会话创建成功，保存 session_id
      if (msg.data && msg.data.session_id) {
        chatStore.setWsSessionId(
          chatStore.currentSessionId,
          msg.data.session_id,
        );
        // 发送聊天消息
        sendPendingChatMessage();
      }
      break;
    case "chunk":
      // 处理流式内容块
      chatStore.appendToLastAI(msg.data?.content || "");
      if (msg.data?.total_tokens) {
        chatStore.updateLastAITokens(msg.data.total_tokens);
      }
      // 处理推理内容（DeepSeek 等模型支持）
      if (msg.data?.reasoning) {
        chatStore.updateThinking(
          (chatStore.currentSession?.messages?.slice(-1)[0]?.thinking || "") +
            msg.data.reasoning,
        );
      }
      scrollToBottom();
      break;
    case "thinking":
      chatStore.updateThinking(cleanThinkingContent(msg.data?.content || ""));
      break;
    case "tool_call":
      chatStore.addToolCall({
        id: msg.data?.tool_call_id,
        tool_name: msg.data?.tool_name,
        arguments: msg.data?.arguments ?? msg.data?.tool_args ?? "",
        tool_args: msg.data?.tool_args ?? msg.data?.arguments ?? "",
        status: "pending",
        timestamp: Date.now(),
      });
      scrollToBottom();
      break;
    case "tool_result":
      chatStore.updateToolResult({
        id: msg.data?.tool_call_id,
        status: "completed",
        content: msg.data?.result,
      });
      scrollToBottom();
      break;
    case "end":
      if (msg.data?.total_tokens) {
        chatStore.updateLastAITokens(msg.data.total_tokens);
      }
      chatStore.finishLastAI();
      chatStore.isLoading = false;
      scrollToBottom();
      break;
    case "error":
      chatStore.finishLastAI("[错误] " + (msg.error?.message || "未知错误"));
      chatStore.isLoading = false;
      break;
    case "queue_status":
      // 队列状态更新，可以显示给用户
      console.log("队列状态:", msg.data);
      break;
    case "pong":
      // 心跳响应
      break;
    default:
      console.log("未处理的消息类型:", msg.type, msg);
  }
});

// 待发送的聊天消息
let pendingChatContent = null;

function sendPendingChatMessage() {
  if (!pendingChatContent) return;

  const wsSessionId = chatStore.getWsSessionId(chatStore.currentSessionId);
  if (!wsSessionId) {
    console.error("没有有效的 WebSocket 会话ID");
    chatStore.finishLastAI("⚠️ 创建会话失败：无法获取会话ID");
    chatStore.isLoading = false;
    pendingChatContent = null;
    return;
  }

  const sent = send({
    type: "chat",
    session_id: wsSessionId,
    content: pendingChatContent,
    stream: true, // 启用流式响应
  });

  if (!sent) {
    chatStore.finishLastAI("⚠️ 发送失败：WebSocket 连接缓冲区已满，请稍后重试");
    chatStore.isLoading = false;
  }

  pendingChatContent = null;
}

async function sendMessage(text) {
  if (!text?.trim()) return;

  chatStore.ensureSession();

  if (wsStatus.value !== "connected" && wsStatus.value !== "reconnecting") {
    connectWs();

    const maxWaitTime = 10000;
    const checkInterval = 200;
    let waitedTime = 0;

    while (wsStatus.value !== "connected" && waitedTime < maxWaitTime) {
      await new Promise((r) => setTimeout(r, checkInterval));
      waitedTime += checkInterval;
    }

    if (wsStatus.value !== "connected") {
      chatStore.addUserMessage(text);
      chatStore.addAIMessage();
      chatStore.finishLastAI("⚠️ 连接失败：请检查 Agent 服务是否启动");
      return;
    }
  }

  chatStore.addUserMessage(text);
  scrollToBottom();

  chatStore.addAIMessage();
  chatStore.isLoading = true;
  scrollToBottom();

  // 检查是否已有 WebSocket 会话
  const wsSessionId = chatStore.getWsSessionId(chatStore.currentSessionId);

  if (wsSessionId) {
    // 已有会话，直接发送聊天消息（启用流式响应）
    const sent = send({
      type: "chat",
      session_id: wsSessionId, // 后端需要这个字段进行校验
      content: text,
      stream: true, // 启用流式响应
    });

    if (!sent) {
      chatStore.finishLastAI(
        "⚠️ 发送失败：WebSocket 连接缓冲区已满，请稍后重试",
      );
      chatStore.isLoading = false;
    }
  } else {
    // 需要先创建 WebSocket 会话
    pendingChatContent = text;
    // 传递前端已有的 session_id，让后端复用该会话
    send({
      type: "create_session",
      data: {
        session_id: chatStore.currentSessionId,
        channel: "websocket",
        user_id: chatStore.userId,
      },
    });
  }
}

// ===== 会话操作 =====
function handleNewChat() {
  chatStore.createSession();
  chatInputRef.value?.focus();
}

async function handleSelectSession(id) {
  console.log("切换会话:", id);

  await chatStore.switchSession(id);
  if (window.innerWidth < 768) {
    sidebarCollapsed.value = true;
  }
  scrollToBottom();
}

function handleDeleteSession(id) {
  console.log("删除会话:", id);
  chatStore.deleteSession(id);
}

// ===== UI 状态 =====
const sidebarCollapsed = ref(false);
const conversationPaneRef = ref(null);
const chatInputRef = ref(null);
const showScrollButton = ref(false);

const hints = [
  { text: "你好，请介绍一下你自己", icon: BotIcon },
  { text: "帮我写一段 Python 快速排序代码", icon: CodeIcon },
  { text: "帮我分析这段代码的问题", icon: LightbulbIcon },
  { text: "给我讲个有趣的笑话", icon: SparklesIcon },
];

// 清理思考内容中的模型特定标签
function cleanThinkingContent(content) {
  if (!content) return "";
  return content
    .replace(/<think>/g, "")
    .replace(/<\/think>/g, "")
    .replace(/<\|start_header_id\|>reasoning<\|end_header_id\|>/g, "")
    .replace(/<\|start_header_id\|>assistant<\|end_header_id\|>/g, "")
    .replace(/<\|message\|>/g, "")
    .trim();
}

function scrollToBottom() {
  conversationPaneRef.value?.scrollToBottom();
  showScrollButton.value = false;
}

watch(
  () => chatStore.currentMessages,
  () => scrollToBottom(),
  { deep: true },
);

// ===== 初始化 =====
onMounted(async () => {
  if (window.innerWidth < 768) {
    sidebarCollapsed.value = true;
  }

  await chatStore.loadSessions();
  if (chatStore.currentSessionId) {
    await chatStore.loadMessages(chatStore.currentSessionId);
  }
});

// 监听 WebSocket 连接状态变化
watch(wsStatus, (newStatus) => {
  chatStore.setWsConnected(newStatus === "connected");
  chatStore.setWsRuntimeStatus(newStatus);
});
</script>

<style scoped>
.chat-workspace {
  flex: 1;
  min-width: 0;
  min-height: 0;
  overflow: hidden;
}

.chat-workspace-inner {
  position: relative;
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 0;
}

.chat-footer {
  padding: 8px;
  background: var(--color-bg-secondary);
  border-top: 1px solid var(--color-border);
}

[data-theme="dark"] .chat-footer {
  background: var(--color-bg-secondary);
}
</style>
