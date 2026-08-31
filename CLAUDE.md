# CLAUDE.md

Behavioral guidelines to reduce common LLM coding mistakes. Merge with project-specific instructions as needed.

**Tradeoff:** These guidelines bias toward caution over speed. For trivial tasks, use judgment.

## 1. Think Before Coding

**Don't assume. Don't hide confusion. Surface tradeoffs.**

Before implementing:

- State your assumptions explicitly. If uncertain, ask.
- If multiple interpretations exist, present them - don't pick silently.
- If a simpler approach exists, say so. Push back when warranted.
- If something is unclear, stop. Name what's confusing. Ask.

## 2. Simplicity First

**Minimum code that solves the problem. Nothing speculative.**

- No features beyond what was asked.
- No abstractions for single-use code.
- No "flexibility" or "configurability" that wasn't requested.
- No error handling for impossible scenarios.
- If you write 200 lines and it could be 50, rewrite it.

Ask yourself: "Would a senior engineer say this is overcomplicated?" If yes, simplify.

## 3. Surgical Changes

**Touch only what you must. Clean up only your own mess.**

When editing existing code:

- Don't "improve" adjacent code, comments, or formatting.
- Don't refactor things that aren't broken.
- Match existing style, even if you'd do it differently.
- If you notice unrelated dead code, mention it - don't delete it.

When your changes create orphans:

- Remove imports/variables/functions that YOUR changes made unused.
- Don't remove pre-existing dead code unless asked.

The test: Every changed line should trace directly to the user's request.

## 4. Goal-Driven Execution

**Define success criteria. Loop until verified.**

Transform tasks into verifiable goals:

- "Add validation" → "Write tests for invalid inputs, then make them pass"
- "Fix the bug" → "Write a test that reproduces it, then make it pass"
- "Refactor X" → "Ensure tests pass before and after"

For multi-step tasks, state a brief plan:

```

1. [Step] → verify: [check]
2. [Step] → verify: [check]
3. [Step] → verify: [check]

```

Strong success criteria let you loop independently. Weak criteria ("make it work") require constant clarification.

---

**These guidelines are working if:** fewer unnecessary changes in diffs, fewer rewrites due to overcomplication, and clarifying questions come before implementation rather than after mistakes.

---

## Project

**action_composer**

行为组合器 — 一个 Go GUI 工具，将常用的按键组合添加为"行为"并按顺序执行，可设置动作间隔、循环延迟和循环次数。用于辅助测试、自动化操作等场景。

### Constraints

- **Tech stack**: Go + Fyne v2 + keybd_event
- **Language**: Go（不使用泛型之外的复杂特性）
- **GUI**: Fyne v2（fyne.io/fyne/v2）
- **Conventions**: 2 空格缩进，中文注释，英文代码，PascalCase 导出，camelCase 私有

---

## Technology Stack

- **语言**: Go 1.26+
- **GUI**: `fyne.io/fyne/v2 v2.7.4`
- **键盘模拟**: `github.com/micmonay/keybd_event v1.1.2`
- **运行方式**:
  - `go run .` — 直接运行（开发）
  - `go build -o action_composer .` — 构建二进制
- **平台**: macOS / Linux / Windows
  - macOS 需在「系统设置 - 辅助功能」中授予终端/应用权限

---

## Conventions

### 代码风格

- 缩进：2 空格
- 字符串：双引号
- 中文注释用于说明性描述，英文用于代码逻辑
- 导出标识符 PascalCase，未导出 camelCase
- 包名小写单词（当前为 `main`）

### 布局约定

- UI 顶层使用 `container.NewBorder` 划分：中央为「快捷按键区 + 行为组合列表区」双列，底部为「功能操作区」
- 快捷按键区使用 `container.NewGridWithColumns(2, ...)` 两列布局
- 行为组合列表使用 `widget.NewList` + `container.NewScroll` 包裹
- 控件统一定义在 `UI` 结构体字段中，构造在 `build*()` 方法中

### 事件处理

- Fyne 按钮使用 `widget.NewButton(text, func(){...})`
- Executor 回调需在 `fyne.Do(...)` 内执行，确保主线程访问控件
- 列表与执行器共享数据时使用 `actionsMu sync.Mutex` 保护

### 并发模型

- 执行在独立 goroutine 中进行，通过 `context.CancelFunc` 取消
- `sleepCtx(ctx, d)` 同时监听 `ctx.Done()` 与定时器，可被立即唤醒
- 状态读写使用 `sync.Mutex`，不依赖 channel 传递业务消息

---

## 功能说明

### 快捷按键区

