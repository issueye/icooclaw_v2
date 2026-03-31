# icooclaw_lang API 参考文档

版本：`0.1.0`  
最后更新：`2026-03-31`

这份文档面向开发者阅读，目标是帮助你理解和扩展 `icooclaw_lang` 当前已经实现并对外暴露的 API 表面。内容覆盖 `iclang` 的核心语法、内建函数、内建库，以及原生库常见返回对象的结构。

这份文档只描述“当前已经实现并验证过”的能力，不把词法层保留但未真正开放的关键字算进稳定 API。

模块系统说明：

- 只支持 `import`
- 当前支持本地 `.is` 文件模块
- `import "./mod.is"` 会按文件名生成模块命名空间，例如 `math.is -> math`
- `import "./mod.is" as alias` 可显式指定别名
- `import { foo, bar } from "./mod.is"` 会把导出符号直接绑定到当前作用域
- 只有被 `export name` 声明的符号可被导入

## 1. 运行入口

CLI 用法：

```bash
iclang run [--max-goroutines n] <file.is> [args...]
iclang build <file.is> [-o app]
iclang init <dir> [-name demo]
iclang repl [--max-goroutines n]
iclang version
```

打包说明：

- `iclang build demo.is -o demo.exe` 会生成一个单文件可执行程序
- 生成的程序内嵌脚本源码和当前 runtime，不再依赖外部 `.is` 文件
- 打包后的程序直接接收脚本参数，例如 `demo.exe input.txt --mode=prod`
- 打包后的程序也支持 runtime 参数，例如 `demo.exe --max-goroutines 4 input.txt`

REPL 和脚本文件执行共享同一套 parser、evaluator 和 builtin 注册表。

并发度控制：

- 默认读取环境变量 `ICLANG_MAX_GOROUTINES`
- `iclang run --max-goroutines n ...` 会覆盖当前运行时并发度
- `iclang repl --max-goroutines n` 会覆盖当前 REPL 运行时并发度

## 2. 语言表面能力

当前已验证语法与特性：

- 变量：`x = 1`
- 常量：`const PI = 3.14`
- 函数：

```is
fn add(a, b) {
    return a + b
}
```

- 条件分支：

```is
if score > 90 {
    print("A")
} else if score > 60 {
    print("B")
} else {
    print("C")
}
```

- 循环：
  - `while condition { ... }`
  - `for item in range(5) { ... }`
- 控制流：`break`、`continue`、`return`
- 错误处理：

```is
try {
    x = 10 / 0
} catch err {
    print(err)
}
```

- 协程：

```is
go worker("alpha")
```

- `match` 表达式：

```is
result = match payload {
    {"kind": "ok", "value": value} if value > 5 -> "high:" + str(value)
    _ -> "other"
}
```

当前 `match` 支持的模式：

- 字面量匹配
- `_` 通配符
- 数组解构
- 重复变量捕获，例如 `[x, x]`
- 哈希解构
- `if ...` 守卫条件

- 模块导入：

```is
import "./modules/math.is" as math
import "./modules/math.is"
import { add, VERSION } from "./modules/math.is"
```

- 模块导出：

```is
fn add(a, b) {
    return a + b
}

export add
```

## 3. 核心类型

`type(value)` 与 `type_of(value)` 当前可能返回的运行时类型：

- `INTEGER`
- `FLOAT`
- `STRING`
- `BOOLEAN`
- `NULL`
- `ARRAY`
- `HASH`
- `FUNCTION`
- `BUILTIN`
- `ERROR`

真值规则：

- 假值：`false`、`null`、`0`、`0.0`、`""`
- 真值：其它所有值

## 4. 核心内建函数

