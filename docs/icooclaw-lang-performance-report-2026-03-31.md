# icooclaw_lang 性能测试与分析报告

日期：`2026-03-31`  
目录：`icooclaw_lang/`  
平台：`Windows amd64`  
CPU：`Intel(R) Core(TM) i5-10500 CPU @ 3.10GHz`

## 1. 测试范围

本次性能测试覆盖以下路径：

- `lexer` 词法扫描
- `parser` 标准脚本解析
- `parser` 大脚本解析
- `evaluator` 纯计算执行
- `evaluator` 函数调用密集执行
- `evaluator` JSON 编解码路径
- `evaluator` 模块导入冷路径
- `evaluator` 模块导入热路径

执行命令：

```powershell
go test ./internal/parser ./internal/evaluator -run ^$ -bench . -benchmem -count 3
```

测试时将 `GOCACHE` / `GOMODCACHE` 指向项目目录内缓存，以避免宿主默认缓存目录权限限制影响结果。

## 2. 结果摘要

### 2.1 Parser / Lexer

| Benchmark | 典型耗时 | 吞吐 | 内存 | 分配次数 |
| --- | ---: | ---: | ---: | ---: |
| `BenchmarkLexProgram` | `8.3-9.2 us/op` | `40-44 MB/s` | `16096 B/op` | `93 allocs/op` |
| `BenchmarkParseProgram` | `16.0-17.3 us/op` | `21-23 MB/s` | `11680 B/op` | `248 allocs/op` |
| `BenchmarkParseLargeProgram` | `43.1-53.4 us/op` | `20-26 MB/s` | `28920 B/op` | `642 allocs/op` |

结论：

- 词法扫描明显快于语法解析。
- `parser` 吞吐量基本稳定在 `20+ MB/s`，说明解析器主路径没有明显退化。
- 大脚本解析性能近似线性增长，没有出现阶跃型恶化。

### 2.2 Evaluator

| Benchmark | 典型耗时 | 吞吐 | 内存 | 分配次数 |
| --- | ---: | ---: | ---: | ---: |
| `BenchmarkEvalProgram` | `574-578 us/op` | `0.42-0.43 MB/s` | `725952-725968 B/op` | `10053 allocs/op` |
| `BenchmarkEvalFunctionCalls` | `542-545 us/op` | `0.16-0.17 MB/s` | `719606-719607 B/op` | `10523 allocs/op` |
| `BenchmarkEvalJSONRoundTrip` | `7.7-8.0 us/op` | `25-26 MB/s` | `6004 B/op` | `110 allocs/op` |

结论：

- 当前主要性能成本集中在 `evaluator`，不是 `lexer/parser`。
- 纯计算和函数调用密集脚本都接近 `0.55 ms/op`，且分配量接近 `700 KB/op`，说明瓶颈更偏向运行时对象创建、作用域构造和递归/循环中的临时值，而不是某一个特定语法。
- `json.stringify/json.parse` 路径相对高效，量级约 `8 us/op`，远低于解释执行主路径。

### 2.3 模块系统

| Benchmark | 典型耗时 | 吞吐 | 内存 | 分配次数 |
| --- | ---: | ---: | ---: | ---: |
| `BenchmarkEvalModuleImportCold` | `55.7-57.0 us/op` | `0.32 MB/s` | `10136 B/op` | `135 allocs/op` |
| `BenchmarkEvalModuleImportWarm` | `3.2-3.4 us/op` | `5.3-5.6 MB/s` | `2552 B/op` | `33 allocs/op` |

结论：

- 模块缓存生效明显。
- 热路径比冷路径大约快 `16-17x`，分配次数下降到约 `1/4`。
- 说明模块系统的主要成本在首轮文件读取、词法/语法解析和导出表构造，而不是缓存命中后的命名空间绑定。

## 3. 关键分析

### 3.1 当前瓶颈不在 Parser

从数量级看：

- `lexer`：`~9 us`
- `parser`：`~16 us`
- `evaluator` 主路径：`~575 us`

这意味着，即使把 parser 再优化 `30%-50%`，对完整脚本执行的总收益也有限。当前真正值得投入的是 evaluator 层。

### 3.2 Allocations 是最明显的问题

`BenchmarkEvalProgram` 与 `BenchmarkEvalFunctionCalls` 都显示：

- `700 KB+ / op`
- `10000+ allocs / op`

这通常意味着：

