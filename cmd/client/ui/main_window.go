package ui

import (
	"CryptoMessenger/cmd/client/domain"
	"CryptoMessenger/cmd/client/grpc_client"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"image/color"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type MainWindow struct {
	window            fyne.Window
	chatClient        *grpc_client.ChatClient
	currentChat       string
	userID            string
	leftPanelContent  *fyne.Container
	rightPanelContent *fyne.Container
}

func NewMainWindow(w fyne.Window, chatClient *grpc_client.ChatClient, userID string) *MainWindow {
	return &MainWindow{
		window:     w,
		chatClient: chatClient,
		userID:     userID,
	}
}

func (m *MainWindow) Show() {
	// Размеры окна
	m.window.Resize(fyne.NewSize(800, 600))

	go m.checkInvitationsPeriodically()
	go m.checkInvitationResponsesPeriodically()

	// Фоновая картинка
	bgImage := canvas.NewImageFromFile("cmd/client/ui/test.jpg")
	bgImage.FillMode = canvas.ImageFillStretch
	// Полупрозрачная накладка: начально 30% (A≈77)
	dim := canvas.NewRectangle(color.NRGBA{R: 255, G: 255, B: 255, A: 30})
	dim.SetMinSize(fyne.NewSize(800, 600))

	m.leftPanelContent = container.NewVBox()
	m.refreshChatList()
	leftScroll := container.NewVScroll(m.leftPanelContent)
	leftScroll.SetMinSize(fyne.NewSize(300, 0))

	m.rightPanelContent = container.NewVBox(widget.NewLabel("Здесь будет показан чат при клике"))
	rightScroll := container.NewVScroll(m.rightPanelContent)
	rightScroll.SetMinSize(fyne.NewSize(500, 0))

	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for range ticker.C {
			fyne.DoAndWait(func() {
				if m.currentChat != "" {
					m.loadCurrentChat()
				}
			})
		}
	}()

	// Кнопки создания чата (сверху слева), настройки и темы (сверху справа)
	// Кнопка "+" для нового чата
	createChatBtn := widget.NewButtonWithIcon("", theme.ContentAddIcon(), m.openNewChatDialog)
	createChatBtn.Importance = widget.LowImportance
	createChatBtn.Alignment = widget.ButtonAlignCenter
	// Кнопка настроек
	settingsBtn := widget.NewButtonWithIcon("", theme.SettingsIcon(), func() {
		// логика настроек
	})
	settingsBtn.Importance = widget.LowImportance
	settingsBtn.Alignment = widget.ButtonAlignCenter
	// Кнопка переключения темы и прозрачности фона
	isDark := true

	themeBtn := widget.NewButtonWithIcon("", theme.ColorPaletteIcon(), func() {
		if isDark {
			fyne.CurrentApp().Settings().SetTheme(DarkTextTheme{})
			dim.FillColor = color.NRGBA{R: 255, G: 255, B: 255, A: 179} // 30% затемнения
		} else {
			fyne.CurrentApp().Settings().SetTheme(theme.DarkTheme())
			dim.FillColor = color.NRGBA{R: 255, G: 255, B: 255, A: 30} // 70% затемнения
		}
		isDark = !isDark
		dim.Refresh()
	})

	themeBtn.Importance = widget.LowImportance
	themeBtn.Alignment = widget.ButtonAlignCenter

	updateChatsList := widget.NewButtonWithIcon("", theme.MediaReplayIcon(), func() {
		m.refreshChatList()
	})
	updateChatsList.Importance = widget.LowImportance
	updateChatsList.Alignment = widget.ButtonAlignCenter

	// Панель сверху: create слева, spacer, settings и theme справа
	topBar := container.New(
		layout.NewHBoxLayout(),
		createChatBtn,
		layout.NewSpacer(),
		settingsBtn,
		themeBtn,
		updateChatsList,
	)

	// Собираем основной сплит
	split := container.NewHSplit(leftScroll, rightScroll)
	split.Offset = 0.3
	body := container.New(layout.NewStackLayout(), split)

	// Оверлей: topBar сверху и body по центру
	overlay := container.NewBorder(topBar, nil, nil, nil, body)

	// Финальный вид: фон, затемнение, overlay
	m.window.SetContent(container.NewMax(
		bgImage,
		dim,
		overlay,
	))
	m.window.Show()
}

