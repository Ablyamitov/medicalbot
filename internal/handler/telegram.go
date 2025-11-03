package handler

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/Ablyamitov/mamedicalbot/internal/entities"
	"github.com/Ablyamitov/mamedicalbot/internal/operation"
	"github.com/Ablyamitov/mamedicalbot/internal/usecase"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type TelegramBotHandler struct {
	bot     *tgbotapi.BotAPI
	useCase *usecase.MedicalBotUseCase
}

func NewTelegramBotHandler(bot *tgbotapi.BotAPI, useCase *usecase.MedicalBotUseCase) *TelegramBotHandler {
	return &TelegramBotHandler{
		bot:     bot,
		useCase: useCase,
	}
}

func (h *TelegramBotHandler) HandleUpdate(update tgbotapi.Update) {
	if update.Message != nil {
		msg := update.Message
		from := msg.From
		chatID, userName, firstName, lastName := msg.Chat.ID, from.UserName, from.FirstName, from.LastName
		user, _ := h.useCase.EnsureUser(
			chatID,
			userName,
			firstName,
			lastName,
		)
		if user.ID != nil && (user.ChatID == nil || user.FirstName == nil || user.LastName == nil || user.Username == nil) {
			user.Username = &userName
			user.ChatID = operation.Pointer(strconv.FormatInt(chatID, 10))
			user.LastName = &lastName
			user.FirstName = &firstName
			err := h.useCase.UpdateUser(user)
			if err != nil {
				log.Printf("h.useCase.UpdateUser: %w", err)
			}
		}
		h.handleMessage(update.Message)
	} else if update.CallbackQuery != nil {
		msg := update.CallbackQuery.Message
		from := update.CallbackQuery.From
		chatID, userName, firstName, lastName := msg.Chat.ID, from.UserName, from.FirstName, from.LastName
		user, _ := h.useCase.EnsureUser(
			chatID,
			userName,
			firstName,
			lastName,
		)
		if user.ID != nil && (user.ChatID == nil || user.FirstName == nil || user.LastName == nil || user.Username == nil) {
			user.Username = &userName
			user.ChatID = operation.Pointer(strconv.FormatInt(chatID, 10))
			user.LastName = &lastName
			user.FirstName = &firstName
			err := h.useCase.UpdateUser(user)
			if err != nil {
				log.Printf("h.useCase.UpdateUser: %w", err)
			}
		}
		h.handleCallbackQuery(update.CallbackQuery)
	}
}

func (h *TelegramBotHandler) handleMessage(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	userID := strconv.FormatInt(message.From.ID, 10)
	switch message.Command() {
	case "start":
		h.handleStartCommand(chatID, userID, message.From.FirstName)
	case "help":
		h.handleHelpCommand(chatID)
	case "tests":
		h.showTestSelection(chatID, userID)
	case "statistic":
		if !h.isAdmin(chatID) {
			h.sendMessage(chatID, "🚫 У вас нет доступа к этой команде.")
			return
		}

		filePath, err := h.useCase.GetStatisticsExcel()
		if err != nil {
			h.sendMessage(chatID, "Ошибка при генерации статистики: "+err.Error())
			return
		}

		doc := tgbotapi.NewDocument(chatID, tgbotapi.FilePath(filePath))
		doc.Caption = "📊 Статистика по тестам"
		if _, err := h.bot.Send(doc); err != nil {
			log.Println("Failed to send Excel:", err)
		}
		return
	default:
		h.sendMessage(chatID, "Используйте /start для начала или /tests для выбора теста.")
	}
}

func (h *TelegramBotHandler) handleStartCommand(chatID int64, userID, firstName string) {
	welcomeText := fmt.Sprintf(
		"👋 Привет, %s!\n\n"+
			"🏥 Я медицинский бот для проведения диагностических тестов.\n\n"+
			"📋 Доступные тесты:\n"+
			"• AMS - Оценка симптомов андрогенного дефицита у мужчин (снижение тестостерона)\n"+
			"• МИЭФ-5 - Оценка эректильной функции\n"+
			"• IPSS - Оценка симптомов предстательной железы\n\n"+
			"⚠️ Внимание: Тесты предназначены для предварительной оценки и не заменяют консультацию врача.",
		firstName,
	)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📋 Выбрать тест", "show_tests"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, welcomeText)
	msg.ReplyMarkup = keyboard
	h.bot.Send(msg)
}

func (h *TelegramBotHandler) handleHelpCommand(chatID int64) {
	helpText := "📖 Справка по боту:\n\n" +
		"🔹 /start - Главное меню\n" +
		"🔹 /tests - Выбор теста\n" +
		"🔹 /help - Показать эту справку\n\n" +
		"❓ Если у вас есть вопросы, обратитесь к разработчику."

	h.sendMessage(chatID, helpText)
}