| 函数 | 签名 | 说明 |
| --- | --- | --- |
| `print` | `print(...args)` | 打印拼接后的文本，返回 `null` |
| `println` | `println(...args)` | 当前行为与 `print` 一致 |
| `len` | `len(value)` | 支持 `STRING`、`ARRAY`、`HASH` |
| `range` | `range(stop)` / `range(start, stop)` | 返回整数数组 |
| `type` | `type(value)` | 返回运行时类型名 |
| `type_of` | `type_of(value)` | `type(value)` 的等价别名 |
| `str` | `str(value)` | 使用运行时 `Inspect()` 字符串化 |
| `int` | `int(value)` | 支持整数、浮点、字符串、布尔值 |
| `float` | `float(value)` | 支持浮点、整数、字符串 |
| `input` | `input()` / `input(prompt)` | 从标准输入读取一行 |
| `push` | `push(array, value)` | 返回新数组 |
| `pop` | `pop(array)` | 返回最后一个元素或 `null` |
| `keys` | `keys(hash)` | 返回键数组 |
| `values` | `values(hash)` | 返回值数组 |
| `abs` | `abs(number)` | 支持整数和浮点数 |
| `read_file` | `read_file(path)` | 旧快捷入口，建议优先用 `fs.read_file` |
| `write_file` | `write_file(path, content)` | 旧快捷入口，建议优先用 `fs.write_file` |

## 5. 内建方法

### 5.0 通用方法

| 方法 | 签名 | 说明 |
| --- | --- | --- |
| `to_string` | `value.to_string()` | 所有运行时对象通用，行为与 `str(value)` 对齐 |

### 5.1 字符串方法

| 方法 | 签名 |
| --- | --- |
| `len` | `"abc".len()` |
| `upper` | `"abc".upper()` |
| `lower` | `"ABC".lower()` |
| `trim` | `"  a ".trim()` |
| `split` | `"a,b".split(",")` |
| `contains` | `"hello".contains("ell")` |
| `starts_with` | `"hello".starts_with("he")` |
| `ends_with` | `"hello".ends_with("lo")` |

### 5.2 数组方法

| 方法 | 签名 |
| --- | --- |
| `len` | `[1,2].len()` |
| `push` | `[1].push(2)` |
| `pop` | `[1,2].pop()` |
| `join` | `["a","b"].join(",")` |
| `contains` | `[1,2,3].contains(2)` |

### 5.3 哈希方法

`HASH` 的方法调用本质上是对哈希字段中的可调用对象进行分发。这也是原生库命名空间对外暴露 API 的方式，例如 `http.client.get(...)`、`db.sqlite.open(...)`。

对于普通对象哈希，如果字段值是函数，也可以直接作为方法调用：

```is
user = {
    "name": "icooclaw",
    "rename": fn(next) {
        this.name = next
        return self.name
    }
}

user.rename("codex")
```

约定：

- 支持匿名函数表达式：`fn(args) { ... }`
- `fn (u user) rename(...) { ... }` 会把方法挂到现有对象 `user` 上
- `obj.method(...)` 调用时会为函数注入 `this`
- `self` 是 `this` 的等价别名
- 如果声明了 receiver 名，例如 `u`，方法体内也可以直接使用 `u`
- `HASH` 现在支持 `obj.field = value` 和 `obj.field += value` 这类点赋值
- 变量和 `HASH` 字段都支持后缀 `++` / `--`
- `ARRAY` / `HASH` 的索引位置也支持 `items[i]++`、`stats["count"] += 1`
- 安全访问支持 `obj?.field` 和 `obj?.method()`，当 `obj == null` 时返回 `null`
- 安全索引支持 `arr?[0]` 和 `obj?["name"]`，当容器为 `null` 时返回 `null`
- 安全更新支持 `obj?.field += 1` 和 `obj?.field++`，当对象为 `null` 时返回 `null`
- 安全索引更新支持 `obj?["count"] += 1`、`obj?["count"]++` 和 `arr?[0]++`
- 连续链式安全访问支持 `user?.profile?.name`、`user?.get_profile()?.name`、`user?.profile?["tag"]`
- 直接把函数取出来再调用时，不会自动保留这个绑定

## 6. 内建库

当前 builtin 根对象：

- `async`
- `db`
- `fs`
- `http`
- `json`
- `toml`
- `yaml`
- `log`
- `time`
- `os`
- `exec`
- `path`
- `crypto`
- `websocket`
- `sse`

### 6.1 async

