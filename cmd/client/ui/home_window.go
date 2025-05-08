package ui

import (
	"CryptoMessenger/cmd/client/grpc_client"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"image"
)

type HomeWindow struct {
	client     *grpc_client.ChatClient
	window     fyne.Window
	userID     string
	onLogout   func()
	onBgChange func(fyne.URIReadCloser)

	bgImage *canvas.Image
	content *fyne.Container
	root    *fyne.Container // Восстанавливаем поле root
}

func NewHomeWindow(w fyne.Window, client *grpc_client.ChatClient, userID string, onLogout func(), onBgChange func(fyne.URIReadCloser)) *HomeWindow {
	bg := canvas.NewImageFromImage(image.NewRGBA(image.Rect(0, 0, 1, 1)))
	bg.FillMode = canvas.ImageFillStretch

	content := container.NewMax()
	root := container.NewMax(bg, content) // Инициализируем root

	return &HomeWindow{
		client:     client,
		window:     w,
		userID:     userID,
		onLogout:   onLogout,
		onBgChange: onBgChange,
		bgImage:    bg,
		content:    content,
		root:       root, // Добавляем root в структуру
	}
}

func (h *HomeWindow) createSettingsAction() *widget.ToolbarAction {
	return widget.NewToolbarAction(theme.SettingsIcon(), func() {
		// Получаем позицию кнопки настроек
		pos := fyne.CurrentApp().Driver().AbsolutePositionForObject(h.window.Canvas().Content())

		menu := widget.NewPopUpMenu(
			fyne.NewMenu("",
				fyne.NewMenuItem("Change Background", h.showBgPicker),
				fyne.NewMenuItem("Logout", h.onLogout),
			),
			h.window.Canvas(),
		)
		menu.ShowAtPosition(pos)
	})
}

// Остальные методы остаются без изменений

func (h *HomeWindow) Show() {
	toolbar := widget.NewToolbar(
		h.createToolbarAction(theme.ViewFullScreenIcon(), "Chats", h.ShowChats),
		h.createToolbarAction(theme.MailReplyAllIcon(), "Invites", h.ShowInvites),
		h.createToolbarAction(theme.ContentAddIcon(), "New Chat", h.ShowNewChat),
		h.createToolbarAction(theme.HistoryIcon(), "History", h.ShowHistory),
		widget.NewToolbarSpacer(),
		h.createSettingsAction(),
	)

	main := container.NewBorder(toolbar, nil, nil, nil,
		container.NewMax(h.bgImage, h.content))
	h.window.SetContent(main)
}

func (h *HomeWindow) createToolbarAction(icon fyne.Resource, tooltip string, action func()) *widget.ToolbarAction {
	return widget.NewToolbarAction(icon, action)
}

func (h *HomeWindow) showBgPicker() {
	dialog.ShowFileOpen(func(uri fyne.URIReadCloser, err error) {
		if err != nil || uri == nil {
			return
		}
		h.onBgChange(uri)
	}, h.window)
}

// Остальные методы ShowChats, ShowInvites и т.д. остаются без изменений
// Остальные методы ShowChats, ShowInvites и т.д. остаются без изменений

// ShowChats показывает список чатов
func (h *HomeWindow) ShowChats() {
	// Пример: пока нет чатов, показываем заглушку
	placeholder := widget.NewLabelWithStyle("You have no chats yet.", fyne.TextAlignCenter, fyne.TextStyle{})
	h.content.Objects = []fyne.CanvasObject{
		container.NewBorder(widget.NewLabelWithStyle("Your Chats", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			nil, nil, nil, placeholder),
	}
	h.content.Refresh()
}

// ShowInvites показывает приглашения
func (h *HomeWindow) ShowInvites() {
	placeholder := widget.NewLabelWithStyle("No pending invites.", fyne.TextAlignCenter, fyne.TextStyle{})
	h.content.Objects = []fyne.CanvasObject{
		container.NewCenter(
			container.NewVBox(
				widget.NewLabelWithStyle("Invitations", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
				placeholder,
			),
		),
	}
	h.content.Refresh()
}

// ShowNewChat показывает форму создания новой комнаты
func (h *HomeWindow) ShowNewChat() {
	entry := widget.NewEntry()
	entry.SetPlaceHolder("Enter new room ID or name")
	btn := widget.NewButton("Create Room", func() {
		// TODO: h.client.CreateRoom(...)
	})
	h.content.Objects = []fyne.CanvasObject{
		container.NewVBox(
			widget.NewLabelWithStyle("Create New Chat", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			entry,
			btn,
		),
	}
	h.content.Refresh()
}

// ShowHistory показывает историю (пока просто заглушка)
func (h *HomeWindow) ShowHistory() {
	placeholder := widget.NewLabelWithStyle("Chat history will appear here.", fyne.TextAlignCenter, fyne.TextStyle{})
	h.content.Objects = []fyne.CanvasObject{
		container.NewBorder(widget.NewLabelWithStyle("History", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			nil, nil, nil, placeholder),
	}
	h.content.Refresh()
}

// ChangeBackground меняет фон приложения
func (h *HomeWindow) ChangeBackground(r fyne.URIReadCloser) {
	defer r.Close()
	img := canvas.NewImageFromReader(r, r.URI().Path())
	img.FillMode = canvas.ImageFillContain
	h.bgImage = img
	// заменяем фон в корне
	h.root.Objects[0] = img
	h.Show() // сбросить окно, чтобы видеть фон
}