//func (m *MainWindow) refreshChatList() {
//	m.leftPanelContent.Objects = nil
//
//	for _, name := range listChatNames() {
//		if name == "" {
//			continue
//		}
//		chat := name
//		btn := widget.NewButton(chat, func() {
//			m.currentChat = chat
//			m.loadCurrentChat()
//		})
//		m.leftPanelContent.Add(btn)
//	}
//	m.leftPanelContent.Refresh()
//}

func (m *MainWindow) refreshChatList() {
	m.leftPanelContent.Objects = nil

	baseDir := filepath.Join("cmd", "client", "users", m.chatClient.UserID, "chats")

	entries, err := os.ReadDir(baseDir)
	if err != nil {
		m.leftPanelContent.Refresh()
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		roomID := entry.Name()
		jsonPath := filepath.Join(baseDir, roomID, "room_info.json")

		data, err := os.ReadFile(jsonPath)
		if err != nil {
			continue
		}

		var info domain.RoomInfo
		if err := json.Unmarshal(data, &info); err != nil {
			continue
		}

		btn := widget.NewButton(info.Name, func() {
			m.currentChat = roomID
			m.loadCurrentChat()
		})

		m.leftPanelContent.Add(btn)
	}

	m.leftPanelContent.Refresh()
}

func (m *MainWindow) loadCurrentChat() {
	path := filepath.Join("cmd", "client", "chats", m.currentChat)
	data, err := os.ReadFile(path)
	if err != nil {
		m.rightPanelContent.Objects = []fyne.CanvasObject{
			widget.NewLabel("Ошибка открытия чата: " + err.Error()),
		}
	} else {
		lbl := widget.NewLabel(string(data))
		lbl.Wrapping = fyne.TextWrapWord
		m.rightPanelContent.Objects = []fyne.CanvasObject{lbl}
	}
	m.rightPanelContent.Refresh()
}

// openNewChatDialog открывает диалог создания нового чата
func (m *MainWindow) openNewChatDialog() {
	chatNameEntry := widget.NewEntry()
	receiverEntry := widget.NewEntry()
	algorithmSelect := widget.NewSelect([]string{"RC5", "RC6"}, nil)
	modeSelect := widget.NewSelect([]string{"CFB", "ECB", "RandomDelta"}, nil)
	paddingSelect := widget.NewSelect([]string{"Zeros", "ANSI"}, nil)
	errorLabel := widget.NewLabel("")
	errorLabel.Hide()
	var dlg *dialog.CustomDialog

	form := container.NewVBox(
		widget.NewLabel("Имя чата:"), chatNameEntry,
		widget.NewLabel("Имя собеседника:"), receiverEntry,
		widget.NewLabel("Алгоритм:"), algorithmSelect,
		widget.NewLabel("Режим шифрования:"), modeSelect,
		widget.NewLabel("Набивка:"), paddingSelect,
	)

	onCreate := func() {
		name := strings.TrimSpace(chatNameEntry.Text)
		recv := strings.TrimSpace(receiverEntry.Text)
		if len(name) < 3 || len(name) > 10 {
			errorLabel.SetText("Имя чата должно быть от 3 до 10 символов.")
			errorLabel.Show()
			return
		}
		if recv == "" || algorithmSelect.Selected == "" ||
			modeSelect.Selected == "" || paddingSelect.Selected == "" {
			errorLabel.SetText("Заполните все поля.")
			errorLabel.Show()
			return
		}
		err := m.chatClient.CreateChat(domain.Chat{
			ChatName:  name,
			Receiver:  recv,
			Algorithm: algorithmSelect.Selected,
			Mode:      modeSelect.Selected,
			Padding:   paddingSelect.Selected,
		})
		if err != nil {
			dialog.ShowError(fmt.Errorf("ошибка создания чата: %v", err), m.window)
			return
		}
		dlg.Hide()
		m.refreshChatList()
	}

	cancelBtn := widget.NewButton("Отмена", func() { dlg.Hide() })
	createBtn := widget.NewButton("Создать", onCreate)
	content := container.NewBorder(
		widget.NewLabelWithStyle("Новый чат", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		container.NewHBox(layout.NewSpacer(), cancelBtn, createBtn),
		nil, nil,
		container.NewPadded(form, errorLabel),
	)
	dlg = dialog.NewCustomWithoutButtons("Создание нового чата", content, m.window)
	dlg.Resize(fyne.NewSize(400, 400))
	dlg.Show()
}

func (m *MainWindow) checkInvitationsPeriodically() {
	ticker := time.NewTicker(7 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			inv, err := m.chatClient.ReceiveInvitation()
			if err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					continue
				}
				log.Printf("Error checking invitations: %v", err)
				continue
			}

			if inv.Sender != "" {
				// Показываем диалог в основном потоке
				fyne.DoAndWait(func() {
					m.showInvitationDialog(inv)
				})
			}
		}
	}
}