- 循环体反复创建新的 `Environment`
- 大量临时 `Object` 包装对象逃逸到堆
- 数组、哈希和字符串拼接路径存在重复分配
- 函数调用和 block 求值过程中存在频繁短生命周期对象创建

和 parser 的 `248 allocs/op` 相比，evaluator 的分配压力高两个数量级以上。

### 3.3 模块系统实现是健康的

模块导入热路径 `~3.3 us/op`，这说明：

- 缓存命中后，导入成本已经足够低
- 目前模块系统不是性能风险点
- 真正需要关注的是“模块内脚本执行逻辑”，而不是 `import` 语句本身

## 4. 优化优先级建议

### P1：降低 evaluator 中的作用域和对象分配

优先检查：

- `for/while` 中是否每轮都创建新的 `Environment`
- `callFunction` 是否可以减少临时环境和中间对象
- `match`、数组、哈希求值中是否存在重复包装

这是最可能带来整体收益的区域。

### P2：复用常见不可变对象

优先考虑：

- `true/false/null` 单例化
- 常用错误路径减少格式化和分配
- 高频小整数或短字符串场景的优化策略

这类优化通常能显著降低 `allocs/op`。

### P3：减少字符串与集合中间值

优先检查：

- `str(...)`
- `Inspect()`
- 数组 `join`
- 哈希/数组构造时的中间对象

这些在脚本语言里很容易变成隐藏成本。

### P4：Parser 只做守住线性复杂度

目前 parser 性能已经够用。短期目标不是“更快”，而是：

- 避免回归
- 保持大脚本解析近似线性
- 防止新语法引入异常分配或指数退化

## 5. 最终判断

`icooclaw_lang` 当前的性能画像很清晰：

- `lexer/parser`：足够轻，未见异常
- `模块导入`：缓存后表现良好
- `evaluator`：是当前主要性能瓶颈

如果接下来要做性能优化，建议不要先从 parser 下手，而应聚焦 evaluator 的对象分配和作用域创建成本。这会比继续优化语法层获得更高的实际收益。

## 6. Profiler 深入分析

在 benchmark 汇总之外，本次还对两个 evaluator 基准抓取了 `pprof`：

- `BenchmarkEvalProgram`
- `BenchmarkEvalFunctionCalls`

采集方式：

```powershell
go test ./internal/evaluator -run ^$ -bench BenchmarkEvalProgram$ -cpuprofile cpu_eval_program.out -memprofile mem_eval_program.out -benchtime=3s
go test ./internal/evaluator -run ^$ -bench BenchmarkEvalFunctionCalls$ -cpuprofile cpu_eval_calls.out -memprofile mem_eval_calls.out -benchtime=3s
go tool pprof -top cpu_eval_program.out
go tool pprof -top mem_eval_program.out
go tool pprof -top cpu_eval_calls.out
go tool pprof -top mem_eval_calls.out
```

### 6.1 CPU 热点

`BenchmarkEvalProgram` 的关键热点：

- `github.com/issueye/icooclaw_lang/internal/evaluator.evalBlockStmt`
- `github.com/issueye/icooclaw_lang/internal/evaluator.evalForStmt`
- `github.com/issueye/icooclaw_lang/internal/object.NewEnclosedEnvironment`
- `github.com/issueye/icooclaw_lang/internal/object.(*Environment).Set`
- `runtime.mallocgc`
- `runtime.newobject`

`BenchmarkEvalFunctionCalls` 的关键热点：

- `github.com/issueye/icooclaw_lang/internal/evaluator.Eval`
- `github.com/issueye/icooclaw_lang/internal/evaluator.evalBlockStmt`
- `github.com/issueye/icooclaw_lang/internal/evaluator.evalCallExpr`
- `github.com/issueye/icooclaw_lang/internal/evaluator.callFunction`
- `github.com/issueye/icooclaw_lang/internal/object.NewEnclosedEnvironment`
- `github.com/issueye/icooclaw_lang/internal/object.(*Environment).Set`
- `runtime.mallocgc`

解释：

- CPU 时间并没有集中在某个复杂算法上，而是被“解释执行框架成本”吞掉了。
- `evalForStmt` / `evalBlockStmt` / `callFunction` 反复触发环境构造和对象分配，这是当前最核心的执行成本来源。

### 6.2 内存分配热点

`BenchmarkEvalProgram` 的 `alloc_space` 前几项：

