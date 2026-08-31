package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// compactTheme 在默认主题基础上缩小字号与内边距，让整体界面更紧凑
type compactTheme struct {
	fyne.Theme
}

func newCompactTheme() fyne.Theme {
	return &compactTheme{Theme: theme.DefaultTheme()}
}

func (t *compactTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNameText:
		return 11
	case theme.SizeNamePadding:
		return 2
	case theme.SizeNameInnerPadding:
		return 4
	case theme.SizeNameLineSpacing:
		return 2
	case theme.SizeNameInputRadius:
		return 3
	}
	return t.Theme.Size(name)
}

// 占位以满足接口（默认主题已提供，此处仅覆盖 Size）
var _ fyne.Theme = (*compactTheme)(nil)

// tinyTheme 字号与全局一致，仅进一步缩小内边距，让快捷按键区按钮更矮（配合 ThemeOverride）
type tinyTheme struct {
	compactTheme
}

func newTinyTheme() fyne.Theme {
	return &tinyTheme{compactTheme{Theme: theme.DefaultTheme()}}
}

func (t *tinyTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNamePadding:
		return 1
	case theme.SizeNameInnerPadding:
		return 2
	}
	return t.compactTheme.Size(name)
}

// bigButtonTheme 局部放大内边距，用于「开始/结束」按钮（配合 ThemeOverride）
type bigButtonTheme struct {
	compactTheme
}

func newBigButtonTheme() fyne.Theme {
	return &bigButtonTheme{compactTheme{Theme: theme.DefaultTheme()}}
}

func (t *bigButtonTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNamePadding:
		return 6
	case theme.SizeNameInnerPadding:
		return 12
	}
	return t.compactTheme.Size(name)
}

// redTitle 小号红色区块标题，替代 Card 自带的大标题
func redTitle(text string) fyne.CanvasObject {
	lbl := widget.NewLabel(text)
	lbl.Importance = widget.DangerImportance
	return lbl
}