func (m *MainWindow) checkInvitationResponsesPeriodically() {
	ticker := time.NewTicker(7 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			resp, err := m.chatClient.ReceiveInvitationResponse()
			if err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					continue
				}
				log.Printf("Error checking invitation responses: %v", err)
				continue
			}

			if resp.Sender != "" {
				fyne.DoAndWait(func() {
					m.showInvitationResponseDialog(resp)
				})
			}
		}
	}
}

func (m *MainWindow) showInvitationDialog(inv domain.Invitation) {
	dialog.ShowCustomConfirm(
		"Новое приглашение",
		"Принять",
		"Отклонить",
		container.NewVBox(
			widget.NewLabel(fmt.Sprintf("От: %s", inv.Sender)),
			widget.NewLabel(fmt.Sprintf("Комната: %s", inv.RoomID)),
		),
		func(accepted bool) {
			err := m.chatClient.ReactToInvitation(domain.Invitation{RoomID: inv.RoomID, Receiver: inv.Sender}, accepted)
			if err != nil {
				dialog.ShowError(fmt.Errorf("Ошибка обработки приглашения: %v", err), m.window)
				return
			}

			if accepted {
				dialog.ShowInformation(
					"Приглашение принято",
					fmt.Sprintf("Вы присоединились к комнате %s", inv.RoomID),
					m.window,
				)
				// Можно обновить список чатов или выполнить другие действия
			} else {
				dialog.ShowInformation(
					"Приглашение отклонено",
					fmt.Sprintf("Вы отклонили приглашение в комнату %s", inv.RoomID),
					m.window,
				)
			}
		},
		m.window,
	)
}

func (m *MainWindow) showInvitationResponseDialog(resp domain.Invitation) {
	content := container.NewVBox(
		widget.NewLabel(fmt.Sprintf("Ответ от: %s", resp.Sender)),
		widget.NewLabel(fmt.Sprintf("Комната: %s", resp.RoomID)),
		widget.NewLabel("Общий ключ успешно сгенерирован!"),
		widget.NewButton("Показать ключ", func() {
			dialog.ShowCustom(
				"Общий ключ",
				"Закрыть",
				container.NewVScroll(
					widget.NewLabel(resp.SharedKey),
				),
				m.window,
			)
		}),
	)

	dialog.ShowCustom(
		"Ответ на приглашение",
		"OK",
		content,
		m.window,
	)
}

func listChatNames() []string {
	dir := "./cmd/client/chats.txt"  // путь к файлу
	content, err := os.ReadFile(dir) // читаем содержимое файла
	if err != nil {
		log.Println("Ошибка при чтении чатов:", err)
		return nil
	}

	// Разбиваем содержимое файла на строки (имена чатов)
	chatNames := strings.Split(string(content), "\n")

	return chatNames
}
