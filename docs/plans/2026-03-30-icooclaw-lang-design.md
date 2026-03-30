# Icooclaw Script Language (icooclaw_lang) 开发计划

**版本**: v1.0  
**日期**: 2026-03-30  
**项目路径**: `icooclaw_lang/`

---

## 1. 项目概述

**目标**: 使用 Golang 创建一门类似 Python 的独立脚本语言

**文件扩展名**: `.is` (icooclaw script)

**核心特性**:
- 变量、函数、循环、条件判断
- 列表、字典、字符串处理
- 基础标准库

**优先级**: 开发效率优先，快速原型开发

---

## 2. 关键字设计

共 23 个关键字：

```
fn        # 函数定义
return    # 返回值
if        # 条件
else      # 条件分支
for       # 循环
while     # 循环
match     # 模式匹配
break     # 跳出循环
continue  # 继续循环
const     # 常量声明
import    # 导入模块
export    # 导出
try       # 异常捕获
catch     # 异常捕获
go        # 协程启动
select    # 通道选择
interface # 接口定义
type      # 类型定义
null      # 空值
true      # 布尔真
false     # 布尔假
in        # 成员测试
_         # 占位符/忽略值
```

**逻辑操作符用符号替代**:
```
&&      # AND
||      # OR
!       # NOT
```

---

## 3. 完整语法规则 (EBNF)

```ebnf
program        = stmt*

stmt           = expr_stmt | if_stmt | for_stmt | while_stmt 
               | func_def | return_stmt | break_stmt | continue_stmt
               | match_stmt | try_stmt | const_stmt | import_stmt
               | go_stmt | type_def | interface_def

expr_stmt      = expr NEWLINE

if_stmt        = IF expr LBRACE stmt* RBRACE (ELSE LBRACE stmt* RBRACE)?
for_stmt       = FOR IDENTIFIER IN expr LBRACE stmt* RBRACE
while_stmt     = WHILE expr LBRACE stmt* RBRACE
func_def       = FN IDENTIFIER LPAREN params? RPAREN LBRACE stmt* RBRACE
return_stmt    = RETURN expr? NEWLINE
break_stmt     = BREAK NEWLINE
continue_stmt = CONTINUE NEWLINE

match_stmt     = MATCH expr LBRACE (case_stmt)* RBRACE
case_stmt      = case_pattern ARROW expr (COMMA expr)* NEWLINE
case_pattern   = pattern ("|" pattern)*
pattern        = INTEGER | STRING | IDENTIFIER | UNDERSCORE | NULL

try_stmt       = TRY LBRACE stmt* RBRACE CATCH IDENTIFIER LBRACE stmt* RBRACE
const_stmt     = CONST IDENTIFIER "=" expr NEWLINE
import_stmt    = IMPORT STRING (AS IDENTIFIER)? NEWLINE
export_stmt    = EXPORT IDENTIFIER NEWLINE

go_stmt        = GO expr NEWLINE

type_def       = TYPE IDENTIFIER "=" type_expr NEWLINE
interface_def  = INTERFACE IDENTIFIER LBRACE (IDENTIFIER LPAREN types? RPAREN)* RBRACE

params         = IDENTIFIER (COMMA IDENTIFIER)*
args           = expr (COMMA expr)*
types          = type_expr (COMMA type_expr)*
type_expr      = IDENTIFIER | "int" | "float" | "string" | "bool"

expr           = assignment
assignment     = IDENTIFIER "=" assignment | logic_or
logic_or       = logic_and (OR logic_and)*
logic_and      = equality (AND equality)*
equality       = comparison (("==" | "!=") comparison)*
comparison     = term (("<" | ">" | "<=" | ">=") term)*
term           = factor (("+" | "-") factor)*
factor         = unary (("*" | "/" | "%") unary)*
unary          = ("!" | "-") unary | primary
primary        = INTEGER | FLOAT | STRING | TRUE | FALSE | NULL
               | IDENTIFIER | IDENTIFIER "(" args ")" 
               | "(" expr ")" | list | dict | "(" type_expr ")"
list           = "[" (expr ("," expr)*)? "]"
dict           = "{" (pair ("," pair)*)? "}"
pair           = expr ":" expr
```

---

## 4. Token 类型设计

```
// 基础类型
IDENTIFIER, INTEGER, FLOAT, STRING

// 关键字 (23个)
FN, RETURN, IF, ELSE, FOR, WHILE, MATCH,
BREAK, CONTINUE, CONST, IMPORT, EXPORT,
TRY, CATCH, GO, SELECT, INTERFACE, TYPE,
NULL, TRUE, FALSE, IN, UNDERSCORE

// 操作符
PLUS, MINUS, STAR, SLASH, PERCENT, ASSIGN,
EQ, NE, LT, LE, GT, GE,
AND, OR, NOT

// 分隔符
LPAREN, RPAREN, LBRACE, RBRACE, LBRACKET, RBRACKET,
COMMA, COLON, DOT, SEMICOLON, ARROW,
NEWLINE, EOF
```