func (h *TelegramBotHandler) handleCallbackQuery(query *tgbotapi.CallbackQuery) {
	chatID := query.Message.Chat.ID
	userID := strconv.FormatInt(query.From.ID, 10)

	// Подтверждаем получение callback
	callback := tgbotapi.NewCallback(query.ID, "")
	h.bot.Request(callback)

	switch {
	case query.Data == "show_tests":
		h.showTestSelection(chatID, userID)
	case strings.HasPrefix(query.Data, "test_"):
		testType := entities.TestType(strings.TrimPrefix(query.Data, "test_"))
		h.showTestDisclaimer(chatID, userID, testType)
	case strings.HasPrefix(query.Data, "start_test_"):
		testType := entities.TestType(strings.TrimPrefix(query.Data, "start_test_"))
		h.startTest(chatID, userID, testType)
	case strings.HasPrefix(query.Data, "continue_test_"):
		testType := entities.TestType(strings.TrimPrefix(query.Data, "continue_test_"))
		h.continueTest(chatID, userID, testType)
	case strings.HasPrefix(query.Data, "answer_"):
		h.handleAnswer(chatID, userID, query.Data)
	case query.Data == "show_result":
		h.showTestResult(chatID, userID)
	}
}

func (h *TelegramBotHandler) showTestSelection(chatID int64, userID string) {
	tests, err := h.useCase.GetAvailableTests()
	if err != nil {
		h.sendMessage(chatID, "Ошибка при загрузке тестов.")
		return
	}

	text := "📋 Выберите тест для прохождения:\n\n"

	var buttons [][]tgbotapi.InlineKeyboardButton

	for _, test := range tests {
		text += fmt.Sprintf("🔹 **%s**\n%s\n\n", test.Name, test.Description)

		button := tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				getTestEmoji(test.Type)+" "+test.Name,
				"test_"+string(test.Type),
			),
		)
		buttons = append(buttons, button)
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(buttons...)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard
	h.bot.Send(msg)
}

func (h *TelegramBotHandler) showTestDisclaimer(chatID int64, userID string, testType entities.TestType) {
	test, err := h.useCase.TestRepo.GetTest(testType)
	if err != nil {
		h.sendMessage(chatID, "Ошибка при загрузке теста.")
		return
	}

	text := fmt.Sprintf("📋 **%s**\n\n%s", test.Name, test.Disclaimer)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🚀 Приступить к тесту", "start_test_"+string(testType)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад к выбору тестов", "show_tests"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard
	h.bot.Send(msg)
}

func (h *TelegramBotHandler) startTest(chatID int64, userID string, testType entities.TestType) {
	session, test, err := h.useCase.StartTest(userID, testType)
	if err != nil {
		h.sendMessage(chatID, "Ошибка при создании теста: "+err.Error())
		return
	}

	startText := fmt.Sprintf(
		"🚀 Начинаем тест: **%s**\n\n"+
			"📊 Всего вопросов: %d\n"+
			"⏱️ Примерное время прохождения: %d минут",
		test.Name,
		len(test.Questions),
		len(test.Questions),
	)

	question, err := h.useCase.GetCurrentQuestion(userID)
	if err != nil {
		h.sendMessage(chatID, "Ошибка при получении вопроса.")
		return
	}

	h.sendMessage(chatID, startText)
	h.sendQuestion(chatID, question, session.CurrentStep, len(test.Questions))
}

func (h *TelegramBotHandler) sendQuestion(chatID int64, question *entities.Question, questionNumber, totalQuestions int) {
	questionText := fmt.Sprintf(
		"❓ **Вопрос %d из %d:**\n\n%s",
		questionNumber,
		totalQuestions,
		question.Text,
	)

	var buttons [][]tgbotapi.InlineKeyboardButton

	// Создаем кнопки для каждого варианта ответа
	for _, option := range question.Options {
		button := tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				option.Text,
				fmt.Sprintf("answer_%d_%d", question.ID, option.Value),
			),
		)
		buttons = append(buttons, button)
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(buttons...)

	msg := tgbotapi.NewMessage(chatID, questionText)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard
	h.bot.Send(msg)
}

