package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"image/color"
)

type DarkTextTheme struct{}

func (DarkTextTheme) Color(n fyne.ThemeColorName, v fyne.ThemeVariant) color.Color {
	if n == theme.ColorNameForeground {
		return color.NRGBA{R: 0, G: 0, B: 0, A: 255}
	}

	// во всех остальных случаях — дефолт из светлой темы
	return theme.LightTheme().Color(n, v)
}

func (DarkTextTheme) Icon(n fyne.ThemeIconName) fyne.Resource {
	return theme.LightTheme().Icon(n)
}
func (DarkTextTheme) Font(s fyne.TextStyle) fyne.Resource {
	return theme.LightTheme().Font(s)
}
func (DarkTextTheme) Size(n fyne.ThemeSizeName) float32 {
	return theme.LightTheme().Size(n)
}

type TransparentButton struct {
	widget.BaseWidget
	OnTapped func()
}

func NewTransparentButton(onTap func()) *TransparentButton {
	btn := &TransparentButton{OnTapped: onTap}
	btn.ExtendBaseWidget(btn)
	return btn
}

func (b *TransparentButton) CreateRenderer() fyne.WidgetRenderer {
	// Без фона и без текста
	return widget.NewSimpleRenderer(container.NewWithoutLayout())
}

func (b *TransparentButton) Tapped(_ *fyne.PointEvent) {
	if b.OnTapped != nil {
		b.OnTapped()
	}
}