| API | 签名 | 返回值 |
| --- | --- | --- |
| `async.pool` | `async.pool(size)` | `HASH` |
| `async.wait_group` | `async.wait_group()` | `HASH` |
| `async.runtime_concurrency` | `async.runtime_concurrency()` | `INTEGER` |
| `async.set_runtime_concurrency` | `async.set_runtime_concurrency(size)` | `INTEGER` |

`async.pool(size)` 返回池对象，当前支持：

| API | 签名 | 说明 |
| --- | --- | --- |
| `pool.submit` | `pool.submit(fn)` / `pool.submit(fn, [args])` | 提交任务到受限并发池 |
| `pool.wait` | `pool.wait()` | 等待池内任务全部完成 |
| `pool.size` | `pool.size()` | 返回池容量 |

`async.wait_group()` 返回 waitgroup 对象，当前支持：

| API | 签名 | 说明 |
| --- | --- | --- |
| `wg.add` | `wg.add(n)` | 增加计数器 |
| `wg.done` | `wg.done()` | 计数器减一 |
| `wg.wait` | `wg.wait()` | 等待计数器归零 |
| `wg.count` | `wg.count()` | 返回当前计数 |

示例：

```is
async.set_runtime_concurrency(4)

total = 0
pool = async.pool(2)
wg = async.wait_group()

fn worker(v) {
    total += v
    wg.done()
}

for i in range(1, 5) {
    wg.add(1)
    pool.submit(worker, [i])
}

wg.wait()
pool.wait()
```

说明：

- runtime 默认并发度会读取环境变量 `ICLANG_MAX_GOROUTINES`
- `async.set_runtime_concurrency(n)` 会修改当前 runtime 后续启动 worker 时的默认并发度

### 6.2 fs

| API | 签名 | 返回值 |
| --- | --- | --- |
| `fs.read_file` | `fs.read_file(path)` | `STRING` |
| `fs.write_file` | `fs.write_file(path, content)` | `NULL` |
| `fs.append_file` | `fs.append_file(path, content)` | `NULL` |
| `fs.exists` | `fs.exists(path)` | `BOOLEAN` |
| `fs.mkdir` | `fs.mkdir(path)` | `NULL` |
| `fs.remove` | `fs.remove(path)` | `NULL` |
| `fs.read_dir` | `fs.read_dir(path)` | `ARRAY<HASH>` |
| `fs.stat` | `fs.stat(path)` | `HASH` |
| `fs.abs` | `fs.abs(path)` | `STRING` |

文件信息对象结构：

```is
{
    "name": "demo.txt",
    "path": "demo.txt",
    "size": 12,
    "is_dir": false,
    "mode": "-rw-r--r--"
}
```

### 6.3 json

| API | 签名 | 返回值 |
| --- | --- | --- |
| `json.parse` | `json.parse(text)` | 运行时对象 |
| `json.stringify` | `json.stringify(value)` | `STRING` |
| `json.stringify_pretty` | `json.stringify_pretty(value)` | `STRING` |

### 6.4 yaml

| API | 签名 | 返回值 |
| --- | --- | --- |
| `yaml.parse` | `yaml.parse(text)` | 运行时对象 |
| `yaml.parse_file` | `yaml.parse_file(path)` | 运行时对象 |
| `yaml.stringify` | `yaml.stringify(value)` | `STRING` |

说明：

- YAML 解析结果会转换成运行时 `HASH` / `ARRAY` / 标量值
- `yaml.parse_file` 适合读取项目配置或模板文件
- `yaml.stringify` 适合把运行时对象导出为 YAML 文本

### 6.5 toml

| API | 签名 | 返回值 |
| --- | --- | --- |
| `toml.parse` | `toml.parse(text)` | 运行时对象 |
| `toml.parse_file` | `toml.parse_file(path)` | 运行时对象 |
| `toml.stringify` | `toml.stringify(value)` | `STRING` |

说明：

- TOML 解析结果会转换成运行时 `HASH` / `ARRAY` / 标量值
- `toml.parse_file` 适合读取简单项目清单或配置文件
- `toml.stringify` 当前要求输入是 `HASH`

