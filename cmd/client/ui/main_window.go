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
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"image/color"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type MainWindow struct {
	window            fyne.Window
	chatClient        *grpc_client.ChatClient
	currentChat       string
	chatNameLabel     *widget.Label
	userName          string
	leftPanelContent  *fyne.Container
	rightPanelContent *fyne.Container
	rightEmptyBox     *fyne.Container
	chatHistory       *fyne.Container
	chatScroll        *container.Scroll
	messageInput      *widget.Entry
	sendButton        *widget.Button
	attachButton      *widget.Button
	cancelButton      *widget.Button
	progressBar       *widget.ProgressBar
	cancelSending     context.CancelFunc
	onLogout          func()
}

func NewMainWindow(w fyne.Window, chatClient *grpc_client.ChatClient, name string, onLogout func()) *MainWindow {
	return &MainWindow{
		window:     w,
		chatClient: chatClient,
		userName:   name,
		onLogout:   onLogout,
	}
}

func (m *MainWindow) Show() {
	// Размеры окна
	m.window.Resize(fyne.NewSize(800, 600))

	go m.checkInvitationsPeriodically()
	go m.checkInvitationResponsesPeriodically()
	go m.checkClearChatRequestsPeriodically()
	go m.getMessages()
	go m.refreshChat()

	// Фоновая картинка
	bgImage := canvas.NewImageFromFile("cmd/client/ui/test.jpg")
	bgImage.FillMode = canvas.ImageFillStretch
	dim := canvas.NewRectangle(color.NRGBA{R: 255, G: 255, B: 255, A: 30})
	dim.SetMinSize(fyne.NewSize(800, 600))

	var selectedFileLabel *widget.Label
	var selectedFilePath string
	selectedFileLabel = widget.NewLabel("")
	selectedFileLabel = widget.NewLabel("")
	selectedFileLabel.Wrapping = fyne.TextTruncate

	removeAttachmentButton := widget.NewButtonWithIcon("", theme.CancelIcon(), func() {
		selectedFilePath = ""
		selectedFileLabel.SetText("")
	})
	removeAttachmentButton.Importance = widget.LowImportance

	attachmentBox := container.NewBorder(
		nil, nil,
		nil, removeAttachmentButton, // кнопка справа
		selectedFileLabel, // сам текст по центру
	)

	welcomeText := widget.NewLabelWithStyle("Выберите чат слева", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	welcomeImage := canvas.NewImageFromFile("cmd/client/ui/welcome.png")
	welcomeImage.FillMode = canvas.ImageFillContain
	welcomeImage.SetMinSize(fyne.NewSize(400, 300))

	vbox := container.NewVBox(
		welcomeText,
		welcomeImage,
	)

	m.rightEmptyBox = container.NewCenter(vbox)

	m.leftPanelContent = container.NewVBox()
	//m.refreshChatList()
	leftScroll := container.NewVScroll(m.leftPanelContent)
	leftScroll.SetMinSize(fyne.NewSize(300, 0))

	m.chatHistory = container.NewVBox()

	m.chatScroll = container.NewVScroll(m.chatHistory)
	m.chatScroll.SetMinSize(fyne.NewSize(500, 0))

	m.messageInput = widget.NewMultiLineEntry()
	m.messageInput.SetPlaceHolder("Введите сообщение...")

	m.messageInput.Wrapping = fyne.TextWrapWord // Включить перенос слов

	m.attachButton = widget.NewButtonWithIcon("", theme.FolderOpenIcon(), func() {
		dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil {
				dialog.ShowError(err, m.window)
				return
			}
			if reader == nil {
				return
			}

			selectedFilePath = reader.URI().Path()
			filename := filepath.Base(selectedFilePath)
			selectedFileLabel.SetText("📎 " + filename)

		}, m.window).Show()
	})

	m.progressBar = widget.NewProgressBar()
	m.progressBar.Hide() // сначала скрыт

	m.cancelButton = widget.NewButton("Отменить", func() {
		if m.cancelSending != nil {
			m.cancelSending()
			m.cancelSending = nil
		}
	})
	m.cancelButton.Hide()

	m.sendButton = widget.NewButton("Отправить", func() {
		text := strings.TrimSpace(m.messageInput.Text)

		if text == "" && selectedFilePath == "" {
			dialog.ShowError(fmt.Errorf("нельзя отправить пустое сообщение и без файла"), m.window)
			return
		}

		m.progressBar.SetValue(0)
		m.progressBar.Show()
		m.cancelButton.Show()

		ctx, cancel := context.WithCancel(context.Background())
		m.cancelSending = cancel

		go func() {
			progressFunc := func(done, total int) {
				if total == 0 {
					return
				}
				progress := float64(done) / float64(total)

				fyne.DoAndWait(func() {
					m.progressBar.SetValue(progress)
					if progress >= 0.9 {
						m.cancelButton.Hide()
					}
				})
			}

			err := m.chatClient.SendMessage(ctx, m.currentChat, text, selectedFilePath, progressFunc)

			fyne.DoAndWait(func() {
				m.progressBar.Hide()
				m.cancelButton.Hide()
				m.cancelSending = nil

				if err != nil && !errors.Is(err, context.Canceled) {
					dialog.ShowError(fmt.Errorf("ошибка отправки: %w", err), m.window)
				} else {
					m.messageInput.SetText("")
					selectedFileLabel.SetText("")
					selectedFilePath = ""
				}
			})
		}()
	})

	inputControls := container.NewHBox(m.attachButton, layout.NewSpacer(), m.sendButton)
	inputBox := container.NewVBox(m.messageInput, attachmentBox, m.cancelButton, m.progressBar, inputControls)

	// Сформировать rightPanelContent один раз
	m.rightPanelContent = container.NewBorder(
		nil,      // top
		inputBox, // bottom
		nil, nil, // left, right
		m.chatScroll, // center
	)

	createChatBtn := widget.NewButtonWithIcon("", theme.ContentAddIcon(), m.openNewChatDialog)
	createChatBtn.Importance = widget.LowImportance
	createChatBtn.Alignment = widget.ButtonAlignCenter

	exitBtn := widget.NewButtonWithIcon("", theme.AccountIcon(), func() {
		if m.cancelSending != nil {
			m.cancelSending()
		}
		m.window.Hide()
		if m.onLogout != nil {
			m.onLogout()
		}
	})
	exitBtn.Importance = widget.LowImportance
	exitBtn.Alignment = widget.ButtonAlignCenter
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

	m.chatNameLabel = widget.NewLabelWithStyle("", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	// Панель сверху: create слева, spacer, settings и theme справа

	homeBtn := widget.NewButtonWithIcon("", theme.HomeIcon(), func() {
		m.chatNameLabel.SetText("")
		m.rightPanelContent.Hide()
		m.rightEmptyBox.Show()
	})
	homeBtn.Importance = widget.LowImportance
	homeBtn.Alignment = widget.ButtonAlignCenter

	deleteHistoryBtn := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
		if m.currentChat == "" {
			dialog.ShowError(errors.New("вы не можете удалить диалог, пока не находитесь в нем"), m.window)
			return
		}

		var dlg dialog.Dialog // создаем переменную заранее

		content := container.NewVBox(
			widget.NewLabel("Выберите действие:"),
			widget.NewButton("Удалить только у меня", func() {
				go func() {
					err := m.chatClient.ClearMyChatHistory(m.currentChat)
					if err != nil {
						fyne.DoAndWait(func() {
							dialog.ShowError(err, m.window)
						})
						return
					}
					fyne.DoAndWait(func() {
						m.chatHistory.Objects = nil
						m.chatHistory.Refresh()
						dlg.Hide() // закрываем диалог
					})
				}()
			}),
			widget.NewButton("Удалить у всех", func() {
				go func() {
					err := m.chatClient.ClearChatHistory(m.currentChat)
					if err != nil {
						fyne.DoAndWait(func() {
							dialog.ShowError(err, m.window)
						})
						return
					}
					fyne.DoAndWait(func() {
						m.chatHistory.Objects = nil
						m.chatHistory.Refresh()
						dlg.Hide()
					})
				}()
			}),
		)

		dlg = dialog.NewCustom("Удалить историю чата", "Закрыть", content, m.window)
		dlg.Show()
	})

	deleteHistoryBtn.Importance = widget.LowImportance
	deleteHistoryBtn.Alignment = widget.ButtonAlignCenter

	topBar := container.New(
		layout.NewHBoxLayout(),
		createChatBtn,
		layout.NewSpacer(),
		m.chatNameLabel,
		layout.NewSpacer(),
		deleteHistoryBtn,
		homeBtn,
		exitBtn,
		themeBtn,
		updateChatsList,
	)

	rightStack := container.NewStack(m.rightEmptyBox, m.rightPanelContent)
	m.rightPanelContent.Hide()

	// Собираем основной сплит
	split := container.NewHSplit(leftScroll, rightStack)
	split.Offset = 0.3
	body := container.New(layout.NewStackLayout(), split)

	// Оверлей: topBar сверху и body по центру
	overlay := container.NewBorder(topBar, nil, nil, nil, body)

	m.refreshChatList()

	// Финальный вид: фон, затемнение, overlay
	m.window.SetContent(container.NewMax(
		bgImage,
		dim,
		overlay,
	))
	m.window.Show()
}

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
			m.chatNameLabel.SetText(info.Name)
			m.rightEmptyBox.Hide()
			m.rightPanelContent.Show()
			m.loadCurrentChat()
			m.chatScroll.ScrollToBottom()
			m.messageInput.SetText("")
		})

		m.leftPanelContent.Add(btn)
	}
	fyne.Do(func() {
		m.leftPanelContent.Refresh()
	})
}