- `Environment.Set`: `1613.94 MB`，约 `39.80%`
- `NewEnvironment`: `1202.57 MB`，约 `29.66%`
- `NewRuntime`: `933.55 MB`，约 `23.02%`
- `coreBuiltins.func4`: `247.06 MB`，约 `6.09%`

`BenchmarkEvalFunctionCalls` 的 `alloc_space` 前几项：

- `Environment.Set`: `1592.94 MB`，约 `39.93%`
- `NewEnvironment`: `1247.57 MB`，约 `31.28%`
- `NewRuntime`: `877.05 MB`，约 `21.99%`
- `coreBuiltins.func4`: `132.64 MB`，约 `3.33%`

解释：

- 分配热点几乎全部集中在环境对象生命周期上。
- `NewRuntime` 在 profile 中占比非常高，说明当前 `NewEnvironment()` 每次都会创建新的 runtime，然后再在 `NewEnclosedEnvironment()` 中被外层 runtime 覆盖，这是一笔典型的冗余分配。
- `Environment.Set` 占比接近 `40%`，说明变量绑定过程本身就很重，特别是在循环和函数调用密集场景里。

## 7. 更具体的优化建议

### P0：消除 `NewEnclosedEnvironment()` 中的冗余 runtime 创建

当前实现会：

1. 调用 `NewEnvironment()`
2. 在内部新建 `Runtime`
3. 然后再把 `env.runtime = outer.runtime`

这意味着大量 `Runtime` 被创建后立刻丢弃。  
从 profile 看，这一项单独就贡献了 `20%+` 的分配空间，应该优先修。

### P1：降低循环体中的 `Environment` 创建频率

当前 `for` 每轮创建新的 enclosed environment。  
如果语言语义允许，可以考虑：

- 复用循环作用域对象
- 或者为循环变量使用轻量级更新路径，而不是整张 `Environment` 重建

这是第二个最值得优化的点。

### P2：降低 `Set` 的 map 写入和锁成本

`Environment.Set` 同时承担：

- 变量查找
- 常量检查
- map 写入
- runtime 级锁

建议检查：

- 是否可以缩小锁粒度
- 是否可以减少 `findVarUnlocked` 的递归查找次数
- 是否可以在局部作用域场景下走更快的直写路径

### P3：减少函数调用路径中的短生命周期对象

重点检查：

- `callFunction`
- `evalArgs`
- `evalAssignExpr`
- `evalInfixExpr`

尤其是函数调用 benchmark 中，`evalCallExpr` 和 `callFunction` 都是持续热点。

## 8. 结论更新

在 benchmark 和 pprof 两层证据下，可以更明确地下结论：

- `icooclaw_lang` 当前性能瓶颈是“运行时环境模型过重”
- 不是 parser，也不是模块系统
- 第一批优化应该围绕 `Environment` / `Runtime` 的创建与写入路径展开

如果继续做下一轮优化，最合理的顺序是：

1. 修 `NewEnclosedEnvironment` 的 runtime 冗余分配
2. 优化 `for` / `callFunction` 的作用域创建
3. 再次跑 benchmark + pprof 验证收益

## 9. 已实施优化与收益验证

本轮已经实际实施了第一阶段优化：

- `NewEnclosedEnvironment()` 不再先调用 `NewEnvironment()` 再覆盖 `runtime`
- `NewDetachedEnvironment()` 同样移除冗余 runtime 创建
- `consts` / `exports` / `moduleCache` / `loadingModule` 改为按需初始化
- 作用域创建时不再重复复制 CLI 参数切片
- `evalArgs` 增加容量预分配

### 9.1 Benchmark 前后对比

#### Evaluator 主路径

| 指标 | 优化前 | 优化后 | 变化 |
| --- | ---: | ---: | ---: |
| `BenchmarkEvalProgram ns/op` | `~574-578 us` | `~396-398 us` | 约 `31%` 改善 |
| `BenchmarkEvalProgram B/op` | `~725952 B` | `~469242 B` | 约 `35%` 改善 |
| `BenchmarkEvalProgram allocs/op` | `10053` | `5039` | 约 `49.9%` 改善 |

#### 函数调用密集路径

| 指标 | 优化前 | 优化后 | 变化 |
| --- | ---: | ---: | ---: |
| `BenchmarkEvalFunctionCalls ns/op` | `~542-545 us` | `~369-370 us` | 约 `32%` 改善 |
| `BenchmarkEvalFunctionCalls B/op` | `~719607 B` | `~463409 B` | 约 `35.6%` 改善 |
| `BenchmarkEvalFunctionCalls allocs/op` | `10523` | `5519` | 约 `47.6%` 改善 |

