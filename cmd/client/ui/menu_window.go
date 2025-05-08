package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
)

// MenuCallbacks собирает колбэки для каждого пункта меню.
type MenuCallbacks struct {
	Logout           func()
	ShowChats        func()
	ShowInvites      func()
	NewChat          func()
	ShowHistory      func()
	ChangeBackground func(fyne.URIReadCloser)
}

// NewMainMenu строит меню в стиле fyne_demo.
func NewMainMenu(a fyne.App, w fyne.Window, c MenuCallbacks) *fyne.MainMenu {
	// File
	logout := fyne.NewMenuItem("Logout", c.Logout)
	file := fyne.NewMenu("File",
		logout,
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Quit", func() { a.Quit() }),
	)

	// Chat (справа от File)
	chatMenu := fyne.NewMenu("Chat",

		fyne.NewMenuItem("Chats", c.ShowChats),
		fyne.NewMenuItem("Invitations", c.ShowInvites),
		fyne.NewMenuItem("New Chat", c.NewChat),
		fyne.NewMenuItem("History", c.ShowHistory),
	)

	// Settings
	changeBg := fyne.NewMenuItem("Change Background", func() {
		dialog.ShowFileOpen(func(r fyne.URIReadCloser, err error) {
			if r == nil {
				return
			}
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			c.ChangeBackground(r)
		}, w)
	})
	settings := fyne.NewMenu("Settings", changeBg)

	// Help
	about := fyne.NewMenuItem("About", func() {
		dialog.ShowInformation("About CryptoMessenger", "CryptoMessenger v1.0\n© 2025", w)
	})
	help := fyne.NewMenu("Help", about)

	main := fyne.NewMainMenu(
		file,
		chatMenu,
		settings,
		help,
	)
	return main
}
