package main

import (
	_ "embed"

	"fyne.io/fyne/v2"
)

//go:embed assets/NotoSansSC-subset.otf
var cjkFontData []byte

// cjkFont 覆盖默认主题的正文字体，解决 Fyne 2.4 无系统字体回退导致的中文方框问题。
// 字体为思源黑体（Noto Sans SC）子集：ASCII + GB2312 全部汉字 + 中日韩符号。
var cjkFont = fyne.NewStaticResource("NotoSansSC.ttf", cjkFontData)
