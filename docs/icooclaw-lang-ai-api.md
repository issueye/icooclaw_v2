# icooclaw_lang AI API 指南

版本：`0.1.0`  
最后更新：`2026-03-31`

这份文档面向 AI 代理、代码生成器和自动化系统，目标是帮助模型稳定生成可运行的 `icooclaw_lang` 脚本。内容重点不是底层实现，而是“应如何生成代码”。

## 1. 生成规则

除非用户明确要求其它风格，否则建议遵循下面这些规则：

1. 多词 API 一律使用下划线命名。
2. 只使用本文档中已经列出的 API。
3. 当某个参数“概念上可省略”但会让调用形式变模糊时，显式传 `null`。
4. SQL 参数一律用数组：`conn.query(sql, [arg1, arg2])`。
5. 预处理语句要么不传参数，要么只传一个数组：`stmt.exec()` 或 `stmt.exec([arg1])`。
6. 所有长生命周期资源都要主动关闭：`server.stop()`、`socket.close()`、`client.close()`、`conn.close()`、`stmt.close()`。
7. 重复执行 SQL 时，优先使用 `prepare`。
8. HTTP handler 返回值优先使用字符串、响应哈希、`null`，或可 JSON 编码的对象。
9. 不要自行发明未记录的 API，也不要假设存在 camelCase 别名。
10. 对于文档未说明的 lexer 保留关键字，一律视为不可用。

运行与打包入口：

```bash
iclang run [--max-goroutines n] demo.is input.txt --mode=prod
iclang build demo.is -o demo.exe
iclang init demo-app -name demo_app
iclang repl [--max-goroutines n]
demo.exe [--max-goroutines n] input.txt --mode=prod
```

## 2. 规范运行时表面

### 2.1 核心函数

```is
print(...args)
println(...args)
len(value)
range(stop)
range(start, stop)
type(value)
type_of(value)
str(value)
int(value)
float(value)
input()
input(prompt)
push(array, value)
pop(array)
keys(hash)
values(hash)
abs(number)
```

### 2.2 核心方法

所有运行时对象都支持：

```is
value.to_string()
```

字符串方法：

```is
text.len()
text.upper()
text.lower()
text.trim()
text.split(sep)
text.contains(sub)
text.starts_with(prefix)
text.ends_with(suffix)
```

数组方法：

```is
arr.len()
arr.push(value)
arr.pop()
arr.join(sep)
arr.contains(value)
```

对象方法约定：

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

- 支持匿名函数表达式：`fn(args) { ... }`
- 当哈希字段是函数时，可以使用 `obj.method(...)`
- 方法体内可使用 `this`，`self` 为等价别名
- `fn (u user) rename(...)` 会把方法挂到现有对象 `user`
- 方法体内也可以直接使用 receiver 名 `u`
- `HASH` 支持 `obj.field = value` 和 `obj.field += value`
- 变量和 `HASH` 字段支持 `++` / `--`
- `ARRAY` / `HASH` 的索引位置支持 `items[i]++`、`stats["count"] += 1`

## 3. 推荐语法模板

### 3.1 函数

```is
fn add(a, b) {
    return a + b
}
```

### 3.2 if / else if / else

```is
if n > 10 {
    print("high")
} else if n > 5 {
    print("mid")
} else {
    print("low")
}
```

### 3.3 循环

```is
for i in range(5) {
    print(i)
}

while flag {
    break
}
```

### 3.4 try/catch

```is
try {
    x = 1 / 0
} catch err {
    print(err)
}
```

### 3.5 go

```is
fn worker(name) {
    print(name)
}

go worker("alpha")
```

### 3.6 match 表达式

```is
result = match payload {
    {"kind": "ok", "value": value} if value > 5 -> "high:" + str(value)
    {"kind": "ok", "value": value} -> "low:" + str(value)
    _ -> "other"
}
```

### 3.7 import 模块

```is
import "./modules/math.is" as math
import { add, VERSION } from "./modules/math.is"
```

模块中导出已有符号：

```is
fn add(a, b) {
    return a + b
}

const VERSION = "1.0.0"

export add
export VERSION
```

## 4. 内建库总览

当前 builtin 根对象：

```is
async
exec
fs
json
toml
yaml
time
os
path
crypto
log
http
websocket
sse
db
```

## 5. API 签名速查

### 5.1 async

```is
async.pool(size)
async.wait_group()
async.runtime_concurrency()
async.set_runtime_concurrency(size)
```

`async.pool(size)` 返回池对象：

```is
pool.submit(fn)
pool.submit(fn, [args])
pool.wait()
pool.size()
```

`async.wait_group()` 返回 waitgroup 对象：

```is
wg.add(n)
wg.done()
wg.wait()
wg.count()
```

运行时并发度约定：

- 默认读取环境变量 `ICLANG_MAX_GOROUTINES`
- `iclang run --max-goroutines n ...` 可覆盖当前运行
- `demo.exe --max-goroutines n ...` 可覆盖打包后的程序运行

