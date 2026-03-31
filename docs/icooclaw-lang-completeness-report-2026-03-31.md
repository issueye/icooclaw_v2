# icooclaw_lang 功能完备度评估报告

日期：`2026-03-31`  
评估对象：`icooclaw_lang/`

## 1. 总结

`icooclaw_lang` 已经从“解释器原型”进入“可交付脚本运行时”的阶段。主链路完整，CLI、打包、模块导入、并发、HTTP/WebSocket/SSE、数据库、YAML/TOML/JSON、对象方法、字段标注序列化、Windows 构建脚本，以及 VS Code 高亮插件都已经形成闭环。

如果把目标定义为“单机脚本语言 + 内嵌服务运行时”，当前完备度已经较高；如果把目标定义为“成熟通用语言生态”，则仍有明显缺口，主要集中在工具链深度、文档一致性、包管理和诊断体验。

综合评分：`8.9 / 10`

## 2. 评分明细

| 维度 | 分数 | 说明 |
| --- | --- | --- |
| 语言核心能力 | `9.1/10` | 变量、常量、函数、闭包、`if/for/while`、`try/catch`、`match`、模块导入导出、对象方法、匿名函数字面量、安全访问、`go` 已可用 |
| 运行时与并发 | `8.7/10` | 已有 runtime 统一协程池、`async.pool`、`wait_group`、CLI 和环境变量并发控制 |
| 标准库/内建库 | `9.1/10` | 文件、时间、路径、进程、日志、加密、HTTP、WebSocket、SSE、DB、JSON、TOML、YAML 已成体系，并支持 `__serde__` 字段标注与 schema 反序列化 |
| CLI 与交付能力 | `8.7/10` | `run/build/init/repl/version`、单文件 bundle、Windows 构建脚本都已具备 |
| 编辑器支持 | `7.4/10` | 已有 VS Code 语法高亮和 snippets，但还没有格式化、补全、诊断、LSP |
| 测试与稳定性 | `8.7/10` | 词法/语法/求值/并发/数据库/流式协议/回归测试覆盖较好，新增了 parser 回归测试和对象方法/安全访问/serde 测试 |
| 文档一致性 | `8.5/10` | `api-reference`、`ai-api`、`help` 已补齐到当前实现，但仍缺少更系统的快速开始和语言规范文档 |

## 3. 已具备的闭环能力

### 3.1 语言与运行时

- 解释器主链路完整：`lexer -> parser -> evaluator -> runtime`
- 已验证闭包、模块导入、错误捕获、模式匹配、协程执行
- `to_string()` 已作为通用方法下沉到所有运行时对象
- runtime 统一协程池已经接管 `Environment.Go(...)`
- CLI、REPL 和 bundled 可执行文件现在都支持可配置的内存上限保护，并支持按主机总内存百分比阈值控制
- 对象方法体系已经形成闭环：匿名 `fn(...) {}`、`fn (u user) rename(...)`、`this` / `self` / receiver 名注入都可用
- 对象与容器更新语义已经统一：变量、点字段、索引位都支持 `=`、复合赋值、`++`、`--`
- 安全访问已经覆盖字段、方法、索引和更新场景，并验证了连续链式访问

### 3.2 工程与交付

- `iclang run`
- `iclang build`
- `iclang init`
- `iclang repl`
- bundled 可执行文件运行
- Windows `build.bat` 构建与版本注入

### 3.3 内建库

- 数据与配置：`json`、`toml`、`yaml`
- 系统能力：`fs`、`os`、`path`、`exec`、`time`
- 服务能力：`http`、`websocket`、`sse`
- 存储能力：`db`
- 运行辅助：`async`、`log`、`crypto`
- 数据编解码支持 `__serde__` 字段标注，覆盖序列化别名、跳过字段、`omitempty`
- `json.parse(text, schema)`、`yaml.parse(text, schema)`、`toml.parse(text, schema)` 已支持按 schema 把外部字段名回填为内部字段名