#### 模块导入路径

| 指标 | 优化前 | 优化后 | 变化 |
| --- | ---: | ---: | ---: |
| `BenchmarkEvalModuleImportCold ns/op` | `~55.7-57.0 us` | `~55.2-55.5 us` | 小幅改善 |
| `BenchmarkEvalModuleImportCold B/op` | `10136 B` | `9560 B` | 改善 |
| `BenchmarkEvalModuleImportCold allocs/op` | `135` | `123` | 改善 |
| `BenchmarkEvalModuleImportWarm ns/op` | `~3.2-3.4 us` | `~2.8 us` | 约 `15%+` 改善 |
| `BenchmarkEvalModuleImportWarm B/op` | `2552 B` | `2024 B` | 改善 |
| `BenchmarkEvalModuleImportWarm allocs/op` | `33` | `22` | 明显改善 |

### 9.2 优化后 pprof 结论

优化后再次抓取 `BenchmarkEvalProgram` profile，内存热点变为：

- `Environment.Set`: `2246.12 MB`，约 `61.45%`
- `NewEnclosedEnvironment`: `997.07 MB`，约 `27.28%`
- `coreBuiltins.func4`: `331.76 MB`，约 `9.08%`

和优化前相比：

- `NewRuntime` 已经从主要内存热点中消失
- `NewEnvironment` 也不再出现在主要分配顶部
- 热点集中度更高，说明冗余分配已经被有效移除

优化后 CPU 侧仍然主要集中在：

- `Environment.Set`
- `evalForStmt`
- `evalBlockStmt`
- `NewEnclosedEnvironment`

这说明下一阶段的重点已经很明确：**继续压缩作用域创建成本和变量写入成本**。

## 10. 关于“是否引入内存池”的结论更新

基于这轮优化结果，当前判断是：

- 小心使用内存池是有价值的
- 但还不应该直接对所有运行时对象做池化

原因：

- 仅通过移除冗余分配，就已经拿到了 `30%+` 的耗时收益和接近 `50%` 的分配次数下降
- 说明当前还有很多“结构性优化”没有做完
- 这类优化比对象池更安全、可解释性更强、回归风险更低

因此建议更新为：

1. 继续先做结构性优化
2. 等 `Environment.Set` / `NewEnclosedEnvironment` 进一步收敛后，再考虑引入**受控范围**的池化

最适合下一步尝试池化的候选：

- 临时切片缓冲
- 某些明确不逃逸的辅助结构

暂时不建议直接池化：

- `Integer` / `String` / `Hash` 等运行时对象
- 可能被闭包、模块导出、外部变量持有的作用域对象

## 11. 第二轮优化尝试

在第一轮“去冗余分配”之后，又继续做了一轮低风险优化：

- 为新建局部作用域增加 `DefineLocal`
- `for` 循环变量绑定改走局部定义快路径
- 函数参数绑定改走局部定义快路径
- `match` 捕获变量和 `catch` 变量绑定改走局部定义快路径
- `Set` 增加“当前作用域命中”的快速分支

### 11.1 第二轮结果

复测命令：

```powershell
go test ./internal/evaluator -run ^$ -bench 'BenchmarkEvalProgram$|BenchmarkEvalFunctionCalls$|BenchmarkEvalModuleImportWarm$' -benchmem -count 5
```

结果观察：

- `BenchmarkEvalProgram` 大致稳定在 `~385-405 us/op`
- `BenchmarkEvalFunctionCalls` 大致稳定在 `~394-456 us/op`
- `allocs/op` 和 `B/op` 基本没有继续下降

解释：

- 这轮优化对“额外查找和调用路径”有帮助，但没有减少根本性的对象创建数量
- 当前真正限制进一步收益的，已经不是“是否走快路径调用 `Set`”，而是：
  - `Set` 内部的锁和 map 写入成本
  - 每轮循环 / 每次函数调用都新建作用域对象

因此下一阶段如果要继续追求明显收益，应优先考虑：

1. 作用域对象复用策略
2. `Environment.Set` 的内部结构优化
3. 临时缓冲池化

## 12. 第三轮优化尝试

第二轮之后，没有继续碰“作用域对象复用”，因为这条线虽然潜在收益更高，但对闭包、递归和模块执行语义的回归风险也更高。  
因此第三轮选择了更保守的两项改动：