func (m *MainWindow) loadCurrentChat() {
	chatPath := filepath.Join("cmd", "client", "users", m.chatClient.UserID, "chats", m.currentChat, "chat.jsonl")

	data, err := os.ReadFile(chatPath)
	if err != nil {
		m.chatHistory.Objects = []fyne.CanvasObject{
			widget.NewLabel("Ошибка открытия чата: " + err.Error()),
		}
		m.chatHistory.Refresh()
		return
	}

	lines := strings.Split(string(data), "\n")
	var messages []fyne.CanvasObject

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		var msg domain.StoredMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			continue
		}

		switch msg.Type {
		case "text":
			label := widget.NewLabel(fmt.Sprintf("[%s] %s: %s", msg.Timestamp.Format(time.DateTime), msg.Sender, msg.Content))
			label.Wrapping = fyne.TextWrapWord
			messages = append(messages, label)

		case "file":
			fileLabel := fmt.Sprintf("[%s] %s отправил файл: %s", msg.Timestamp.Format(time.DateTime), msg.Sender, msg.Filename)
			filePath := filepath.Join(msg.Filepath)

			if _, err := os.Stat(filePath); err == nil {
				ext := strings.ToLower(filepath.Ext(filePath))
				label := widget.NewLabel(fileLabel)
				label.Wrapping = fyne.TextWrapWord

				switch ext {
				case ".png", ".jpg", ".jpeg", ".gif":
					img := canvas.NewImageFromFile(filePath)
					img.FillMode = canvas.ImageFillContain
					img.SetMinSize(fyne.NewSize(200, 200))

					tapImgObj := NewTransparentButton(func() {
						fullImg := canvas.NewImageFromFile(filePath)
						fullImg.FillMode = canvas.ImageFillContain
						fullImg.SetMinSize(fyne.NewSize(600, 600))

						w := fyne.CurrentApp().NewWindow(msg.Filename)
						w.SetContent(container.NewMax(fullImg))
						w.Resize(fyne.NewSize(800, 800))
						w.Show()
					})

					imgWithClick := container.NewMax(img, tapImgObj)
					messages = append(messages, label, imgWithClick)

				default:
					// Кнопка для других типов файлов
					uri := storage.NewFileURI(filePath)
					openBtn := widget.NewButtonWithIcon("Открыть файл", theme.FileIcon(), func() {
						f, err := storage.OpenFileFromURI(uri)
						if err != nil {
							dialog.ShowError(err, m.window)
							return
						}
						defer f.Close()

						var openCmd *exec.Cmd
						switch runtime.GOOS {
						case "linux":
							openCmd = exec.Command("xdg-open", filePath)
						case "windows":
							openCmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", filePath)
						case "darwin":
							openCmd = exec.Command("open", filePath)
						}
						if openCmd != nil {
							_ = openCmd.Start()
						}
					})
					openBtn.Importance = widget.LowImportance
					openBtn.Resize(fyne.NewSize(30, 30)) // маленькая кнопка

					messages = append(messages, label, openBtn)
				}
			} else {
				label := widget.NewLabel(fileLabel + " (файл не найден)")
				label.Wrapping = fyne.TextWrapWord
				messages = append(messages, label)
			}
		default:
			// Неизвестный тип сообщения — игнорируем или логируем
		}
	}

	m.chatHistory.Objects = messages
	m.chatHistory.Refresh()
	m.chatScroll.ScrollToBottom()
}

