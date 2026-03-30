# icooclaw_v2 代码精简计划

> 生成日期：2026-03-30  
> 配套文档：`code-redundancy-analysis-report.md`  
> 目标：在保证功能完备的前提下，精简约 18% 的冗余代码（~3,650 行）

---

## 一、总体原则

1. **功能零回归** — 每个阶段完成后必须通过现有测试
2. **渐进式重构** — 按优先级分 5 个阶段执行，每阶段独立可验证
3. **先删后合** — 优先删除死代码（零风险），再进行合并重构
4. **保持接口兼容** — 对外 API 不变，仅重构内部实现

---

## 二、阶段划分

### 阶段一：删除死代码（预估节省 ~530 行）

> 风险：极低 | 预估耗时：1 小时

| 任务 | 文件 | 操作 |
|------|------|------|
| T1.1 | `frontend/src/composables/useChat.js`（170 行） | 删除整个文件 |
| T1.2 | `frontend/src/composables/useWailsChat.js`（115 行） | 删除整个文件 |
| T1.3 | `frontend/src/components/ModeSwitch.vue`（37 行） | 删除整个文件 |
| T1.4 | `frontend/src/components/SettingsDialog.vue`（170 行） | 删除整个文件 |
| T1.5 | `frontend/src/mocks/chalk.js`（23 行） | 删除整个文件 |
| T1.6 | `frontend/src/services/wails.js:88-124`（EventEmitter + WailsEvents） | 删除未使用的代码段 |
| T1.7 | 全局搜索确认上述文件无任何引用 | 验证 |

**验证步骤：**
- [ ] `npm run build` 编译通过
- [ ] 全文搜索被删文件名，确认无 import/require 引用
- [ ] 手动启动应用，验证功能正常

---

### 阶段二：消除直接重复（预估节省 ~400 行）

> 风险：低 | 预估耗时：2-3 小时

#### T2.1 统一 `isWailsEnv()` 定义

- **当前：** 3 处定义（`http.js:3-5`、`wails.js:7-9`、`App.vue:239-241`）
- **目标：** 仅保留 `services/wails.js` 中的导出版本
- **操作：**
  1. 在 `wails.js` 中确认 `isWailsEnv` 已 export
  2. 修改 `http.js`，改为 `import { isWailsEnv } from './wails.js'`
  3. 修改 `App.vue`，改为 `import { isWailsEnv } from './services/wails.js'`
  4. 删除 `http.js` 和 `App.vue` 中的本地定义

#### T2.2 ChatSidebar 会话按钮提取为子组件

- **当前：** `ChatSidebar.vue` 中 4 段近乎相同的模板（行 80-178）
- **目标：** 创建 `SessionItem.vue` 子组件
- **操作：**
  1. 新建 `components/SessionItem.vue`，接受 `session`、`isActive` props，emit `select`、`delete` 事件
  2. 在 `ChatSidebar.vue` 中导入并替换 4 段重复模板
  3. 合并 4 个分组为统一的 `groupedSessions` 计算属性

#### T2.3 后端消息类型统一

- **当前：** `pkg/bus/bus.go` 和 `pkg/channels/models/message.go` 重复定义 4 个结构体
- **目标：** 保留 `pkg/bus/bus.go` 中的定义，`channels/models` 改为引用
- **操作：**
  1. 在 `channels/models/message.go` 中 import `bus` 包，使用类型别名或嵌入
  2. 修改 `BusToOutMessage()` 为简单的类型转换
  3. 全局搜索确保无编译错误

#### T2.4 Page 类型统一

- **当前：** `pkg/storage/model.go` 用 `int64`，`pkg/gateway/models/model.go` 用 `int`
- **目标：** 统一为一个定义
- **操作：**
  1. 删除 `pkg/gateway/models/model.go` 中的 `Page` 定义
  2. 修改 gateway 层 import `pkg/storage` 的 `Page`
  3. 若存在循环依赖，则将 `Page` 提取到独立的 `pkg/types/` 包

#### T2.5 错误定义合并

