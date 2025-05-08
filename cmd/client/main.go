package main

import (
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"

	"CryptoMessenger/cmd/client/grpc_client"
	"CryptoMessenger/cmd/client/ui"
)

type appNavigator struct {
	app        fyne.App
	window     fyne.Window
	chatClient *grpc_client.ChatClient
}

func (nav *appNavigator) ShowAuth() {
	nav.window.SetMainMenu(nil) // сбрасываем меню
	authWin := ui.NewAuthWindow(nav.window, nav.chatClient, func(userID string) {
		nav.ShowHome(userID)
	})
	authWin.Show()
}

func (nav *appNavigator) ShowHome(userID string) {
	// Объявляем переменную заранее
	var homeWin *ui.HomeWindow

	// Инициализируем homeWin, передавая колбэк с использованием указателя
	homeWin = ui.NewHomeWindow(
		nav.window,
		nav.chatClient,
		userID,
		func() { nav.ShowAuth() }, // onLogout
		func(r fyne.URIReadCloser) {
			if homeWin != nil { // Проверка на nil для безопасности
				homeWin.ChangeBackground(r)
			}
		},
	)

	homeWin.Show()
	homeWin.ShowChats()

	// Создаем меню с методами homeWin
	menu := ui.NewMainMenu(nav.app, nav.window, ui.MenuCallbacks{
		Logout:           func() { nav.ShowAuth() },
		ShowChats:        homeWin.ShowChats,
		ShowInvites:      homeWin.ShowInvites,
		NewChat:          homeWin.ShowNewChat,
		ShowHistory:      homeWin.ShowHistory,
		ChangeBackground: homeWin.ChangeBackground,
	})
	nav.window.SetMainMenu(menu)
}

func main() {
	a := app.New()
	w := a.NewWindow("CryptoMessenger")
	w.Resize(fyne.NewSize(800, 600))

	chatClient, err := grpc_client.NewChatClient("localhost:50051")
	if err != nil {
		log.Fatalf("cannot connect to server: %v", err)
	}
	defer chatClient.Close()

	nav := &appNavigator{app: a, window: w, chatClient: chatClient}
	nav.ShowAuth()
	w.ShowAndRun()
}