### 3.4 工具链

- 示例脚本较完整
- VS Code 扩展可打包为 VSIX
- 已有构建与测试路径

## 4. 本轮新增能力

这一轮能力提升，直接提高了语言“可写业务对象”和“可对接真实配置/接口”的完备度：

- 对象允许定义和挂载方法，既支持对象字面量内联方法，也支持 receiver 风格声明
- `HASH` / `ARRAY` 的点访问、索引访问、复合赋值、自增自减语义已经趋于统一
- 安全访问从只读扩展到了更新语义，例如 `obj?.field += 1`、`arr?[0]++`
- 连续安全访问已可稳定支持 `user?.profile?.name`、`user?.get_profile()?.name`
- JSON/YAML/TOML 已支持基于 `__serde__` 的字段标注和基于 `schema` 的反序列化字段回填

## 5. 主要短板

### 5.1 还不是完整生态

目前更像“带丰富内建库的嵌入式脚本运行时”，还不是成熟语言生态。缺的不是能不能跑，而是：

- 还没有格式化器
- 还没有 LSP 级补全/诊断
- 还没有依赖管理或包仓库
- 还没有正式语言规范或兼容性承诺

### 5.2 文档一致性已改善，但还不够体系化

前一轮存在的典型问题主要是实现和文档漂移：

- `docs/icooclaw-lang-ai-api.md` 未及时反映 `async`、`yaml`、`toml`、`init`、runtime 并发配置、`to_string()`
- CLI `help` 的 builtin 列表漏写了 `exec`、`toml`
- `api-reference` 漏写 `toml` 和通用 `to_string()`

这些问题本轮已经完成修正，但体系化文档仍然不足，尤其缺：

- 从 `init -> run -> build` 串起来的快速开始
- 独立的对象模型和方法系统说明
- 独立的序列化标注与 schema 反序列化指南

### 5.3 并发与原生库还有继续深化空间

当前并发模型已经可用，但还偏“基础可控”，例如：

- `async.pool` 还没有结果收集
- `async.pool` / `wait_group` 还没有超时等待
- runtime 还没有对外暴露更细的 stats

## 6. 风险判断

当前项目适合：

- 编写自动化脚本
- 构建轻量 HTTP/WS/SSE 服务
- 编写单机工具程序
- 做内嵌脚本运行时或实验性业务 DSL

当前项目暂不适合直接宣称为：

- 完整通用编程语言生态
- 具备成熟 IDE 体验的生产级开发平台
- 拥有稳定第三方包生态的脚本平台

## 7. 建议优先级

### P1

- 增加格式化器或至少定义官方代码风格
- 暴露 runtime stats，补足并发诊断
- 给 `async.pool` 增加结果和错误收集
- 增加对象模型专题文档，固定 receiver / `this` / `self` / 安全访问语义
- 补一个更细的 runtime 资源诊断接口，至少包含内存与任务队列信息

### P2

- 做 VS Code 补全、诊断、跳转
- 让 `build` / manifest 配置能力继续标准化
- 补一个面向用户的“快速开始”文档
- 增加 `serde` 专题文档，说明 `__serde__`、`omitempty`、schema 回填的边界

### P3

- 设计包管理/依赖声明方案
- 固化语言规范与版本兼容策略
- 增加更多标准库一致性约束和错误码约定

## 8. 本轮补齐项

本次已同步修正：

- `docs/icooclaw-lang-ai-api.md`
- `docs/icooclaw-lang-api-reference.md`
- `icooclaw_lang/cmd/iclang/main.go` 中的 REPL `help` 内建库列表
- `docs/icooclaw-lang-completeness-report-2026-03-31.md` 的评分和能力评估

结论上，`icooclaw_lang` 现在可以被评价为“语言与运行时能力已经较完整、对象编程和序列化能力可用、但生态仍在早中期”的脚本语言运行时。
