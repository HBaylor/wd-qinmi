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

// Font 返回嵌入的 CJK 字体，Fyne 2.4 无系统字体回退，缺中文会显示方框
func (t *compactTheme) Font(style fyne.TextStyle) fyne.Resource {
	return cjkFont
}

// 占位以满足接口（默认主题已提供，此处仅覆盖 Size）
var _ fyne.Theme = (*compactTheme)(nil)

// redTitle 小号红色区块标题，替代 Card 自带的大标题
func redTitle(text string) fyne.CanvasObject {
	lbl := widget.NewLabel(text)
	lbl.Importance = widget.DangerImportance
	return lbl
}