### 6.6 time

| API | 签名 |
| --- | --- |
| `time.now` | `time.now()` |
| `time.now_unix` | `time.now_unix()` |
| `time.now_unix_ms` | `time.now_unix_ms()` |
| `time.sleep` | `time.sleep(seconds)` |
| `time.sleep_ms` | `time.sleep_ms(ms)` |

`time.now()` 返回对象：

```is
{
    "unix": 1711785600,
    "unix_ms": 1711785600123,
    "rfc_3339": "2026-03-30T20:00:00+08:00",
    "date": "2026-03-30",
    "time": "20:00:00",
    "year": 2026,
    "month": 3,
    "day": 30,
    "hour": 20,
    "minute": 0,
    "second": 0,
    "weekday": "Monday",
    "timestamp": "2026-03-30 20:00:00"
}
```

### 6.7 os

| API | 签名 |
| --- | --- |
| `os.cwd` | `os.cwd()` |
| `os.getenv` | `os.getenv(name)` |
| `os.setenv` | `os.setenv(name, value)` |
| `os.pid` | `os.pid()` |
| `os.hostname` | `os.hostname()` |
| `os.temp_dir` | `os.temp_dir()` |
| `os.args` | `os.args()` |
| `os.arg` | `os.arg(index)` |
| `os.has_flag` | `os.has_flag(name)` |
| `os.flag` | `os.flag(name)` |
| `os.flag_or` | `os.flag_or(name, fallback)` |
| `os.script_path` | `os.script_path()` |

`os.args()` 返回脚本收到的原始命令行参数，不包含 `iclang`、`run` 和脚本文件路径本身。例如：

```bash
iclang run demo.is input.txt --mode=prod --verbose
```

脚本内：

```is
os.args()              # ["input.txt", "--mode=prod", "--verbose"]
os.arg(0)              # "input.txt"
os.flag("mode")        # "prod"
os.has_flag("verbose") # true
os.script_path()       # "demo.is"
```

### 6.8 path

| API | 签名 |
| --- | --- |
| `path.join` | `path.join(part1, part2, ...)` |
| `path.base` | `path.base(path_value)` |
| `path.ext` | `path.ext(path_value)` |
| `path.dir` | `path.dir(path_value)` |
| `path.clean` | `path.clean(path_value)` |

### 6.9 exec

| API | 签名 |
| --- | --- |
| `exec.look_path` | `exec.look_path(name)` |
| `exec.command` | `exec.command(name)` / `exec.command(name, [args])` |
| `exec.command_in` | `exec.command_in(dir, name)` / `exec.command_in(dir, name, [args])` |
| `exec.start` | `exec.start(name)` / `exec.start(name, [args])` |
| `exec.start_in` | `exec.start_in(dir, name)` / `exec.start_in(dir, name, [args])` |

`exec.command(...)` / `exec.command_in(...)` 返回结构：

```is
{
    "ok": true,
    "code": 0,
    "stdout": "go version go1.24.0 windows/amd64\n",
    "stderr": "",
    "output": "go version go1.24.0 windows/amd64\n",
    "command": "go version",
    "dir": "E:\\code\\issueye\\icooclaw_v2"
}
```

约定：

- 参数数组必须是 `ARRAY<STRING>`
- 非零退出码不会抛解释器错误，而是返回 `ok=false` 和实际 `code`
- 如果命令根本无法启动，例如二进制不存在，则返回 `ok=false`、`code=-1`
- `exec.look_path(name)` 找不到时返回 `null`

`exec.start(...)` / `exec.start_in(...)` 返回进程对象，支持：

| API | 签名 |
| --- | --- |
| `proc.read` | `proc.read()` |
| `proc.wait` | `proc.wait()` |
| `proc.kill` | `proc.kill()` |
| `proc.is_running` | `proc.is_running()` |
| `proc.pid` | `proc.pid()` |
| `proc.stdout` | `proc.stdout()` |
| `proc.stderr` | `proc.stderr()` |