- **当前：** `pkg/errors/` 和 `pkg/channels/errs/` 有 3 个重复错误
- **目标：** 统一到 `pkg/errors/`
- **操作：**
  1. 将 `channels/errs` 中的 `ClassifySendError`、`IsRetriable` 迁移至 `pkg/errors/`
  2. 全局替换 channels 中对 `errs.ErrXxx` 的引用为 `errors.ErrXxx`
  3. 删除 `pkg/channels/errs/` 包

#### T2.6 工具函数合并

- **当前：** `mustMarshalJSON`、`firstNonEmpty`/`firstString`、`truncate` 分散在各处
- **目标：** 统一到 `pkg/utils/`
- **操作：**
  1. 在 `pkg/utils/json.go` 中添加 `MustMarshalJSON`
  2. 在 `pkg/utils/string.go` 中添加 `FirstNonEmpty`、`Truncate`
  3. 替换所有引用

**验证步骤：**
- [ ] `go build ./...` 编译通过
- [ ] `go test ./...` 测试通过
- [ ] `npm run build` 前端编译通过
- [ ] 手动验证核心功能

---

### 阶段三：后端模式整合（预估节省 ~1,200 行）

> 风险：中 | 预估耗时：4-6 小时

#### T3.1 Provider 实现统一 — 最大收益项

- **当前：** 6 个文件（~600 行）各自手写 Chat/ChatStream
- **目标：** 全部改用 `openAICompatibleProvider`
- **操作：**
  1. 扩展 `openAICompatibleProfile` 结构体，增加可选字段：
     - `CustomHeaders map[string]string`
     - `URLBuilder func(model, action string) string`
  2. 将 `groq.go` 改为构造函数：`NewGroqProvider(cfg) Provider { return newOpenAICompatible(profile) }`
  3. 依次处理 `mistral.go`、`grok.go`、`zhipu.go`、`silicon_flow.go`、`moonshot.go`
  4. 处理 `openrouter.go`（需 CustomHeaders）和 `azure_openai.go`（需 URLBuilder）
  5. 每改一个 Provider，运行相关测试确认

**参考模式（现有正确实现）：**
```go
// deepseek.go — 已正确使用
func NewDeepSeekProvider(cfg ProviderConfig) protocol.Provider {
    return newOpenAICompatibleProvider(openAICompatibleProfile{...}, cfg)
}
```

**目标模式：**
```go
// groq.go — 改造后
func NewGroqProvider(cfg ProviderConfig) protocol.Provider {
    return newOpenAICompatibleProvider(openAICompatibleProfile{
        baseURL: "https://api.groq.com/openai/v1",
        chatPath: "/chat/completions",
    }, cfg)
}
```

#### T3.2 Channel 通用逻辑提取

- **当前：** 4 个 Channel 重复 `IsAllowed` + `ParseConfig`
- **目标：** 提取通用基类/函数
- **操作：**
  1. 在 `pkg/channels/models/` 创建 `base.go`：
     ```go
     type BaseConfig struct {
         Enabled         bool
         AllowFrom       []string
         ReasoningChatID string
     }
     func ParseBaseConfig(raw map[string]any) BaseConfig { ... }
     func IsSenderAllowed(allowFrom []string, senderID string) bool { ... }
     ```
  2. 各 Channel 的 config 嵌入 `BaseConfig`
  3. 各 Channel 的 `IsAllowed` 改为调用 `IsSenderAllowed`

#### T3.3 Storage Page() 统一使用 pageQuery

- **当前：** skill、task、binding、memory 自行实现分页
- **目标：** 统一使用 `pageQuery[T]()`
- **操作：**
  1. 检查 `pageQuery` 是否支持自定义排序（已有 `order` 参数）
  2. 重构 `SkillStorage.Page()` 使用 `pageQuery`
  3. 重构 `TaskStorage.Page()` 使用 `pageQuery`
  4. 重构 `BindingStorage.Page()` 使用 `pageQuery`
  5. 重构 `MemoryStorage.Search()` 使用 `pageQuery`

#### T3.4 Query/Response 泛型化