### 5.2 fs

```is
fs.read_file(path)
fs.write_file(path, content)
fs.append_file(path, content)
fs.exists(path)
fs.mkdir(path)
fs.remove(path)
fs.read_dir(path)
fs.stat(path)
fs.abs(path)
```

### 5.3 json

```is
json.parse(text)
json.stringify(value)
json.stringify_pretty(value)
```

### 5.4 toml

```is
toml.parse(text)
toml.parse_file(path)
toml.stringify(value)
```

### 5.5 yaml

```is
yaml.parse(text)
yaml.parse_file(path)
yaml.stringify(value)
```

### 5.6 time

```is
time.now()
time.now_unix()
time.now_unix_ms()
time.sleep(seconds)
time.sleep_ms(ms)
```

`time.now()` 返回结构：

```is
{
    "unix": INTEGER,
    "unix_ms": INTEGER,
    "rfc_3339": STRING,
    "date": STRING,
    "time": STRING,
    "year": INTEGER,
    "month": INTEGER,
    "day": INTEGER,
    "hour": INTEGER,
    "minute": INTEGER,
    "second": INTEGER,
    "weekday": STRING,
    "timestamp": STRING
}
```

### 5.7 os

```is
os.cwd()
os.getenv(name)
os.setenv(name, value)
os.pid()
os.hostname()
os.temp_dir()
os.args()
os.arg(index)
os.has_flag(name)
os.flag(name)
os.flag_or(name, fallback)
os.script_path()

exec.look_path(name)
exec.command(name)
exec.command(name, [args])
exec.command_in(dir, name)
exec.command_in(dir, name, [args])
exec.start(name)
exec.start(name, [args])
exec.start_in(dir, name)
exec.start_in(dir, name, [args])
```

### 5.8 path

```is
path.join(part1, part2, ...)
path.base(path_value)
path.ext(path_value)
path.dir(path_value)
path.clean(path_value)
```

### 5.9 crypto

```is
crypto.md5(text)
crypto.sha_1(text)
crypto.sha_256(text)
crypto.base_64_encode(text)
crypto.base_64_decode(text)
```

### 5.10 log

```is
log.debug(message...)
log.debug(fields, message...)
log.info(message...)
log.info(fields, message...)
log.warn(message...)
log.warn(fields, message...)
log.error(message...)
log.error(fields, message...)
log.set_level("debug" | "info" | "warn" | "error")
log.level()
log.set_json(true | false)
log.is_json()
log.set_output("stdout" | "stderr" | file_path)
log.output()
log.reset()
```

当需要结构化字段时，第一个参数必须是哈希：

```is
log.info({"request_id": "req-1", "code": 200}, "request complete")
```

### 5.11 http

客户端：

```is
http.client.get(url)
http.client.get(url, headers)
http.client.post(url, body)
http.client.post(url, body, headers)
http.client.request(method, url)
http.client.request(method, url, body_or_null)
http.client.request(method, url, body_or_null, headers)
```

客户端响应对象：

```is
{
    "status": STRING,
    "status_code": INTEGER,
    "body": STRING,
    "headers": HASH<STRING, ARRAY<STRING>>
}
```

服务端：

```is
server = http.server.new()
server.route(path, body)
server.route(method, path, body)
server.route_json(path, value)
server.route_json(method, path, value)
server.route_response(path, response_hash)
server.route_response(method, path, response_hash)
server.route_file(path, file_path)
server.route_file(method, path, file_path)
server.handle(path, handler)
server.handle(method, path, handler)
server.not_found(response_hash)
server.start(addr)
server.stop()
server.addr()
server.url(path)
server.is_running()
server.stats()
```

HTTP handler 签名：

```is
fn handler(req) {
    return {"status_code": 200, "body": "ok"}
}
```

HTTP 请求对象结构：

```is
{
    "method": STRING,
    "path": STRING,
    "raw_query": STRING,
    "query": HASH,
    "body": STRING,
    "headers": HASH<STRING, ARRAY<STRING>>,
    "host": STRING
}
```

HTTP 响应哈希支持键：

```is
{
    "status_code": INTEGER,
    "body": STRING,
    "headers": HASH,
    "file_path": STRING,
    "method": STRING
}
```

handler 返回规则：

- `null` -> `200` 空 body
- `STRING` -> `200 text/plain`
- 响应哈希 -> 直接作为响应
- 其它对象 -> 自动转 JSON

### 5.12 websocket

客户端：

```is
websocket.client.connect(url)
websocket.client.connect(url, headers)
socket.send(text)
socket.send_json(value)
socket.read()
socket.read_message()
socket.close()
socket.is_closed()
```

`socket.read_message()` 返回结构：

```is
{
    "type": "text" | "binary" | "close" | "ping" | "pong" | "unknown",
    "data": STRING
}
```

服务端：