- `NewEnclosedEnvironment()` / `NewDetachedEnvironment()` 的 `store` 改为小容量预分配
- 为 `evalCallExpr` / `evalMethodCallExpr` 引入只在调用期内使用的临时参数切片池

设计边界：

- 只池化“立即调用、立即释放”的参数列表
- `ArrayLiteral` 仍然保留普通 `evalArgs` 路径，避免把会逃逸到结果对象里的切片放回池中
- 不直接池化运行时值对象，也不复用作用域对象

### 12.1 第三轮结果

复测命令：

```powershell
go test ./internal/evaluator -run ^$ -bench BenchmarkEvalProgram$ -benchmem -count 3
go test ./internal/evaluator -run ^$ -bench BenchmarkEvalFunctionCalls$ -benchmem -count 3
go test ./internal/evaluator -run ^$ -bench BenchmarkEvalModuleImportWarm$ -benchmem -count 3
```

结果：

| Benchmark | 典型耗时 | 内存 | 分配次数 |
| --- | ---: | ---: | ---: |
| `BenchmarkEvalProgram` | `396-400 us/op` | `469491-469502 B/op` | `5037 allocs/op` |
| `BenchmarkEvalFunctionCalls` | `378-381 us/op` | `455651-455656 B/op` | `5018 allocs/op` |
| `BenchmarkEvalModuleImportWarm` | `2.78-2.79 us/op` | `1993 B/op` | `21 allocs/op` |

### 12.2 结果解释

- `BenchmarkEvalProgram` 基本与第一轮优化后的水平持平，说明参数切片池化对“非调用主导型”脚本帮助有限
- `BenchmarkEvalFunctionCalls` 继续改善，且 `B/op` / `allocs/op` 进一步下降，说明这轮优化主要命中了高频调用路径
- `BenchmarkEvalModuleImportWarm` 也有小幅改善，但仍然不是主要瓶颈

因此第三轮结论是：

- **临时缓冲池化有效，但收益集中在 call-heavy workload**
- 它可以作为安全的局部优化手段
- 但如果要继续拿到量级更大的整体收益，下一优先级仍然是：
  - `Environment.Set`
  - `NewEnclosedEnvironment`
  - 更谨慎的作用域创建/复用途径设计

## 13. 第四轮优化尝试

第三轮之后，继续沿着“减少运行时锁与重复局部写入成本”的方向做了一次更细化的尝试：

- 函数参数绑定从“每个参数一次 `DefineLocal`”改为单次加锁的批量绑定
- `match` 命中的绑定写入也改为一次性批量落入新作用域

实现目标不是减少对象数量，而是验证“减少锁次数 / 减少重复局部写入调用”是否仍有可见收益。

### 13.1 第四轮结果

复测命令：

```powershell
go test ./internal/evaluator -run ^$ -bench BenchmarkEvalProgram$ -benchmem -count 3
go test ./internal/evaluator -run ^$ -bench BenchmarkEvalFunctionCalls$ -benchmem -count 3
go test ./internal/evaluator -run ^$ -bench BenchmarkEvalModuleImportWarm$ -benchmem -count 3
```

结果：

| Benchmark | 典型耗时 | 内存 | 分配次数 |
| --- | ---: | ---: | ---: |
| `BenchmarkEvalProgram` | `392-396 us/op` | `469490-469499 B/op` | `5037 allocs/op` |
| `BenchmarkEvalFunctionCalls` | `378-383 us/op` | `455651-455658 B/op` | `5018 allocs/op` |
| `BenchmarkEvalModuleImportWarm` | `2.734-2.739 us/op` | `1993 B/op` | `21 allocs/op` |

### 13.2 结果解释

- `BenchmarkEvalProgram` 有轻微 CPU 改善，但非常有限
- `BenchmarkEvalFunctionCalls` 与第三轮基本持平
- `B/op` 和 `allocs/op` 完全没有继续下降

这说明：

- 仅仅减少“同一作用域内多次局部写入的锁次数”，对当前 evaluator 已经不是决定性因素
- 剩余的主要成本仍然来自：
  - 作用域对象本身的创建
  - `store` 的 map 写入
  - 解释执行过程中大量短生命周期对象

因此第四轮的结论比前几轮更明确：

- **局部微优化仍能带来小幅 CPU 改善**
- 但要再拿到明显收益，必须进入更高风险但更高收益的区域，例如：
  - 设计受控的作用域复用途径
  - 调整 `Environment` 的内部表示
  - 针对循环/调用热点做更专门的轻量作用域模型
