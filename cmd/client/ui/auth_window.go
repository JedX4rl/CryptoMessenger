package ui

import (
	"CryptoMessenger/cmd/client/grpc_client"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"image/color"
)

type AuthWindow struct {
	chatClient *grpc_client.ChatClient
	window     fyne.Window
	onSuccess  func(userID string)
	progress   *widget.ProgressBarInfinite
}

func NewAuthWindow(w fyne.Window, chatClient *grpc_client.ChatClient, onSuccess func(userID string)) *AuthWindow {
	w.SetTitle("CryptoMessenger - Authentication")
	w.Resize(fyne.NewSize(800, 400))
	return &AuthWindow{
		chatClient: chatClient,
		window:     w,
		onSuccess:  onSuccess,
		progress:   widget.NewProgressBarInfinite(),
	}
}

func (a *AuthWindow) Show() {
	// 1) основное и минимальное размеры окна через прозрачный Rect
	a.window.Resize(fyne.NewSize(800, 400))
	minSize := canvas.NewRectangle(color.Transparent)
	minSize.SetMinSize(fyne.NewSize(800, 400))
	fyne.CurrentApp().Settings().SetTheme(theme.DarkTheme())

	bg := canvas.NewImageFromFile("cmd/client/ui/test.jpg")
	bg.FillMode = canvas.ImageFillStretch

	// 3) кнопка темы, обёрнутая в HBoxLayout, чтобы не растягивалась
	isDark := true
	var themeBtn *widget.Button

	themeBtn = widget.NewButtonWithIcon("", theme.ColorPaletteIcon(), func() {
		if isDark {
			fyne.CurrentApp().Settings().SetTheme(DarkTextTheme{})
		} else {
			fyne.CurrentApp().Settings().SetTheme(theme.DarkTheme())
		}
		isDark = !isDark
	})

	themeBtn.Importance = widget.LowImportance
	themeBtn.Alignment = widget.ButtonAlignCenter

	topBtn := container.New(layout.NewHBoxLayout(), layout.NewSpacer(), themeBtn)

	// 4) поля формы
	entryName := widget.NewEntry()
	entryName.SetPlaceHolder("Enter your username or UserID")
	entryPassword := widget.NewPasswordEntry()
	entryPassword.SetPlaceHolder("Enter your password")
	validate := func() error {
		if entryName.Text == "" {
			return fmt.Errorf("username cannot be empty")
		}
		if entryPassword.Text == "" {
			return fmt.Errorf("password cannot be empty")
		}
		return nil
	}

	btnReg := widget.NewButtonWithIcon("Register", theme.ContentAddIcon(), func() {
		if err := validate(); err != nil {
			dialog.ShowError(err, a.window)
			return
		}
		pd := dialog.NewCustom("Registering...", "Cancel", a.progress, a.window)
		pd.Show()
		go func() {
			err := a.chatClient.RegisterUser(entryName.Text, entryPassword.Text)
			pd.Hide()
			if err != nil {
				dialog.ShowError(err, a.window)
				return
			}
			info := dialog.NewInformation("Registration Successful", "You're in!", a.window)
			info.SetOnClosed(func() { a.onSuccess(entryName.Text) })
			info.Show()
		}()
	})
	btnLog := widget.NewButtonWithIcon("Login", theme.LoginIcon(), func() {
		if err := validate(); err != nil {
			dialog.ShowError(err, a.window)
			return
		}
		pd := dialog.NewCustom("Authenticating...", "Cancel", a.progress, a.window)
		pd.Show()
		go func() {
			err := a.chatClient.LoginUser(entryName.Text, entryPassword.Text)
			fyne.DoAndWait(func() { pd.Hide() })
			if err != nil {
				dialog.ShowError(fmt.Errorf("Invalid credentials"), a.window)
				return
			}
			info := dialog.NewInformation("Welcome Back!", "Authentication successful", a.window)
			info.SetOnClosed(func() { a.onSuccess(entryName.Text) })
			info.Show()
		}()
	})

	// 5) собираем форму
	form := container.NewVBox(
		widget.NewLabelWithStyle("🔐 Welcome to CryptoMessenger", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("Secure your conversations with end-to-end encryption", fyne.TextAlignCenter, fyne.TextStyle{}),
		layout.NewSpacer(),
		entryName,
		entryPassword,
		layout.NewSpacer(),
		container.NewGridWithColumns(2, btnReg, btnLog),
	)

	// 6) имитируем Card, но с фоновой Rect и заданной прозрачностью
	//    — для полностью непрозрачной карточки: A = 255
	//    — для 30% прозрачности: A ≈ 77
	cardBg := canvas.NewRectangle(color.NRGBA{R: 255, G: 255, B: 255, A: 77})
	cardBg.SetMinSize(fyne.NewSize(400, 280))

	cardContent := container.NewMax(cardBg, container.NewPadded(form))

	// 7) собираем overlay: кнопка сверху-лево, карточка по центру
	overlay := container.NewBorder(
		topBtn, // top
		nil,    // bottom
		nil,    // left
		nil,    // right
		container.NewCenter(cardContent),
	)

	// 8) финальный стек: сначала minSize, потом фон, потом overlay
	a.window.SetContent(container.NewMax(
		minSize,
		bg,
		overlay,
	))

	a.window.Show()
}