`proc.read()` 会阻塞直到读到下一行输出，返回：

```is
{
    "stream": "stdout",
    "text": "Reply from 127.0.0.1: bytes=32 time<1ms TTL=128"
}
```

进程输出示例：

```is
proc = exec.start("ping", ["127.0.0.1", "-n", "4"])
while true {
    line = proc.read()
    if line == null {
        break
    }
    print("[" + line.stream + "] " + line.text)
}
print(proc.wait())
```

### 6.10 crypto

| API | 签名 |
| --- | --- |
| `crypto.md5` | `crypto.md5(text)` |
| `crypto.sha_1` | `crypto.sha_1(text)` |
| `crypto.sha_256` | `crypto.sha_256(text)` |
| `crypto.base_64_encode` | `crypto.base_64_encode(text)` |
| `crypto.base_64_decode` | `crypto.base_64_decode(text)` |

当前 crypto API 都以字符串为输入，并返回字符串。

### 6.11 log

| API | 签名 |
| --- | --- |
| `log.debug` | `log.debug(message...)` 或 `log.debug(fields, message...)` |
| `log.info` | `log.info(message...)` 或 `log.info(fields, message...)` |
| `log.warn` | `log.warn(message...)` 或 `log.warn(fields, message...)` |
| `log.error` | `log.error(message...)` 或 `log.error(fields, message...)` |
| `log.set_level` | `log.set_level("debug" \| "info" \| "warn" \| "error")` |
| `log.level` | `log.level()` |
| `log.set_json` | `log.set_json(true \| false)` |
| `log.is_json` | `log.is_json()` |
| `log.set_output` | `log.set_output("stdout" \| "stderr" \| file_path)` |
| `log.output` | `log.output()` |
| `log.reset` | `log.reset()` |

文本模式日志格式：

```text
2026-03-30T20:00:00+08:00 INFO hello request_id=req-1
```

JSON 模式日志格式：

```json
{"timestamp":"2026-03-30T20:00:00+08:00","level":"INFO","message":"hello","fields":{"request_id":"req-1"}}
```

### 6.12 http

结构：

- `http.client`
- `http.server`

#### http.client

| API | 签名 |
| --- | --- |
| `http.client.get` | `http.client.get(url)` / `http.client.get(url, headers)` |
| `http.client.post` | `http.client.post(url, body)` / `http.client.post(url, body, headers)` |
| `http.client.request` | `http.client.request(method, url)` / `http.client.request(method, url, body_or_null)` / `http.client.request(method, url, body_or_null, headers)` |

响应对象结构：

```is
{
    "status": "200 OK",
    "status_code": 200,
    "body": "hello",
    "headers": {
        "Content-Type": ["text/plain; charset=utf-8"]
    }
}
```

#### http.server

创建方式：

```is
server = http.server.new()
```

路由与服务接口：

| API | 签名 | 说明 |
| --- | --- | --- |
| `server.route` | `server.route(path, body)` / `server.route(method, path, body)` | 静态文本响应 |
| `server.route_json` | `server.route_json(path, value)` / `server.route_json(method, path, value)` | 自动 JSON 响应 |
| `server.route_response` | `server.route_response(path, response_hash)` / `server.route_response(method, path, response_hash)` | 完整响应控制 |
| `server.route_file` | `server.route_file(path, file_path)` / `server.route_file(method, path, file_path)` | 文件响应 |
| `server.handle` | `server.handle(path, handler)` / `server.handle(method, path, handler)` | 动态处理函数 |
| `server.not_found` | `server.not_found(response_hash)` | 自定义 404 |
| `server.start` | `server.start(addr)` | 返回实际绑定地址 |
| `server.stop` | `server.stop()` | 停止服务 |
| `server.addr` | `server.addr()` | 返回绑定地址或 `null` |
| `server.url` | `server.url(path)` | 返回完整 URL |
| `server.is_running` | `server.is_running()` | 布尔值 |
| `server.stats` | `server.stats()` | 服务统计信息 |

handler 签名：

```is
fn handler(req) {
    return {"status_code": 200, "body": "ok"}
}
```