func (m *MainWindow) openNewChatDialog() {
	chatNameEntry := widget.NewEntry()
	receiverEntry := widget.NewEntry()
	algorithmSelect := widget.NewSelect([]string{"RC5", "RC6"}, nil)
	modeSelect := widget.NewSelect([]string{"ECB", "CBC", "PCBC", "CFB", "OFB", "CTR", "RandomDelta"}, nil)
	paddingSelect := widget.NewSelect([]string{"Zeros", "ANSIX923", "PKCS7", "ISO10126"}, nil)
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

func (m *MainWindow) getMessages() {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			progressFunc := func(done, total int) {
				if total == 0 {
					return
				}
				progress := float64(done) / float64(total)
				fyne.DoAndWait(func() {
					m.progressBar.SetValue(progress)
					m.progressBar.Show()
				})
			}

			err := m.chatClient.ReceiveMessage(m.currentChat, progressFunc)
			if err != nil {
				slog.Error(err.Error())
			} else {
				fyne.DoAndWait(func() {
					m.progressBar.Hide()
					m.progressBar.SetValue(0)
				})
			}
		}
	}
}

func (m *MainWindow) refreshChat() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if _, ok := m.chatClient.Messages.LoadAndDelete(m.currentChat); ok {
				fyne.DoAndWait(func() {
					m.loadCurrentChat()
				})
			}
		}
	}
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
				fyne.DoAndWait(func() {
					m.showInvitationDialog(inv)
				})
			}
		}
	}
}

