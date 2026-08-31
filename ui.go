package main

import (
	"errors"
	"fmt"
	"image/color"
	"strconv"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// UI 持有所有 Fyne 控件与业务状态。
// 与 executor 之间的回调全部经 fyne.Do 切回主线程，确保 Fyne 控件访问的线程安全。
type UI struct {
	fyneApp fyne.App
	win     fyne.Window
	exec    *Executor
	sim     KeySimulator

	actionsMu sync.Mutex
	actions   []Action

	licensed  bool
	machineID string

	// 控件
	presetButtons []*widget.Button
	actionList    *widget.List
	clearBtn      *widget.Button
	intervalEntry *widget.Entry
	delayEntry    *widget.Entry
	loopsEntry    *widget.Entry
	startBtn      *widget.Button
	stopBtn       *widget.Button
	statusLabel   *widget.Label
	countLabel    *widget.Label
	logBox        *widget.RichText
	logScroll     *container.Scroll
	licenseEntry  *widget.Entry
	licenseBtn    *widget.Button
}

// NewUI 创建并装配界面。UI 自带 Executor，回调通过 fyne.Do 回到主线程。
func NewUI(sim KeySimulator) *UI {
	u := &UI{
		fyneApp: app.New(),
		win:     nil,
		sim:     sim,
	}
	u.fyneApp.Settings().SetTheme(newCompactTheme())
	u.win = u.fyneApp.NewWindow("行为组合器 (v1.0.0)")
	u.win.Resize(fyne.NewSize(420, 510))

	// 自引用建立回调：在 Executor 触发回调时 *u 已经构造完成，访问字段是安全的。
	u.exec = NewExecutor(sim, Callbacks{
		OnLog: func(msg string, level LogLevel) {
			fyne.Do(func() { u.appendLog(msg, level) })
		},
		OnFinished: func(loops int) {
			fyne.Do(func() { u.onExecutionFinished(loops) })
		},
		OnStopped: func() {
			fyne.Do(func() { u.onExecutionStopped() })
		},
	})

	u.build()
	u.initLicense()
	return u
}

// Run 启动 Fyne 事件循环。
func (u *UI) Run() {
	u.win.ShowAndRun()
}

func (u *UI) build() {
	// 顶部区域高度固定为 200px，限制两区不被拉伸
	top := container.New(&fixedHeightLayout{height: 200},
		container.NewGridWithColumns(
			2,
			u.buildPresetPanel(),
			u.buildListPanel(),
		),
	)
	content := container.NewBorder(
		nil,                // top
		u.buildFuncPanel(), // bottom
		nil, nil,           // left, right
		top,                // center
	)
	u.win.SetContent(content)
}

// bordered 用描边矩形包裹内容，方便区分区域边界
func bordered(obj fyne.CanvasObject) fyne.CanvasObject {
	border := canvas.NewRectangle(color.Transparent)
	border.StrokeWidth = 1
	border.StrokeColor = theme.Color(theme.ColorNameInputBorder)
	return container.NewBorder(nil, nil, nil, nil, obj, border)
}

// fixedHeightLayout 固定子对象高度（宽度随容器），阻止被外层布局拉伸
type fixedHeightLayout struct {
	height float32
}

func (l *fixedHeightLayout) Layout(objs []fyne.CanvasObject, size fyne.Size) {
	objs[0].Resize(fyne.NewSize(size.Width, l.height))
}

func (l *fixedHeightLayout) MinSize(objs []fyne.CanvasObject) fyne.Size {
	min := fyne.NewSize(0, l.height)
	if len(objs) > 0 {
		min.Width = objs[0].MinSize().Width
	}
	return min
}

// fixedHeight 将控件强制为指定高度（宽度自适应容器/内容）
func fixedHeight(obj fyne.CanvasObject, h float32) fyne.CanvasObject {
	return container.New(&fixedHeightLayout{height: h}, obj)
}

// fixedSize 将控件强制为指定尺寸
func fixedSize(obj fyne.CanvasObject, w, h float32) fyne.CanvasObject {
	return container.New(&fixedSizeLayout{width: w, height: h}, obj)
}

// fixedSizeLayout 固定子对象尺寸
type fixedSizeLayout struct {
	width, height float32
}

func (l *fixedSizeLayout) Layout(objs []fyne.CanvasObject, size fyne.Size) {
	objs[0].Resize(fyne.NewSize(l.width, l.height))
}

func (l *fixedSizeLayout) MinSize(_ []fyne.CanvasObject) fyne.Size {
	return fyne.NewSize(l.width, l.height)
}

// ---------- 快捷按键区 ----------
func (u *UI) buildPresetPanel() fyne.CanvasObject {
	// 与 Python 版一致：两列布局，按添加顺序排列
	presets := []struct {
		value string
		text  string
	}{
		{"ctrl_1", "Ctrl+1"},
		{"enter", "Enter"},
		{"ctrl_2", "Ctrl+2"},
		{"ctrl_tab", "Ctrl+Tab"},
		{"ctrl_3", "Ctrl+3"},
		{"tab", "Tab"},
	}

	objs := make([]fyne.CanvasObject, 0, len(presets))
	for _, p := range presets {
		p := p // 闭包捕获
		btn := widget.NewButton(p.text, func() {
			u.addAction(p.value, p.text)
		})
		u.presetButtons = append(u.presetButtons, btn)
		objs = append(objs, btn)
	}
	grid := container.NewGridWithColumns(1, objs...)
	card := widget.NewCard("", "", grid)
	return bordered(container.NewThemeOverride(
		container.NewVBox(redTitle("快捷按键区"), card),
		newTinyTheme(),
	))
}

// ---------- 行为组合列表区 ----------
func (u *UI) buildListPanel() fyne.CanvasObject {
	u.actionList = widget.NewList(
		func() int {
			u.actionsMu.Lock()
			defer u.actionsMu.Unlock()
			return len(u.actions)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("")
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			u.actionsMu.Lock()
			defer u.actionsMu.Unlock()
			if id < len(u.actions) {
				item.(*widget.Label).SetText(fmt.Sprintf("%d. %s", id+1, u.actions[id].Text))
			}
		},
	)
	u.actionList.OnSelected = func(id widget.ListItemID) {
		u.actionList.Unselect(id)
	}

	scroll := container.NewScroll(u.actionList)
	scroll.SetMinSize(fyne.NewSize(0, 180))

	u.clearBtn = widget.NewButton("清空列表", func() {
		u.clearActions()
	})

	return bordered(container.NewBorder(
		widget.NewLabel("行为组合列表（按添加顺序执行）"),
		container.NewHBox(layout.NewSpacer(), u.clearBtn, layout.NewSpacer()),
		nil, nil,
		scroll,
	))
}

// ---------- 功能操作区 ----------
func (u *UI) buildFuncPanel() fyne.CanvasObject {
	u.intervalEntry = widget.NewEntry()
	u.intervalEntry.SetText("2.0")
	u.intervalEntry.Validator = numericValidator()

	u.delayEntry = widget.NewEntry()
	u.delayEntry.SetText("20.0")
	u.delayEntry.Validator = numericValidator()

	u.loopsEntry = widget.NewEntry()
	u.loopsEntry.SetText("1000")
	u.loopsEntry.Validator = numericValidator()

	paramRow := container.NewVBox(
		container.NewGridWithColumns(2, widget.NewLabel("动作间隔 (秒):"), fixedHeight(u.intervalEntry, 24)),
		container.NewGridWithColumns(2, widget.NewLabel("循环延迟 (秒):"), fixedHeight(u.delayEntry, 24)),
		container.NewGridWithColumns(2, widget.NewLabel("循环次数:"), fixedHeight(u.loopsEntry, 24)),
	)

	u.statusLabel = widget.NewLabel("状态：空闲")
	u.countLabel = widget.NewLabel("已添加：0 个行为")

	u.startBtn = widget.NewButton("开始", func() {
		u.startExecution()
	})
	u.startBtn.Importance = widget.HighImportance

	u.stopBtn = widget.NewButton("结束", func() {
		u.stopExecution()
	})
	u.stopBtn.Importance = widget.DangerImportance
	u.stopBtn.Disable()
	u.startBtn.Disable() // 默认未授权，验证通过后解锁

	bigBtnRow := container.NewGridWithColumns(2,
		container.NewThemeOverride(u.startBtn, newBigButtonTheme()),
		container.NewThemeOverride(u.stopBtn, newBigButtonTheme()),
	)

	statusRow := container.NewBorder(
		nil, nil,
		nil,
		nil,
		bigBtnRow,
	)

	u.logBox = widget.NewRichText()
	u.logBox.Wrapping = fyne.TextWrapBreak
	u.logScroll = container.NewScroll(u.logBox)
	u.logScroll.SetMinSize(fyne.NewSize(0, 90))

	authEntry := widget.NewEntry()
	authEntry.Disable()
	authEntry.SetText(getMachineID())
	authCopyBtn := widget.NewButton("复制", func() {
		u.fyneApp.Clipboard().SetContent(authEntry.Text)
	})
	authRow := container.NewBorder(nil, nil, nil,
		fixedSize(container.NewThemeOverride(authCopyBtn, newBigButtonTheme()), 100, 24),
		fixedHeight(authEntry, 24))

	u.licenseEntry = widget.NewEntry()
	u.licenseEntry.SetPlaceHolder("请输入授权码")
	u.licenseBtn = widget.NewButton("授权", func() {
		u.onLicense(u.licenseEntry.Text)
	})
	u.licenseBtn.Importance = widget.DangerImportance // 未授权默认红色
	licenseRow := container.NewBorder(nil, nil, nil,
		fixedSize(container.NewThemeOverride(u.licenseBtn, newBigButtonTheme()), 100, 24),
		fixedHeight(u.licenseEntry, 24))

	authPanel := bordered(container.NewVBox(redTitle("认证授权"), authRow, licenseRow))

	funcPanel := container.NewVBox(
		paramRow,
		widget.NewSeparator(),
		statusRow,
		authPanel,
		widget.NewSeparator(),
		widget.NewLabel("执行日志："),
		u.logScroll,
	)
	titleRow := container.NewBorder(
		nil, nil,
		redTitle("功能操作区"), nil,
		container.NewHBox(layout.NewSpacer(), u.statusLabel, u.countLabel),
	)
	return bordered(container.NewVBox(titleRow, widget.NewCard("", "", funcPanel)))
}

// ---------- 授权控制 ----------

// initLicense 启动时读取本机缓存授权码，命中则自动解锁
func (u *UI) initLicense() {
	u.machineID = getMachineID()
	if cached, ok := loadCachedLicense(); ok && verifyLicense(u.machineID, cached) {
		u.setLicensed(cached)
	}
}

// setLicensed 切换授权状态：未授权时禁用开始/结束按钮；授权时回显授权码并锁定
func (u *UI) setLicensed(code string) {
	ok := code != ""
	u.licensed = ok
	if ok {
		u.startBtn.Enable()
		u.licenseEntry.SetText(code)
		u.licenseEntry.Disable()
		u.licenseBtn.Importance = widget.SuccessImportance
		u.licenseBtn.Refresh()
		u.appendLog(fmt.Sprintf("[%s] 授权验证通过，功能已解锁", nowHMS()), LogSuccess)
		u.setStatus("状态：已授权")
	} else {
		u.startBtn.Disable()
		u.stopBtn.Disable()
		u.appendLog(fmt.Sprintf("[%s] 未授权，请输入授权码解锁功能", nowHMS()), LogWarning)
	}
}

// onLicense 授权码提交处理
func (u *UI) onLicense(code string) {
	if strings.TrimSpace(code) == "" {
		u.appendLog("授权码不能为空", LogWarning)
		return
	}
	if verifyLicense(u.machineID, code) {
		if err := saveCachedLicense(code); err != nil {
			u.appendLog(fmt.Sprintf("[%s] 授权码缓存失败: %v", nowHMS(), err), LogError)
		}
		u.setLicensed(code)
	} else {
		u.appendLog(fmt.Sprintf("[%s] 授权码错误", nowHMS()), LogError)
	}
}

// ---------- 行为操作 ----------
func (u *UI) addAction(value, text string) {
	u.actionsMu.Lock()
	u.actions = append(u.actions, Action{Value: value, Text: text})
	u.actionsMu.Unlock()

	u.actionList.Refresh()
	u.updateCount()
	u.appendLog(fmt.Sprintf("[%s] 添加行为: %s", nowHMS(), text), LogInfo)
	u.setStatus(fmt.Sprintf("已添加：%s", text))
}

func (u *UI) clearActions() {
	u.actionsMu.Lock()
	u.actions = u.actions[:0]
	u.actionsMu.Unlock()

	u.actionList.Refresh()
	u.updateCount()
	u.appendLog(fmt.Sprintf("[%s] 清空所有行为", nowHMS()), LogWarning)
	u.setStatus("已清空所有行为")
}

// ---------- 执行控制 ----------
func (u *UI) startExecution() {
	if u.exec.IsRunning() {
		u.setStatus("状态：执行中，请先结束")
		return
	}

	interval, delay, loops, err := u.parseParams()
	if err != nil {
		u.setStatus(fmt.Sprintf("参数错误: %v", err))
		u.appendLog(fmt.Sprintf("[%s] 参数错误: %v", nowHMS(), err), LogError)
		return
	}

	actions := u.snapshotActions()
	if len(actions) == 0 {
		u.setStatus("列表为空，请先添加行为")
		return
	}

	if err := u.exec.Start(actions, interval, delay, loops); err != nil {
		u.setStatus(err.Error())
		return
	}

	u.setRunningUI(true)
	u.appendLog(
		fmt.Sprintf("[%s] 开始执行 (动作间隔=%.1fs, 循环延迟=%.1fs, 循环=%d次)",
			nowHMS(), interval.Seconds(), delay.Seconds(), loops),
		LogInfo,
	)
}

func (u *UI) stopExecution() {
	if !u.exec.IsRunning() {
		return
	}
	u.exec.Stop()
}

func (u *UI) onExecutionFinished(loops int) {
	actions := u.snapshotActions()
	u.appendLog(
		fmt.Sprintf("[%s] 执行完成: %d 步 x %d 次", nowHMS(), len(actions), loops),
		LogSuccess,
	)
	u.setStatus("执行完成")
	u.setRunningUI(false)
}

func (u *UI) onExecutionStopped() {
	u.appendLog(fmt.Sprintf("[%s] 用户停止执行", nowHMS()), LogWarning)
	u.setStatus("状态：已停止")
	u.setRunningUI(false)
}

// ---------- UI 状态切换 ----------
func (u *UI) setRunningUI(running bool) {
	if running {
		u.startBtn.Disable()
		u.stopBtn.Enable()
		for _, b := range u.presetButtons {
			b.Disable()
		}
		u.clearBtn.Disable()
		u.intervalEntry.Disable()
		u.delayEntry.Disable()
		u.loopsEntry.Disable()
		return
	}
	u.startBtn.Enable()
	u.stopBtn.Disable()
	for _, b := range u.presetButtons {
		b.Enable()
	}
	u.clearBtn.Enable()
	u.intervalEntry.Enable()
	u.delayEntry.Enable()
	u.loopsEntry.Enable()
}

// ---------- 辅助 ----------
func (u *UI) snapshotActions() []Action {
	u.actionsMu.Lock()
	defer u.actionsMu.Unlock()
	snap := make([]Action, len(u.actions))
	copy(snap, u.actions)
	return snap
}

func (u *UI) updateCount() {
	n := len(u.snapshotActions())
	u.countLabel.SetText(fmt.Sprintf("已添加：%d 个行为", n))
}

func (u *UI) setStatus(msg string) {
	u.statusLabel.SetText("状态：" + msg)
}

func (u *UI) appendLog(msg string, level LogLevel) {
	var style widget.RichTextStyle
	switch level {
	case LogError:
		style.ColorName = theme.ColorNameError
	case LogSuccess:
		style.ColorName = theme.ColorNameSuccess
	case LogWarning:
		style.ColorName = theme.ColorNameWarning
	default:
		style.ColorName = theme.ColorNamePrimary
	}
	u.logBox.Segments = append(u.logBox.Segments, &widget.TextSegment{
		Text:  msg + "\n",
		Style: style,
	})
	u.logBox.Refresh()
	if u.logScroll != nil {
		u.logScroll.ScrollToBottom()
	}
}

func (u *UI) parseParams() (time.Duration, time.Duration, int, error) {
	parseFloat := func(s string) (float64, error) {
		s = strings.TrimSpace(s)
		if s == "" {
			return 0, errors.New("不能为空")
		}
		return strconv.ParseFloat(s, 64)
	}
	parseInt := func(s string) (int, error) {
		s = strings.TrimSpace(s)
		if s == "" {
			return 0, errors.New("不能为空")
		}
		return strconv.Atoi(s)
	}

	iv, err := parseFloat(u.intervalEntry.Text)
	if err != nil || iv < 0 {
		return 0, 0, 0, errors.New("动作间隔必须是 >= 0 的数字")
	}
	dv, err := parseFloat(u.delayEntry.Text)
	if err != nil || dv < 0 {
		return 0, 0, 0, errors.New("循环延迟必须是 >= 0 的数字")
	}
	lv, err := parseInt(u.loopsEntry.Text)
	if err != nil || lv < 0 {
		return 0, 0, 0, errors.New("循环次数必须是 >= 0 的整数")
	}
	return time.Duration(iv * float64(time.Second)),
		time.Duration(dv * float64(time.Second)),
		lv,
		nil
}

func numericValidator() fyne.StringValidator {
	return func(s string) error {
		s = strings.TrimSpace(s)
		if s == "" {
			return errors.New("不能为空")
		}
		if _, err := strconv.ParseFloat(s, 64); err != nil {
			return errors.New("必须是数字")
		}
		return nil
	}
}