传入 handler 的请求对象：

```is
{
    "method": "GET",
    "path": "/hello",
    "raw_query": "name=icooclaw",
    "query": {"name": "icooclaw"},
    "body": "",
    "headers": {"Accept": ["*/*"]},
    "host": "127.0.0.1:8080"
}
```

handler 返回值规则：

- `null` -> `200` 空文本响应
- `STRING` -> `200` 文本响应
- 含有 `status_code`、`body`、`headers`、`file_path`、`method` 之一的 `HASH` -> 视为 HTTP 响应对象
- 其它运行时值 -> 自动编码成 JSON 响应

响应哈希支持字段：

- `status_code: INTEGER`
- `body: STRING`
- `headers: HASH`
- `file_path: STRING`
- `method: STRING`

服务统计对象：

```is
{
    "addr": "127.0.0.1:53124",
    "is_running": true,
    "route_count": 4,
    "request_count": 12,
    "uptime_ms": 120
}
```

### 6.13 websocket

结构：

- `websocket.client`
- `websocket.server`

#### websocket.client

| API | 签名 |
| --- | --- |
| `websocket.client.connect` | `websocket.client.connect(url)` / `websocket.client.connect(url, headers)` |
| `socket.send` | `socket.send(text)` |
| `socket.send_json` | `socket.send_json(value)` |
| `socket.read` | `socket.read()` |
| `socket.read_message` | `socket.read_message()` |
| `socket.close` | `socket.close()` |
| `socket.is_closed` | `socket.is_closed()` |

`socket.read_message()` 返回对象：

```is
{
    "type": "text",
    "data": "hello"
}
```

#### websocket.server

| API | 签名 |
| --- | --- |
| `websocket.server.new` | `websocket.server.new()` |
| `server.handle` | `server.handle(path, handler)` |
| `server.broadcast` | `server.broadcast(path, text)` |
| `server.broadcast_json` | `server.broadcast_json(path, value)` |
| `server.active_count` | `server.active_count(path)` |
| `server.start` | `server.start(addr)` |
| `server.stop` | `server.stop()` |
| `server.addr` | `server.addr()` |
| `server.url` | `server.url(path)` |
| `server.is_running` | `server.is_running()` |
| `server.stats` | `server.stats()` |

handler 签名：

```is
fn ws_handler(req, socket) {
    text = socket.read()
    socket.send("echo:" + text)
}
```

统计对象：

```is
{
    "addr": "127.0.0.1:54000",
    "is_running": true,
    "handler_count": 1,
    "request_count": 2,
    "connection_count": 2,
    "active_count": 1,
    "uptime_ms": 80
}
```

### 6.14 sse

结构：

- `sse.client`
- `sse.server`

#### sse.client

| API | 签名 |
| --- | --- |
| `sse.client.connect` | `sse.client.connect(url)` / `sse.client.connect(url, headers)` |
| `client.read` | `client.read()` |
| `client.close` | `client.close()` |
| `client.is_closed` | `client.is_closed()` |

`client.read()` 返回对象：

```is
{
    "event": "message",
    "data": "hello",
    "id": "evt-1",
    "retry": "1500"
}
```

#### sse.server

| API | 签名 |
| --- | --- |
| `sse.server.new` | `sse.server.new()` |
| `server.handle` | `server.handle(path, handler)` |
| `server.start` | `server.start(addr)` |
| `server.stop` | `server.stop()` |
| `server.addr` | `server.addr()` |
| `server.url` | `server.url(path)` |
| `server.is_running` | `server.is_running()` |
| `server.stats` | `server.stats()` |

stream 对象方法：

| API | 签名 |
| --- | --- |
| `stream.send` | `stream.send(data)` |
| `stream.send_event` | `stream.send_event(event_name, data)` |
| `stream.send_with_id` | `stream.send_with_id(data, id)` |
| `stream.send_event_with_id` | `stream.send_event_with_id(event_name, data, id)` |
| `stream.set_retry` | `stream.set_retry(ms)` |
| `stream.send_json` | `stream.send_json(value)` |
| `stream.close` | `stream.close()` |
| `stream.is_closed` | `stream.is_closed()` |