- 6 个核心键盘快捷键：Ctrl+1, Enter, Ctrl+2, Ctrl+Tab, Ctrl+3, Tab
- 2 列 grid 布局
- 点击按钮将行为添加到右侧列表

### 行为组合列表

- 显示已添加的行为序列（带序号）
- 支持清空列表操作
- 按添加顺序执行

### 功能操作区

- **动作间隔 (秒)**：相邻动作之间的延迟（默认 1.0s）
- **循环延迟 (秒)**：一轮结束后到下一轮开始前的等待时间（默认 10.0s）
- **循环次数**：重复执行轮数（默认 1 次）
- **开始/结束** 按钮控制执行；执行中禁用所有输入控件与添加按钮
- **状态/计数** 标签实时显示当前状态
- **执行日志**：彩色分级日志（Info/Success/Warning/Error），自动滚到底部

---

## Architecture

### 文件结构

```
main.go        — 入口：初始化 KeySimulator、构造 UI、启动 Fyne 事件循环
keysim.go      — KeySimulator 接口 + realKeySimulator（基于 keybd_event）
executor.go    — Executor：异步执行逻辑（goroutine + context 取消）
ui.go          — UI：Fyne 控件、状态机、回调、参数解析
```

### 核心类型

- `Action{Value, Text string}` — 行为描述；`Value` 是内部标识（用于 `keyMap`），`Text` 是用户可见文本
- `LogLevel int` — `LogInfo | LogSuccess | LogWarning | LogError`
- `Callbacks{OnLog, OnFinished, OnStopped}` — Executor 通知 UI 的回调集合
- `KeySimulator interface { Press(string) error; Close() error }` — 键盘模拟抽象，便于将来替换或注入 mock
- `Executor` — 异步执行器；`Start(actions, interval, delay, loops)` / `Stop()` / `IsRunning()`
- `UI` — 持有所有 Fyne 控件、共享的 `actionsMu` 互斥的 `actions []Action`

### 类职责

- `Executor`
  - `Start(actions, interval, delay, loops)` — 校验入参后启动 goroutine；运行中再次调用返回 error
  - `Stop()` — 调用 `cancel()`，立即返回；真正退出发生在下一个循环检查点
  - `run(ctx, ...)` — 双层循环；动作间与循环间用 `sleepCtx` 监听取消
  - `sleepCtx(ctx, d)` — 可被 `ctx.Done()` 唤醒的 sleep
- `UI`
  - `NewUI(sim)` — 构造 fyneApp、window、executor；Executor 回调通过 `fyne.Do` 切回主线程
  - `build()` / `buildPresetPanel()` / `buildListPanel()` / `buildFuncPanel()` — 界面装配
  - `addAction / clearActions` — 增删行为，更新列表与日志
  - `startExecution / stopExecution` — 控制执行；切换 `setRunningUI(running)` 启用/禁用控件
  - `onExecutionFinished / onExecutionStopped` — Executor 回调
  - `setRunningUI / setStatus / appendLog / updateCount` — UI 状态更新
  - `parseParams / numericValidator` — 参数解析与校验

### 数据流

1. 用户点击快捷键按钮 → `addAction(value, text)` → `actions = append(...)` + `actionList.Refresh()` + 日志
2. 用户点击「开始」 → `startExecution()` → 解析参数 + 快照 actions → `exec.Start(...)` → 切换为运行态 UI
3. Executor goroutine 按 `sleepCtx` 节奏执行 `sim.Press(action.Value)`，通过 `Callbacks.OnLog` 写日志
4. 自然完成 → `OnFinished(loops)`；用户停止 → `OnStopped()`；均在 `fyne.Do` 内切回主线程
5. 用户点击「清空列表」 → `clearActions()` → `actions = actions[:0]` + `actionList.Refresh()`

### 取消语义

- `Stop()` 立即返回，不阻塞 UI
- 当前动作完成后立刻检查 `ctx.Err()` 并退出
- `sleepCtx` 期间被取消会立即返回 `false`，调用方据此退出
- 自然完成与被停止走不同回调（`OnFinished` vs `OnStopped`），UI 区分日志级别

---

## Anti-Patterns

- 不要在 UI 回调中直接访问 Fyne 控件，必须先经 `fyne.Do` 切回主线程
- 不要在 `executor.go` 中直接引用 Fyne 类型，保持纯逻辑无 GUI 依赖
- 不要用 `time.Sleep` 替换 `sleepCtx`，否则无法响应取消
- 不要省略 `actionsMu` 互斥；UI 列表回调与 Executor 快照可能并发
- 不要在 `keyMap` 中添加没有对应 `keybd_event.VK_*` 常量的条目
- 不要把 `keybd_event` 替换为 `robotgo`，除非用户明确要求（后者依赖 CGo、体积大）