- **当前：** 11 个存储文件各自定义 `QueryXxx`/`ResQueryXxx`
- **目标：** 使用 Go 泛型统一
- **操作：**
  1. 在 `pkg/storage/` 创建 `types.go`：
     ```go
     type PagedQuery[T any] struct {
         Page    Page   `json:"page"`
         KeyWord string `json:"key_word"`
         Filter  T      `json:"filter,omitempty"`
     }
     type PagedResult[T any] struct {
         Page    Page `json:"page"`
         Records []T  `json:"records"`
     }
     ```
  2. 逐步替换各存储文件中的定义

#### T3.5 Gateway Handler 使用 standardResourceHandler

- **当前：** SkillHandler 和 TaskHandler 手写完整 CRUD
- **目标：** 嵌入 `standardResourceHandler`
- **操作：**
  1. SkillHandler 嵌入 `standardResourceHandler`，保留 Export/Import/Install/Upsert/GetByName/Save
  2. TaskHandler 嵌入 `standardResourceHandler`，添加 `beforeCreate`/`beforeUpdate` 钩子用于 normalizeTask 和 syncTask

**验证步骤：**
- [ ] `go build ./...` 编译通过
- [ ] `go test ./...` 测试通过
- [ ] 启动 Agent 服务，通过 Chat 客户端执行完整的对话流程
- [ ] 验证各 Provider 可正常调用
- [ ] 验证各 Channel 消息收发正常

---

### 阶段四：前端模式整合（预估节省 ~600 行）

> 风险：中 | 预估耗时：3-4 小时

#### T4.1 API 服务 CRUD 工厂化

- **当前：** 7 个 API 文件共享相同 CRUD 模式
- **目标：** 创建工厂函数
- **操作：**
  1. 在 `services/common-api.js` 中添加：
     ```js
     export function createCrudApi(resourceName) {
       return {
         getPage: (params) => request(`/api/v1/${resourceName}/page`, { method: 'POST', body: JSON.stringify(createPageRequest(params)) }),
         getAll: () => request(`/api/v1/${resourceName}`, { method: 'POST' }),
         getById: (id) => request(`/api/v1/${resourceName}/get`, { method: 'POST', body: JSON.stringify({ id }) }),
         create: (data) => request(`/api/v1/${resourceName}/create`, { method: 'POST', body: JSON.stringify(data) }),
         update: (data) => request(`/api/v1/${resourceName}/update`, { method: 'POST', body: JSON.stringify(data) }),
         delete: (id) => request(`/api/v1/${resourceName}/delete`, { method: 'POST', body: JSON.stringify({ id }) }),
       };
     }
     ```
  2. 逐个改造 `providers-api.js`、`channels-api.js`、`mcp-api.js`、`memories-api.js`、`tasks-api.js`、`agents-api.js`
  3. `skills-api.js` 因有额外方法（export/import/install），保留额外方法 + 扩展基础工厂

#### T4.2 创建 MetricCard 组件

- **当前：** 6 个页面重复相同的指标卡片 HTML
- **目标：** 创建可复用组件
- **操作：**
  1. 新建 `components/common/MetricCard.vue`：
     ```vue
     <script setup>
     defineProps({ icon: Object, color: String, value: [String, Number], label: String })
     </script>
     ```
  2. 在 6 个页面中替换为 `<MetricCard />`

#### T4.3 创建 useDialogState 组合式函数

- **当前：** 6 个管理组件重复 open/close/reset 模式
- **目标：** 封装为可复用函数
- **操作：**
  1. 新建 `composables/useDialogState.js`：
     ```js
     export function useDialogState(initialForm) {
       const showDialog = ref(false);
       const editingItem = ref(null);
       const form = reactive({ ...initialForm });
       const dialogVisible = computed({ ... });
       function resetForm() { Object.assign(form, initialForm); }
       function closeDialog() { showDialog.value = false; editingItem.value = null; resetForm(); }
       function openForCreate() { resetForm(); showDialog.value = true; }
       function openForEdit(item) { Object.assign(form, item); editingItem.value = item; }
       return { showDialog, editingItem, form, dialogVisible, resetForm, closeDialog, openForCreate, openForEdit };
     }
     ```
  2. 在 6 个管理组件中替换

