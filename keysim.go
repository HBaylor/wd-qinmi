package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/micmonay/keybd_event"
)

// KeySimulator 抽象键盘模拟，便于将来替换实现或注入 mock 进行测试。
type KeySimulator interface {
	// Press 模拟按下并释放一个已注册的行为。
	Press(action string) error
	// Close 释放底层资源。
	Close() error
}

// realKeySimulator 是基于 keybd_event 的真实实现。
//
// 推荐库: github.com/micmonay/keybd_event
// 平台支持:
//   - Windows: 原生支持，开箱即用
//   - Linux:   需要 X11，使用 uinput
//   - macOS:   可用，但需要在「系统设置 - 辅助功能」中授予终端/应用权限
//
// 若 macOS 上权限受限导致失败，可在 keyMap 范围内替换为
// github.com/go-vgo/robotgo（功能更全面，但体积更大且重度依赖 CGo）。
type realKeySimulator struct {
	kb keybd_event.KeyBonding
}

func newRealKeySimulator() (*realKeySimulator, error) {
	kb, err := keybd_event.NewKeyBonding()
	if err != nil {
		return nil, fmt.Errorf("初始化 keybd_event 失败: %w", err)
	}
	return &realKeySimulator{kb: kb}, nil
}

// keyMap 将内部 action 值映射到修饰键与虚拟键码。
// 与 Python 端键盘模拟行为保持一致。
var keyMap = map[string]struct {
	ctrl bool
	keys []int
}{
	"ctrl_1":   {ctrl: true, keys: []int{keybd_event.VK_1}},
	"ctrl_2":   {ctrl: true, keys: []int{keybd_event.VK_2}},
	"ctrl_3":   {ctrl: true, keys: []int{keybd_event.VK_3}},
	"enter":    {ctrl: false, keys: []int{keybd_event.VK_ENTER}},
	"tab":      {ctrl: false, keys: []int{keybd_event.VK_TAB}},
	"ctrl_tab": {ctrl: true, keys: []int{keybd_event.VK_TAB}},
	"ctrl_c":   {ctrl: true, keys: []int{keybd_event.VK_C}},
	"ctrl_v":   {ctrl: true, keys: []int{keybd_event.VK_V}},
	"ctrl_z":   {ctrl: true, keys: []int{keybd_event.VK_Z}},
	"ctrl_a":   {ctrl: true, keys: []int{keybd_event.VK_A}},
}

func (s *realKeySimulator) Press(action string) error {
	// 与 Python 端保持一致的小延迟，避免系统识别丢失
	time.Sleep(50 * time.Millisecond)

	entry, ok := keyMap[action]
	if !ok {
		return fmt.Errorf("未注册的按键行为: %s", action)
	}

	s.kb.HasCTRL(false)
	if entry.ctrl {
		s.kb.HasCTRL(true)
	}
	s.kb.SetKeys(entry.keys...)
	if err := s.kb.Launching(); err != nil {
		return fmt.Errorf("按键发送失败 [%s]: %w", strings.ToUpper(action), err)
	}
	return nil
}

func (s *realKeySimulator) Close() error {
	// keybd_event 当前没有需要显式释放的资源
	return nil
}