func (h *TelegramBotHandler) handleAnswer(chatID int64, userID, callbackData string) {
	// Парсим callback data: answer_questionID_value
	parts := strings.Split(callbackData, "_")
	if len(parts) != 3 {
		return
	}

	value, err := strconv.Atoi(parts[2])
	if err != nil {
		h.sendMessage(chatID, "Ошибка при обработке ответа.")
		return
	}
	user, err := h.useCase.GetUserByChatID(userID)
	if err != nil {
		h.sendMessage(chatID, "Ошибка при обработке ответа.")
		return
	}

	nextQuestion, err := h.useCase.SubmitAnswer(operation.Dereference(user.ID), value)
	if err != nil {
		h.sendMessage(chatID, "Ошибка при обработке ответа: "+err.Error())
		return
	}

	if nextQuestion == nil {
		// Тест завершен
		if ok, err := h.useCase.CompleteTest(operation.Dereference(user.ID)); !ok || err != nil {
			h.sendMessage(chatID, "Ошибка завершения теста.")
		}
		h.sendCompletionMessage(chatID, userID)
	} else {
		// Показываем следующий вопрос
		session, _ := h.useCase.SessionRepo.GetSessionByUser(operation.Dereference(user.ID))
		test, _ := h.useCase.TestRepo.GetTest(session.TestType)
		h.sendQuestion(chatID, nextQuestion, session.CurrentStep, len(test.Questions))
	}
}

func (h *TelegramBotHandler) sendCompletionMessage(chatID int64, userID string) {
	completionText := "🎉 Поздравляю! Вы ответили на все вопросы.\n\n" +
		"📊 Сейчас я проанализирую ваши ответы и предоставлю результат..."

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📊 Показать результат", "show_result"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, completionText)
	msg.ReplyMarkup = keyboard
	h.bot.Send(msg)
}

func (h *TelegramBotHandler) showTestResult(chatID int64, userID string) {
	user, err := h.useCase.GetUserByChatID(userID)
	if err != nil {
		h.sendMessage(chatID, "Ошибка при получении результата.")
		return
	}
	result, err := h.useCase.GetTestResult(operation.Dereference(user.ID))
	if err != nil {
		h.sendMessage(chatID, "Ошибка при получении результата: "+err.Error())
		return
	}

	resultText := fmt.Sprintf(
		"📊 **Результат теста %s**\n\n"+
			"🔢 Общий балл: **%d**\n\n"+
			"📋 **Интерпретация:**\n%s\n\n"+
			"💡 **Рекомендации:**\n%s",
		result.TestType,
		result.Score,
		result.Interpretation,
		strings.Join(result.Recommendations, "\n"),
	)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📋 Пройти другой тест", "show_tests"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔄 Пройти этот тест еще раз", "test_"+string(result.TestType)),
		),
	)

	msg := tgbotapi.NewMessage(chatID, resultText)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard
	h.bot.Send(msg)

}
func (h *TelegramBotHandler) continueTest(chatID int64, userID string, testType entities.TestType) {
	// Получаем активную сессию пользователя для этого теста
	user, err := h.useCase.UserRepo.GetByChatID(userID)
	if err != nil {
		h.sendMessage(chatID, "❌ Активная сессия не найдена. Начните тест заново.")
		return
	}
	if user.ID == nil {
		h.sendMessage(chatID, "❌ Активная сессия не найдена. Начните тест заново.")
		return
	}
	session, err := h.useCase.SessionRepo.GetActiveSessionByUserAndType(operation.Dereference(user.ID), testType, "in_progress")
	if err != nil || session == nil {
		h.sendMessage(chatID, "❌ Активная сессия не найдена. Начните тест заново.")
		return
	}

	// Получаем информацию о тесте
	test, err := h.useCase.TestRepo.GetTest(testType)
	if err != nil {
		h.sendMessage(chatID, "Ошибка при загрузке теста.")
		return
	}

	// Показываем сообщение о продолжении
	continueText := fmt.Sprintf(
		"🔄 Продолжаем тест: **%s**\n\n"+
			"📊 Пройдено вопросов: %d из %d\n"+
			"⏱️ Вы вернулись к тесту",
		test.Name,
		session.CurrentStep-1,
		len(test.Questions),
	)

	h.sendMessage(chatID, continueText)

	// Показываем текущий вопрос
	question, err := h.useCase.GetCurrentQuestion(userID)
	if err != nil {
		h.sendMessage(chatID, "Ошибка при получении вопроса.")
		return
	}

	h.sendQuestion(chatID, question, session.CurrentStep, len(test.Questions))
}

func (h *TelegramBotHandler) sendMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	h.bot.Send(msg)
}

func getTestEmoji(testType entities.TestType) string {
	switch testType {
	case entities.TestTypeAMS:
		return "👨‍⚕️"
	case entities.TestTypeMIEF:
		return "❤️"
	case entities.TestTypeIPSS:
		return "🏥"
	default:
		return "📋"
	}
}

var adminChatIDs = []int64{
	428268899,
	1584657325,
}

func (h *TelegramBotHandler) isAdmin(chatID int64) bool {
	for _, id := range adminChatIDs {
		if id == chatID {
			return true
		}
	}
	return false
}
