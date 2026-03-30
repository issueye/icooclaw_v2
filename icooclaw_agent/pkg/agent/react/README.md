# `pkg/agent/react`

`ReActAgent` 的实现已经按“入口层 / 运行时编排 / 工具协议 / trace 持久化”做了初步拆分。
这个目录后续继续重构时，应优先保持这些边界，而不是把新逻辑重新堆回 `index.go` 或 `runtime_helpers.go`。

## 文件职责

- `index.go`
  只保留公开类型、依赖定义、构造函数。
- `chat.go`
  非流式对话入口，负责同步路径的高层编排。
- `chat_stream.go`
  流式对话入口，负责流式路径的高层编排和 chunk callback 适配。
- `execution_runtime.go`
  单轮迭代执行期结构、循环控制、sync/stream collect 阶段。
- `runtime_helpers.go`
  单轮开始、请求日志、统一收尾、stream done 等迭代辅助逻辑。
- `message_runtime.go`
  消息清洗、`prepareChat`、请求前消息准备。
- `prompt_runtime.go`
  prompt/message 构建、工具定义转换。
- `provider_runtime.go`
  provider 与 model 选择、agent logger 获取。
- `tool_protocol.go`
  工具调用参数解析、流式 tool call 合并与校验。
- `tool_runtime.go`
  工具执行、assistant/tool message 回灌。
- `trace_persistence.go`
  thinking/tool trace 组装、assistant 消息落库、metadata 构建。
- `hook_runtime.go`
  `ReactHooks` 调用封装。
- `agent_context.go`
  agent profile 与系统提示词附加上下文。
- `summary.go`
  对话摘要能力。
- `think_filter.go`
  `<think>` 流式过滤与文本剥离。
- `hooks.go`
  hook 接口定义。

## 当前执行流

### 同步路径

1. `Chat` 调 `prepareChat`
2. `RunLLM` 调 `runIterationLoop`
3. 每轮执行：
   - `beginLLMIteration`
   - `collectSyncIteration`
   - `resolveIterationOutcome`
4. 完成后落 assistant 消息并触发 `finishAgentRun`

### 流式路径

1. `ChatStream` 调 `prepareChat`
2. `RunLLMStream` 调 `runIterationLoop`
3. 每轮执行：
   - `beginLLMIteration`
   - `collectStreamIteration`
   - `resolveStreamCollectorOutcome`
   - `applyStreamIterationResolution`
4. 完成后触发 `finishAgentRun`

## 约束

- 新增“迭代控制流”逻辑，优先放到 `execution_runtime.go`
- 新增“工具协议适配”逻辑，优先放到 `tool_protocol.go`
- 新增“trace / assistant metadata / 持久化”逻辑，优先放到 `trace_persistence.go`
- 不要把新的运行期 helper 继续堆回 `index.go`
- `chat.go` / `chat_stream.go` 应保持为薄入口，不承载底层细节

## 下一步建议

- 把 `runtime_helpers.go` 继续压缩成更纯粹的 iteration helper 文件
- 为 `runIterationLoop`、`collectStreamIteration`、`resolveIterationOutcome` 增加更明确的针对性测试
- 如果后续继续重构，可考虑把 sync/stream 结果统一成单一 `iterationCollector` 抽象