#### T4.4 Store loading/error 模式封装

- **当前：** 40+ 处重复 try/catch/finally
- **目标：** 创建通用异步包装函数
- **操作：**
  1. 新建 `stores/helpers.js`：
     ```js
     export async function withLoading(loading, error, fn) {
       loading.value = true;
       error.value = null;
       try { return await fn(); }
       catch (e) { error.value = e.message; throw e; }
       finally { loading.value = false; }
     }
     ```
  2. 在 `provider.js` 和 `skill.js` 中替换所有重复模式

#### T4.5 提取共享常量

- **当前：** `statusOptions` 和 localStorage key 分散
- **目标：** 集中管理
- **操作：**
  1. 新建 `constants/index.js`，导出 `STATUS_FILTER_OPTIONS`
  2. 新建 `constants/storage-keys.js`，导出所有 localStorage key 常量
  3. 替换所有引用

**验证步骤：**
- [ ] `npm run build` 编译通过
- [ ] 无 ESLint 报错
- [ ] 手动验证所有管理页面的 CRUD 功能
- [ ] 验证对话功能正常

---

### 阶段五：结构优化（预估节省 ~150 行 + 结构清晰度提升）

> 风险：低-中 | 预估耗时：2-3 小时

#### T5.1 配置文件统一

- **操作：**
  1. 删除 `icooclaw_agent/pkg/config/config.example.toml`（与根目录的 `config.example.toml` 漂移）
  2. 确保根目录 `config.example.toml` 是唯一示例配置
  3. 在 `config.go` 的 `WriteDefaultConfig()` 中添加注释说明示例配置位置

#### T5.2 Hook 系统评估与清理

- **操作：**
  1. 全局搜索 `pkg/hooks/` 中 `CompositeHooks`、`LoggingHooks` 的引用
  2. 若无引用，删除 `pkg/hooks/` 整个目录
  3. 若有引用，将其与 `pkg/app/hooks*.go` 合并为统一系统

#### T5.3 Provider 包装链简化

- **操作：**
  1. 评估是否可以让 platform 实现直接嵌入 `*adapter.BaseProvider`
  2. 若可行，删除 `platforms/types.go` 中的 `BaseProvider` 包装层

#### T5.4 前端 wails.js 简化

- **操作：**
  1. 让 `wailsService` 对象使用 `wailsjs/go/services/App.js` 的自动生成绑定
  2. 删除手动包装的方法

**验证步骤：**
- [ ] `go build ./...` 编译通过
- [ ] `go test ./...` 测试通过
- [ ] `npm run build` 编译通过
- [ ] 端到端功能验证

---

## 三、执行时间线

```
阶段一（死代码清理）     ──── Day 1 ──── 1h
阶段二（直接重复消除）   ──── Day 1-2 ── 2-3h
阶段三（后端模式整合）   ──── Day 2-4 ── 4-6h
阶段四（前端模式整合）   ──── Day 3-5 ── 3-4h
阶段五（结构优化）       ──── Day 5 ──── 2-3h
```

**总计预估：** 12-17 小时

---

## 四、风险控制

| 风险点 | 应对措施 |
|--------|---------|
| Provider 重构导致 API 调用失败 | 每改一个 Provider 立即测试；保留原代码注释对比 |
| Storage 泛型化引入编译错误 | 先在一个存储类型上验证，确认模式后再推广 |
| 前端组合式函数改写影响响应性 | 使用 Vue DevTools 检查响应性是否保持 |
| 配置文件合并导致部署问题 | 在 .gitignore 中保留运行时配置的备份 |

---

## 五、预期成果

| 指标 | 当前 | 优化后 | 改善 |
|------|------|--------|------|
| 源文件数 | ~200 | ~185 | -7.5% |
| 代码总行数 | ~20,000 | ~16,350 | -18% |
| 重复定义数 | 29 处 | 0 处 | -100% |
| 未使用代码 | ~530 行 | 0 行 | -100% |
| 新增共享组件/函数 | — | ~8 个 | 复用性提升 |
