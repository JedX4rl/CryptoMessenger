package main

import (
	"CryptoMessenger/cmd/client/ui"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"

	"CryptoMessenger/cmd/client/grpc_client"
)

func main() {
	a := app.New()
	w := a.NewWindow("CryptoMessenger")
	w.Resize(fyne.NewSize(800, 600))

	chatClient, err := grpc_client.NewChatClient("localhost:50051")
	if err != nil {
		log.Fatalf("cannot connect to server: %v", err)
	}
	defer chatClient.Close()
	auth := ui.NewAuthWindow(w, chatClient, func(userID string) {
		w.Hide()
		newWindow := ui.NewMainWindow(w, chatClient, userID)
		newWindow.Show()
	})
	auth.Show()

	w.ShowAndRun()
}