```is
server = websocket.server.new()
server.handle(path, handler)
server.broadcast(path, text)
server.broadcast_json(path, value)
server.active_count(path)
server.start(addr)
server.stop()
server.addr()
server.url(path)
server.is_running()
server.stats()
```

WebSocket handler 签名：

```is
fn ws_handler(req, socket) {
    text = socket.read()
    socket.send("echo:" + text)
}
```

### 5.13 sse

客户端：

```is
sse.client.connect(url)
sse.client.connect(url, headers)
client.read()
client.close()
client.is_closed()
```

`client.read()` 返回结构：

```is
{
    "event": STRING,
    "data": STRING,
    "id": STRING,
    "retry": STRING
}
```

服务端：

```is
server = sse.server.new()
server.handle(path, handler)
server.start(addr)
server.stop()
server.addr()
server.url(path)
server.is_running()
server.stats()
```

stream 对象：

```is
stream.send(data)
stream.send_event(event_name, data)
stream.send_with_id(data, id)
stream.send_event_with_id(event_name, data, id)
stream.set_retry(ms)
stream.send_json(value)
stream.close()
stream.is_closed()
```

SSE handler 签名：

```is
fn events(req, stream) {
    stream.set_retry(1500)
    stream.send_with_id("hello", "evt-1")
}
```

### 5.14 db

打开连接：

```is
db.sqlite.open(path)
db.mysql.open(dsn)
db.pg.open(dsn)
```

连接对象：

```is
conn.driver()
conn.begin()
conn.prepare(sql)
conn.exec(sql)
conn.exec(sql, [params])
conn.query(sql)
conn.query(sql, [params])
conn.query_one(sql)
conn.query_one(sql, [params])
conn.ping()
conn.stats()
conn.close()
```

事务对象：

```is
tx.driver()
tx.prepare(sql)
tx.exec(sql)
tx.exec(sql, [params])
tx.query(sql)
tx.query(sql, [params])
tx.query_one(sql)
tx.query_one(sql, [params])
tx.commit()
tx.rollback()
tx.is_closed()
```

预处理语句对象：

```is
stmt.driver()
stmt.sql()
stmt.exec()
stmt.exec([params])
stmt.query()
stmt.query([params])
stmt.query_one()
stmt.query_one([params])
stmt.close()
stmt.is_closed()
```

`exec` 返回结构：

```is
{
    "rows_affected": INTEGER,
    "last_insert_id": INTEGER
}
```

查询返回规则：

- `query(...)` -> `ARRAY<HASH>`
- `query_one(...)` -> `HASH | null`

数据库统计对象：

```is
{
    "driver": STRING,
    "open_connections": INTEGER,
    "in_use": INTEGER,
    "idle": INTEGER,
    "wait_count": INTEGER,
    "max_idle_closed": INTEGER,
    "max_lifetime_closed": INTEGER
}
```

参数转换规则：

- `STRING` -> SQL 字符串
- `INTEGER` -> 整数
- `FLOAT` -> 浮点
- `BOOLEAN` -> 布尔值
- `NULL` -> SQL null
- `ARRAY` / `HASH` -> JSON 字符串

## 6. 安全生成模板

### 6.1 最小 HTTP 服务

```is
server = http.server.new()
server.route_json("/health", {"ok": true})
server.start("127.0.0.1:0")
print(server.url("/health"))
server.stop()
```

### 6.2 带完整响应控制的 HTTP handler

```is
fn greet(req) {
    return {
        "status_code": 200,
        "headers": {"Content-Type": "text/plain; charset=utf-8"},
        "body": "hello " + req.query.name
    }
}

server = http.server.new()
server.handle("/greet", greet)
server.start("127.0.0.1:0")
print(http.client.get(server.url("/greet?name=ai")).body)
server.stop()
```

### 6.3 SQLite 预处理语句

```is
conn = db.sqlite.open("demo.db")
conn.exec("create table users (id integer primary key autoincrement, name text)")

stmt = conn.prepare("insert into users (name) values (?)")
stmt.exec(["alice"])
stmt.exec(["bob"])
stmt.close()

print(conn.query("select id, name from users order by id"))
conn.close()
```

### 6.4 SSE 推送

```is
fn events(req, stream) {
    stream.set_retry(1500)
    stream.send_event_with_id("update", "done", "evt-1")
}
```

### 6.5 并发池与 waitgroup

```is
async.set_runtime_concurrency(4)

pool = async.pool(2)
wg = async.wait_group()
total = 0

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
print(total.to_string())
```

## 7. 不要这样生成

- 不要生成 camelCase API，例如 `readFile`、`stringifyPretty`、`setLevel`。
- 不要假设 SQL 参数支持可变参数展开。
- 不要在脚本结束前遗留 HTTP/WebSocket/SSE 服务不关闭。
- 不要假设存在本文档未写出的原生库。
- 不要假设 `query` 返回结构体；它返回的是哈希数组。