---

## 5. 数据类型系统

```go
// 对象类型
Integer  { value int64 }
Float    { value float64 }
String   { value string }
Boolean  { value bool }
Null     {}

// 容器类型
Array    { elements []Object }
Hash     { pairs   map[string]Object }

// 函数类型
Function { 
    name   string
    params []string
    body   []ast.Stmt
    env    *Environment
}

Builtin  { fn builtinFunction }
```

---

## 6. 标准库函数

```python
print(...)              # 打印
len(obj)                # 长度
range(start, stop)      # 范围数组
type(obj)               # 类型名
str(obj)                # 转字符串
int(obj)                # 转整数
float(obj)              # 转浮点数
input(prompt)           # 用户输入
read_file(path)         # 读文件
write_file(path, content) # 写文件
```

---

## 7. 项目结构

```
icooclaw_lang/
├── cmd/
│   └── iclang/
│       └── main.go           # CLI 入口
├── internal/
│   ├── lexer/
│   │   ├── lexer.go          # 词法分析器
│   │   └── token.go         # Token 定义
│   ├── parser/
│   │   └── parser.go         # 语法分析器
│   ├── ast/
│   │   └── ast.go            # AST 节点
│   ├── evaluator/
│   │   ├── evaluator.go      # 解释器
│   │   └── environment.go    # 环境/作用域
│   ├── object/
│   │   └── object.go         # 对象类型系统
│   └── builtin/
│       └── builtin.go        # 内置函数
├── stdlib/
│   └── stdlib.is             # 标准库脚本
├── examples/
│   └── hello.is              # 示例脚本
├── go.mod
└── README.md
```

---

## 8. 执行流程

```
1. 解析命令行参数 (iclang run hello.is)
2. 读取源文件
3. 词法分析 → Token 列表
4. 语法分析 → AST
5. 创建全局环境
6. 遍历 AST 执行
7. 输出结果
```

---

## 9. 错误处理

```go
type Error struct {
    Message string
    Line    int
    Column  int
}
```

**示例错误信息**:
```
Error: unexpected token at line 5, column 10
  |
5 | if x = 1 {
  |          ^
  | expect '=='
```

---

## 10. 开发里程碑

| 阶段 | 周数 | 目标 |
|------|------|------|
| **阶段一：核心解释器** | Week 1-4 | 词法分析、语法分析、AST构建、基础表达式求值、变量、函数 |
| **阶段二：控制流** | Week 5-8 | IF/ELSE、FOR/WHILE、BREAK/CONT、RETURN、MATCH、TRY/CATCH |
| **阶段三：数据类型与标准库** | Week 9-12 | 数组、字典、内置函数、标准库、CONST、TYPE、INTERFACE |
| **阶段四：并发与模块** | Week 13-14 | GO/SELECT、IMPORT/EXPORT 模块系统 |
| **阶段五：完善与测试** | Week 15-16 | 单元测试、集成测试、文档、发布 v1.0.0 |

---

## 11. 技术选型

| 组件 | 方案 | 理由 |
|------|------|------|
| 解析器 | 手写递归下降 | 比 gocc 更灵活，代码量可控 |
| 虚拟机 | 直接 AST 遍历 | 简单够用，后续可优化为字节码 |
| 内存管理 | 手动管理 + 垃圾回收 | Go GC 足够 |
| 包管理 | Go modules | 官方标准 |

---

## 12. 语法示例

```python
# 变量
name = "world"
count = 42

# 常量
const PI = 3.14159

# 函数
fn greet(name) {
    print("Hello, " + name)
}

# 条件
if count > 10 {
    print("large")
} else {
    print("small")
}

# 循环
sum = 0
for i in range(10) {
    sum = sum + i
}

# 模式匹配
result = match x {
    1 | 2 -> "one or two"
    3 -> "three"
    _ -> "other"
}

# 异常处理
try {
    dangerous()
} catch err {
    print("Error: " + err)
}

# 列表
numbers = [1, 2, 3, 4, 5]
for n in numbers {
    print(n)
}

# 字典
person = {"name": "Alice", "age": 30}
print(person["name"])

# 类型定义
type Point = {x: int, y: int}

# 接口
interface Drawable {
    draw()
    getArea() -> float
}

# 导入模块
import "io.is" as io
io.writeFile("test.txt", "hello")

# 并发
go processTask(data)
```
