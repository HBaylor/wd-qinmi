package main

import (
	"log"
)

func main() {
	// 创建真实键盘模拟器
	sim, err := newRealKeySimulator()
	if err != nil {
		log.Fatalf("无法初始化键盘模拟器: %v", err)
	}
	defer sim.Close()

	// 构造 UI（UI 内部会创建 Executor 并通过 fyne.Do 将回调切回主线程）
	ui := NewUI(sim)
	ui.Run()
}