func (m *MainWindow) checkClearChatRequestsPeriodically() {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if m.currentChat != "" {
				err := m.chatClient.ReceiveClearChatHistoryRequest(m.currentChat)
				if err != nil {
					if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, domain.ErrNotFound) {
						continue
					}
					continue
				}
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
				switch resp.Accepted {
				case true:
					fyne.DoAndWait(func() {
						m.showSuccessInvitationResponseDialog(resp)
					})
				case false:
					fyne.DoAndWait(func() {
						m.showRejectedInvitationResponseDialog(resp)
					})
				}
			}
			fyne.DoAndWait(func() {
				m.refreshChatList()
			})
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
			widget.NewLabel(fmt.Sprintf("Комната: %s", inv.RoomName)),
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
					fmt.Sprintf("Вы присоединились к комнате %s", inv.RoomName),
					m.window,
				)
				m.refreshChatList()
				// Можно обновить список чатов или выполнить другие действия
			} else {
				if err = os.RemoveAll(filepath.Join("cmd", "client", "users", inv.Receiver, "chats", inv.RoomID)); err != nil {
					slog.Error("Error", err)
				}
				dialog.ShowInformation(
					"Приглашение отклонено",
					fmt.Sprintf("Вы отклонили приглашение в комнату %s", inv.RoomName),
					m.window,
				)
			}
		},
		m.window,
	)
}

func (m *MainWindow) showSuccessInvitationResponseDialog(resp domain.Invitation) {
	content := container.NewVBox(
		widget.NewLabel(fmt.Sprintf("Ответ от: %s", resp.Sender)),
		widget.NewLabel("Общий ключ успешно сгенерирован!"),
	)

	dialog.ShowCustom(
		"Ответ на приглашение",
		"OK",
		content,
		m.window,
	)
}

func (m *MainWindow) showRejectedInvitationResponseDialog(resp domain.Invitation) {
	content := container.NewVBox(
		widget.NewLabel(fmt.Sprintf("Ответ от: %s", resp.Sender)),
		widget.NewLabel("Пользователь не хочет с вами общаться!"),
	)

	dialog.ShowCustom(
		"Ответ на приглашение",
		"Плаки-плаки",
		content,
		m.window,
	)
}
