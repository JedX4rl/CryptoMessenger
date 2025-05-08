package ui

import (
	"CryptoMessenger/cmd/client/grpc_client"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
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
	w.SetFixedSize(true)
	return &AuthWindow{
		chatClient: chatClient,
		window:     w,
		onSuccess:  onSuccess,
		progress:   widget.NewProgressBarInfinite(),
	}
}

func (a *AuthWindow) Show() {
	card := widget.NewCard(
		"🔐 Welcome to CryptoMessenger",
		"Secure your conversations with end-to-end encryption",
		nil,
	)

	// Поля ввода
	entryName := widget.NewEntry()
	entryName.SetPlaceHolder("Enter your username or existing UserID")

	entryPassword := widget.NewPasswordEntry()
	entryPassword.SetPlaceHolder("Enter your password")

	// Валидация полей
	validateFields := func() error {
		if entryName.Text == "" {
			return fmt.Errorf("username cannot be empty")
		}
		if entryPassword.Text == "" {
			return fmt.Errorf("password cannot be empty")
		}
		return nil
	}

	// Стильные кнопки
	btnRegister := widget.NewButtonWithIcon("Register", theme.ContentAddIcon(), func() {
		if err := validateFields(); err != nil {
			dialog.ShowError(err, a.window)
			return
		}

		progressDialog := dialog.NewCustom("Registering...", "Cancel", a.progress, a.window)
		progressDialog.Show()

		go func() {
			err := a.chatClient.RegisterUser(entryName.Text, entryPassword.Text)

			progressDialog.Hide()

			if err != nil {
				dialog.ShowError(err, a.window)
				return
			}

			dialog.ShowInformation(
				"Registration Successful",
				fmt.Sprintf("Your're in!"),
				a.window,
			)
		}()
	})

	btnLogin := widget.NewButtonWithIcon("Login", theme.LoginIcon(), func() {
		if err := validateFields(); err != nil {
			dialog.ShowError(err, a.window)
			return
		}

		progressDialog := dialog.NewCustom("Authenticating...", "Cancel", a.progress, a.window)
		progressDialog.Show()

		go func() {
			err := a.chatClient.LoginUser(entryName.Text, entryPassword.Text)

			progressDialog.Hide()

			if err != nil {
				dialog.ShowError(fmt.Errorf("Invalid credentials. Please check and try again."), a.window)
				return
			}

			dialog.ShowInformation(
				"Welcome Back!",
				"Authentication successful",
				a.window,
			)
			a.onSuccess(entryName.Text) // Используем имя пользователя как ID
		}()
	})

	// Красивое расположение элементов
	form := container.NewVBox(
		widget.NewLabelWithStyle("Get started with CryptoMessenger", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		layout.NewSpacer(),
		entryName,
		entryPassword,
		layout.NewSpacer(),
		container.NewGridWithColumns(
			2,
			btnRegister,
			btnLogin,
		),
	)

	card.SetContent(form)
	centered := container.NewCenter(card)
	a.window.SetContent(centered)
	a.window.Show()
}
