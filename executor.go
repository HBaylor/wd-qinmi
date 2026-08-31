package main

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// Action 描述列表中的一项按键行为。
// Value 是内部标识（用于键映射），Text 是用户可见的展示文本。
type Action struct {
	Value string
	Text  string
}

// LogLevel 用于在 UI 中按颜色区分日志。
type LogLevel int

const (
	LogInfo LogLevel = iota
	LogSuccess
	LogWarning
	LogError
)

// Callbacks 由 UI 注入，所有回调都会在执行器的 Goroutine 中被调用，
// UI 层需要自行 fyne.Do 切换到主线程。
type Callbacks struct {
	// OnLog 每产生一条日志时触发
	OnLog func(msg string, level LogLevel)
	// OnFinished 所有循环自然完成时触发，参数为完成的循环数
	OnFinished func(loops int)
	// OnStopped 用户主动停止时触发
	OnStopped func()
}

// Executor 在 Goroutine 中按顺序、循环地模拟按键行为。
// 取消语义: 调用 Stop 后当前动作完成后立刻退出；动作间/循环间的 sleep
// 也会被 ctx.Done 唤醒而不会持续阻塞。
type Executor struct {
	sim KeySimulator

	mu      sync.Mutex
	running bool
	cancel  context.CancelFunc
	cb      Callbacks
}

func NewExecutor(sim KeySimulator, cb Callbacks) *Executor {
	return &Executor{sim: sim, cb: cb}
}

// IsRunning 返回当前是否处于执行状态。
func (e *Executor) IsRunning() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.running
}

// Start 在新的 Goroutine 中启动执行。
// actions: 行为序列; interval: 相邻动作间隔; delay: 循环间隔;
// loops: 循环次数（小于 1 会被规范化为 1）。
// 处于运行中再次调用将返回错误，调用方应在 UI 层先判断状态。
func (e *Executor) Start(actions []Action, interval, delay time.Duration, loops int) error {
	if len(actions) == 0 {
		return errors.New("行为列表为空")
	}
	if interval < 0 || delay < 0 {
		return errors.New("间隔时间不能为负")
	}
	if loops < 1 {
		loops = 1
	}

	e.mu.Lock()
	if e.running {
		e.mu.Unlock()
		return errors.New("执行器已在运行")
	}
	e.running = true
	ctx, cancel := context.WithCancel(context.Background())
	e.cancel = cancel
	e.mu.Unlock()

	go e.run(ctx, actions, interval, delay, loops)
	return nil
}

// Stop 请求执行器停止。该方法立即返回，真正退出发生在下一次循环检查点。
func (e *Executor) Stop() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.cancel != nil {
		e.cancel()
	}
}

func (e *Executor) run(ctx context.Context, actions []Action, interval, delay time.Duration, loops int) {
	defer func() {
		e.mu.Lock()
		e.running = false
		e.cancel = nil
		e.mu.Unlock()
	}()

	for loopIdx := 0; loopIdx < loops; loopIdx++ {
		if ctx.Err() != nil {
			e.fireStopped()
			return
		}
		for i, a := range actions {
			if ctx.Err() != nil {
				e.fireStopped()
				return
			}
			// 第一个动作、第一个循环不等待
			if loopIdx > 0 || i > 0 {
				if !sleepCtx(ctx, interval) {
					e.fireStopped()
					return
				}
			}
			if err := e.sim.Press(a.Value); err != nil {
				e.cb.OnLog(fmt.Sprintf("[%s] 模拟按键失败: %s - %v", nowHMS(), a.Text, err), LogError)
				continue
			}
			e.cb.OnLog(fmt.Sprintf("[%s] %s", nowHMS(), a.Text), LogInfo)
		}
		// 最后一轮完成后不再延迟
		if loopIdx < loops-1 {
			if !sleepCtx(ctx, delay) {
				e.fireStopped()
				return
			}
		}
	}
	if e.cb.OnFinished != nil {
		e.cb.OnFinished(loops)
	}
}

func (e *Executor) fireStopped() {
	if e.cb.OnStopped != nil {
		e.cb.OnStopped()
	}
}

// sleepCtx 等待 d 期间可被 ctx 唤醒。
// 返回 true 表示自然耗尽，false 表示被取消。
func sleepCtx(ctx context.Context, d time.Duration) bool {
	if d <= 0 {
		return ctx.Err() == nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}

func nowHMS() string {
	return time.Now().Format("15:04:05")
}
