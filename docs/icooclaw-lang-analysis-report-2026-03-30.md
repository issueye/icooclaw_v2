# icooclaw_lang 分析报告

日期：2026-03-30  
目标目录：`icooclaw_lang/`

## 1. 结论摘要

`icooclaw_lang` 当前已经具备一套可运行的解释器雏形，主链路完整：`Lexer -> Parser -> AST -> Evaluator -> Object/Environment -> Builtins -> CLI`。基础能力如变量赋值、函数、`if/for/while`、数组、哈希、内置函数、字符串/数组方法已经可以工作。

但这个模块目前仍处于“原型可运行、语言未闭环”的状态，主要问题不在编译，而在语义完整性和语法稳定性：

- 可以 `go build` / `go test`，但 `go test` 仅覆盖编译，无任何行为测试。
- `match` 是当前最明显的断点，设计文档和示例把它当成核心特性，但实际解析逻辑不完整，示例无法运行。
- 解析器对“单行块语句”存在死循环风险，例如 `if true { print("x") }`。
- 常量重赋值没有向用户报错，属于语义错误被吞掉。
- `import/export/go/select/interface/type` 中只有部分做了占位，绝大多数尚未真正实现。

整体判断：适合作为继续迭代的语言内核原型，不适合作为稳定脚本运行时直接交付。

## 2. 当前实现结构

### 2.1 模块分层

- CLI 入口：`cmd/iclang/main.go`
- 词法分析：`internal/lexer/token.go`、`internal/lexer/lexer.go`
- AST：`internal/ast/ast.go`
- 语法分析：`internal/parser/parser.go`
- 运行时对象与作用域：`internal/object/object.go`、`internal/object/environment.go`
- 解释执行：`internal/evaluator/evaluator.go`
- 内置函数：`internal/builtin/builtin.go`

### 2.2 已落地能力

- 标量类型：`INTEGER`、`FLOAT`、`STRING`、`BOOLEAN`、`NULL`
- 容器类型：数组、哈希
- 语句：函数定义、`return`、`if/else`、`for in`、`while`、`const`、`break`、`continue`、`try/catch`
- 表达式：前缀、算术、比较、逻辑、赋值、复合赋值、函数调用、索引、点访问、方法调用
- 运行时能力：闭包环境、嵌套作用域、数组/哈希索引赋值
- 内置函数：`print`、`println`、`len`、`range`、`type`、`str`、`int`、`float`、`input`、`push`、`pop`、`keys`、`values`、`abs`、`read_file`、`write_file`
- 方法调用：
  - 字符串：`len/upper/lower/trim/split/contains/starts_with/ends_with`
  - 数组：`len/push/pop/join/contains`

## 3. 设计与实现差异

对照 `docs/plans/2026-03-30-icooclaw-lang-design.md`，当前实现与设计存在以下差异：

| 设计项 | 当前状态 | 说明 |
| --- | --- | --- |
| `match` 可作为语言核心能力使用 | 未完成 | 解析存在缺陷，示例失败 |
| `import/export/go` | 仅有占位 | 解析可进入 AST，执行阶段直接返回 `null` |
| `select/interface/type` | 未实现 | 仅存在 token/帮助文案，无 parser/evaluator 支持 |
| 独立变量声明语句 | 未实现 | AST 中有 `LetStmt`，但 parser 不会产出该节点 |
| 完整错误定位 | 未完成 | 仅有行号，列号未贯通到解释期错误 |
| 标准库/stdlib | 未实现 | 设计文档提到 `stdlib/`，仓库中不存在 |

## 4. 验证结果

### 4.1 编译验证

在 `icooclaw_lang/` 目录执行：

```powershell
go test ./...
go build ./...
go build -o iclang.exe ./cmd/iclang
```

结果：

- 所有包均可编译
- 没有任何 `_test.go` 测试文件
- CLI 二进制可生成

### 4.2 脚本验证

已验证脚本：

| 脚本 | 结果 | 说明 |
| --- | --- | --- |
| `examples/features_test.is` | 通过 | 基础循环、函数、数组、哈希可运行 |
| `examples/simple_test.is` | 通过 | 基础条件分支可运行 |
| `examples/test_try.is` | 通过 | `try/catch` 可捕获运行时错误 |
| `examples/test_match.is` | 失败 | `match` 解析失败并输出调试日志 |
| `examples/hello.is` | 失败 | 因包含 `match` 语法而失败 |
| `examples/test_comprehensive.is` | 卡死 | 单行块语句触发解析器死循环 |

## 5. 关键问题

### 5.1 高风险：单行块语句会导致解析器卡死

相关位置：

- `internal/parser/parser.go:336`
- `internal/parser/parser.go:536`

现象：

- 脚本 `if true { print("x") }` 会一直卡住
- `examples/test_comprehensive.is` 因多处 `if ... { print(...) }` 写法触发超时

原因：

- `parseExpressionStmt()` 在表达式后仅在 `peekToken` 是换行或分号时前进一步
- 当块内最后一个语句与 `}` 同行时，`curToken` 会停留在表达式结尾，如 `)` 或字符串字面量
- `parseBlockStmt()` 的循环退出条件只看 `curToken == RBRACE`，此时不会推进 token，进入死循环

影响：

- 任何允许单行块写法的脚本都可能挂死解释器
- 这属于语法层稳定性问题，优先级应为 P0/P1

### 5.2 高风险：`match` 特性未闭环，且示例与实现矛盾

相关位置：