handler 签名：

```is
fn events(req, stream) {
    stream.set_retry(1500)
    stream.send_with_id("hello", "evt-1")
}
```

`sse.server.stats()` 的结构与 `websocket.server.stats()` 相同。

### 6.15 db

结构：

- `db.sqlite`
- `db.mysql`
- `db.pg`

连接创建：

| API | 签名 |
| --- | --- |
| `db.sqlite.open` | `db.sqlite.open(path)` |
| `db.mysql.open` | `db.mysql.open(dsn)` |
| `db.pg.open` | `db.pg.open(dsn)` |

连接对象方法：

| API | 签名 |
| --- | --- |
| `conn.driver` | `conn.driver()` |
| `conn.begin` | `conn.begin()` |
| `conn.prepare` | `conn.prepare(sql)` |
| `conn.exec` | `conn.exec(sql)` / `conn.exec(sql, [params])` |
| `conn.query` | `conn.query(sql)` / `conn.query(sql, [params])` |
| `conn.query_one` | `conn.query_one(sql)` / `conn.query_one(sql, [params])` |
| `conn.ping` | `conn.ping()` |
| `conn.stats` | `conn.stats()` |
| `conn.close` | `conn.close()` |

事务对象方法：

| API | 签名 |
| --- | --- |
| `tx.driver` | `tx.driver()` |
| `tx.prepare` | `tx.prepare(sql)` |
| `tx.exec` | `tx.exec(sql)` / `tx.exec(sql, [params])` |
| `tx.query` | `tx.query(sql)` / `tx.query(sql, [params])` |
| `tx.query_one` | `tx.query_one(sql)` / `tx.query_one(sql, [params])` |
| `tx.commit` | `tx.commit()` |
| `tx.rollback` | `tx.rollback()` |
| `tx.is_closed` | `tx.is_closed()` |

预处理语句对象方法：

| API | 签名 |
| --- | --- |
| `stmt.driver` | `stmt.driver()` |
| `stmt.sql` | `stmt.sql()` |
| `stmt.exec` | `stmt.exec()` / `stmt.exec([params])` |
| `stmt.query` | `stmt.query()` / `stmt.query([params])` |
| `stmt.query_one` | `stmt.query_one()` / `stmt.query_one([params])` |
| `stmt.close` | `stmt.close()` |
| `stmt.is_closed` | `stmt.is_closed()` |

`exec` 返回对象：

```is
{
    "rows_affected": 1,
    "last_insert_id": 3
}
```

查询返回规则：

- `query(...)` -> `ARRAY<HASH>`
- `query_one(...)` -> `HASH` 或 `null`

行对象示例：

```is
{
    "id": 1,
    "name": "alice",
    "age": 20
}
```

连接统计对象：

```is
{
    "driver": "sqlite",
    "open_connections": 1,
    "in_use": 0,
    "idle": 1,
    "wait_count": 0,
    "max_idle_closed": 0,
    "max_lifetime_closed": 0
}
```

SQL 参数转换规则：

- `STRING` -> SQL 字符串
- `INTEGER` -> 整数
- `FLOAT` -> 浮点
- `BOOLEAN` -> 布尔值
- `NULL` -> SQL `NULL`
- `ARRAY` / `HASH` -> JSON 字符串

## 7. 实践建议

- 多词 API 统一使用下划线命名。
- `go` 会启动异步任务；运行时会在进程退出前等待被跟踪的协程完成。
- 对于 HTTP、WebSocket、SSE 这类长期资源，脚本结束前应显式调用 `server.stop()`。
- 对数据库连接和预处理语句，长生命周期脚本里应主动 `close()`。
- `query` 和 `exec` 的参数绑定使用数组，不是可变参数风格。
- 动态 HTTP handler 最稳妥的返回方式是 `{"status_code": ..., "body": ..., "headers": ...}`。
- 本文列出的能力可以视为当前稳定表面；只在 lexer 中保留但未在这里说明的 token，不应当视为对外稳定 API。