- `internal/parser/parser.go:186`
- `internal/parser/parser.go:381`
- `internal/parser/parser.go:429`
- `examples/test_match.is`
- `examples/hello.is`

现象：

- `result = match x { ... }` 直接报错：`no prefix parse function for match found`
- `match` 解析过程中会向标准输出打印大量调试信息
- `parseMatchStmt()` / `parseMatchCase()` 在第一个 case 后 token 推进混乱，后续 case 解析失效

原因：

- `match` 只在 `parseStatement()` 中作为语句入口处理，没有注册为 prefix 表达式
- 设计文档和示例同时把 `match` 当表达式使用
- 源码中保留了 `fmt.Printf` 调试输出，说明这部分仍在调试中

影响：

- 设计文档中的代表性特性不可用
- 示例脚本失效会显著降低模块可信度

### 5.3 高风险：常量重赋值错误被吞掉

相关位置：

- `internal/object/environment.go:30`
- `internal/evaluator/evaluator.go:591`
- `internal/evaluator/evaluator.go:633`

现象：

脚本：

```is
const A = 1
A = 2
print(A)
```

实际输出为：

```text
1
```

应有行为：

- 明确报错，提示不能给常量重新赋值

原因：

- `Environment.Set()` 在常量重赋值时返回 `Error`
- `evalAssignExpr()` 和 `evalCompoundAssignExpr()` 调用 `env.Set(...)` 后未检查返回值，直接返回右值或计算结果

影响：

- 语言语义与用户直觉不一致
- 调试成本高，因为用户看不到失败原因

### 5.4 中风险：CLI/REPL 可用性不足

相关位置：

- `cmd/iclang/main.go:18`
- `cmd/iclang/main.go:30`
- `cmd/iclang/main.go:95`

问题：

- `versionFlag` 定义了但没有对默认 flag 做 `Parse()`，`iclang --version` 不会按预期工作
- REPL 通过 `fmt.Scanln(&input)` 读取输入，只能读到一个以空白分隔的 token
- 这意味着带空格的表达式、字符串、复杂语句在 REPL 中不可正常输入

影响：

- CLI 行为与帮助信息不完全一致
- REPL 更接近演示功能，不适合作为真实交互环境

### 5.5 中风险：宣称支持的语言特性多数仍是占位

相关位置：

- `cmd/iclang/main.go:131`
- `internal/evaluator/evaluator.go:56`
- `internal/evaluator/evaluator.go:58`
- `internal/evaluator/evaluator.go:60`
- `internal/lexer/token.go`

问题：

- 帮助信息列出了 `go/select/interface/type/import/export`
- 其中：
  - `import/export/go` 进入 evaluator 后直接返回 `null`
  - `select/interface/type` 没有 parser 分支，也没有 evaluator 分支

影响：

- 用户会认为这些关键字可用，但实际只能触发解析失败或空操作
- 这类“文案先于实现”的不一致会放大误用成本

### 5.6 中低风险：测试体系缺失

现状：

- 没有单元测试
- 没有解析器/解释器回归测试
- 示例脚本承担了事实上的集成测试职责，但没有自动化执行

影响：

- 当前这类 parser 卡死、语义回退、占位特性误发布，都会在没有测试的情况下反复出现

## 6. 优点与可复用基础

尽管问题明显，`icooclaw_lang` 仍然有几项基础值得保留：

- 架构分层清晰，解释器结构标准，后续维护成本可控
- 运行时对象模型较直接，适合继续扩展标准类型
- `Environment` 已支持嵌套作用域和闭包捕获
- 内置函数和方法调用机制已经打通，适合继续做标准库扩展
- `try/catch`、数组/哈希、字符串方法等基础能力已经形成可验证闭环

## 7. 建议优先级

### 第一阶段：先把解释器“跑稳”

1. 修复 `parseBlockStmt()` 与 `parseExpressionStmt()` 的 token 推进逻辑，消除单行块死循环。
2. 明确 `match` 到底是“语句”还是“表达式”，二选一后重写 parser，并删除调试输出。
3. 修复 `evalAssignExpr()` / `evalCompoundAssignExpr()` 对 `env.Set()` 返回错误的处理。
4. 将帮助文案中未实现的关键字先下线，避免误导。

### 第二阶段：补齐语言闭环

1. 如果保留 `import/export/go`，则至少给出真实行为或显式 `not implemented` 错误。
2. 决定是否保留 `select/interface/type`；若短期不做，建议从关键字和帮助文案中移除。
3. 统一“变量声明模型”，决定是继续用隐式赋值创建变量，还是补上显式声明语句。

### 第三阶段：建立回归能力

1. 增加 lexer/parser/evaluator 三层测试。
2. 将 `examples/*.is` 转成自动化脚本回归用例。
3. 针对以下场景单独建测试：
   - 单行块语句
   - `match` 多分支
   - 常量重赋值
   - 复合赋值类型兼容
   - REPL 输入

## 8. 最终判断

`icooclaw_lang` 目前最接近“可继续开发的解释器原型”，而不是“可对外稳定使用的脚本语言”。如果后续目标是快速把它用进 `icooclaw` 主工程，建议先收缩语言面，只保留已经稳定的基础特性；不要在 `match/import/go/select/interface/type` 尚未闭环前扩大对外承诺。

如果后续目标是把它继续做成独立语言，那么优先级不是增加更多关键字，而是先修掉解析器稳定性、语义错误上报和自动化测试这三件事。
