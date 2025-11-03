package usecase

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Ablyamitov/mamedicalbot/internal/entities"
	"github.com/Ablyamitov/mamedicalbot/internal/operation"
	"github.com/Ablyamitov/mamedicalbot/internal/repositories"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/xuri/excelize/v2"
)

type MedicalBotUseCase struct {
	TestRepo     repositories.TestRepository
	SessionRepo  repositories.SessionRepository
	QuestionRepo repositories.QuestionRepository
	UserRepo     repositories.UserRepository
}

func NewMedicalBotUseCase(testRepo repositories.TestRepository,
	sessionRepo repositories.SessionRepository,
	questionRepo repositories.QuestionRepository,
	userRepo repositories.UserRepository) *MedicalBotUseCase {
	return &MedicalBotUseCase{
		TestRepo:     testRepo,
		SessionRepo:  sessionRepo,
		QuestionRepo: questionRepo,
		UserRepo:     userRepo,
	}
}
func (uc *MedicalBotUseCase) GetAvailableTests() ([]entities.Test, error) {
	return uc.TestRepo.GetAllTests()
}

func (uc *MedicalBotUseCase) StartTest(patientID string, testType entities.TestType) (*entities.Session, *entities.Test, error) {
	// 1️⃣ Получаем сам тест (вопросы и т.д.)
	test, err := uc.TestRepo.GetTest(testType)
	if err != nil {
		return nil, nil, err
	}

	user, err := uc.UserRepo.GetByChatID(patientID)
	if err != nil {
		return nil, nil, err
	}

	if user == nil {
		return nil, nil, fmt.Errorf("user must be not nil")
	}

	// 2️⃣ Проверяем, есть ли уже активная сессия по этому типу
	existingSession, err := uc.SessionRepo.GetActiveSessionByUserAndType(operation.Dereference(user.ID), testType, "started")
	if err != nil {
		return nil, nil, err
	}

	// 3️⃣ Если активная сессия есть — продолжаем её
	if existingSession != nil {
		return existingSession, test, nil
	}

	// 4️⃣ Иначе создаем новую
	newSession, err := uc.SessionRepo.CreateSession(operation.Dereference(user.ID), testType)
	if err != nil {
		return nil, nil, err
	}

	return newSession, test, nil
}
func (uc *MedicalBotUseCase) GetCurrentQuestion(patientID string) (*entities.Question, error) {
	user, err := uc.UserRepo.GetByChatID(patientID)
	if err != nil {
		return nil, err
	}

	session, err := uc.SessionRepo.GetSessionByUser(operation.Dereference(user.ID))
	if err != nil {
		return nil, err
	}

	test, err := uc.TestRepo.GetTest(session.TestType)
	if err != nil {
		return nil, err
	}

	if session.CurrentStep > len(test.Questions) {
		return nil, fmt.Errorf("all questions completed")
	}

	question, err := uc.getQuestionByStep(session.CurrentStep, test.ID)
	if err != nil {
		return nil, fmt.Errorf("uc.getQuestionByStep: %w", err)
	}
	if question == nil {
		return nil, fmt.Errorf("all questions completed")
	}

	return question, nil
}

func (uc *MedicalBotUseCase) SubmitAnswer(userID string, value int) (*entities.Question, error) {

	session, err := uc.SessionRepo.GetSessionByUser(userID)
	if err != nil {
		return nil, err
	}

	test, err := uc.TestRepo.GetTest(session.TestType)
	if err != nil {
		return nil, err
	}

	if session.CurrentStep > len(test.Questions) {
		return nil, fmt.Errorf("all questions already completed")
	}

	// Добавляем ответ
	currentQuestion, err := uc.getQuestionByStep(session.CurrentStep, test.ID)
	if err != nil {
		return nil, fmt.Errorf("uc.getQuestionByStep: %w", err)
	}
	answer := entities.Answer{
		QuestionID: currentQuestion.ID,
		Value:      value,
	}

	err = uc.SessionRepo.AddAnswer(session.ID, answer)
	if err != nil {
		return nil, err
	}

	// Переходим к следующему вопросу
	session.CurrentStep++
	session.Status = "in_progress"

	if session.CurrentStep > len(test.Questions) {
		session.Status = "completed"
		uc.SessionRepo.UpdateSession(session)
		return nil, nil // Все вопросы завершены
	}

	uc.SessionRepo.UpdateSession(session)

	nextQuestion, err := uc.getQuestionByStep(session.CurrentStep, test.ID)
	// Возвращаем следующий вопрос
	return nextQuestion, nil
}

func (uc *MedicalBotUseCase) GetTestResult(patientID string) (*entities.TestResult, error) {
	session, err := uc.SessionRepo.GetSessionByUser(patientID)
	if err != nil {
		return nil, err
	}

	if session.Status != "completed" {
		return nil, fmt.Errorf("test not completed yet")
	}

	score := uc.calculateScore(session.Answers)
	interpretation := uc.interpretResult(session.TestType, score)
	recommendations := uc.generateRecommendations(session.TestType, score)

	saveRecommendations := make([]string, len(recommendations))
	copy(saveRecommendations, recommendations)
	saveRecommendations = saveRecommendations[:len(saveRecommendations)-1]
	if err := uc.SessionRepo.SaveTestResult(session, score, interpretation, saveRecommendations); err != nil {
		return nil, err
	}

	result := &entities.TestResult{
		SessionID:       session.ID,
		TestType:        session.TestType,
		Score:           score,
		Interpretation:  interpretation,
		Recommendations: recommendations,
	}

	return result, nil
}

func (uc *MedicalBotUseCase) CompleteTest(userID string) (bool, error) {
	session, err := uc.SessionRepo.GetSessionByUser(userID)
	if err != nil {
		return false, err
	}
	session.Status = "completed"
	if err := uc.SessionRepo.UpdateSession(session); err != nil {
		return false, err
	}
	return true, nil
}

func (uc *MedicalBotUseCase) SendRemindersForIncompleteTests(bot *tgbotapi.BotAPI) error {
	// 1️⃣ Получаем всех пациентов, у кого есть незавершённые тесты
	users, err := uc.SessionRepo.GetUsersWithUnfinishedTests()
	if err != nil {
		return fmt.Errorf("get patients: %w", err)
	}

	for _, userID := range users {
		// 2️⃣ Проверяем, является ли последний тест пользователя завершенным
		lastTestCompleted, err := uc.SessionRepo.IsLastTestCompleted(userID)
		if err != nil {
			continue
		}
		if lastTestCompleted {
			// последний тест завершен — не напоминаем
			continue
		}

		// 3️⃣ Берём его незавершённые сессии, отсортированные по времени (последняя — первая)
		sessions, err := uc.SessionRepo.GetUnfinishedSessions(userID)
		if err != nil || len(sessions) == 0 {
			continue
		}

		// 4️⃣ Берём только последнюю активную сессию
		lastSession := sessions[0]

		timeSince := time.Since(lastSession.UpdatedAt)
		fmt.Printf("Raw time since: %v\n", timeSince)

		if timeSince < 12*time.Hour {
			continue
		}

		test, err := uc.TestRepo.GetTest(lastSession.TestType)
		if err != nil {
			continue
		}

		msg := fmt.Sprintf(
			"⏰ Привет! Вы начали тест *%s*, но не закончили его.\n\n"+
				"Продолжим прямо сейчас?",
			test.Name,
		)

		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(
					"✅ Продолжить тест",
					"continue_test_"+string(lastSession.TestType),
				),
			),
		)

		user, err := uc.UserRepo.GetByID(userID)
		if err != nil {
			continue
		}
		chatID, err := strconv.ParseInt(operation.Dereference(user.ChatID), 10, 64)
		if err != nil {
			continue
		}

		msgObj := tgbotapi.NewMessage(chatID, msg)
		msgObj.ParseMode = "Markdown"
		msgObj.ReplyMarkup = keyboard

		if _, err := bot.Send(msgObj); err != nil {
			log.Printf("failed to send reminder to %s: %v", user.ChatID, err)
			continue
		}

		// 5️⃣ Обновляем UpdatedAt, чтобы не слать часто
		lastSession.UpdatedAt = time.Now()
		_ = uc.SessionRepo.UpdateSession(lastSession)
	}

	return nil
}

func (uc *MedicalBotUseCase) EnsureUser(chatID int64, username, firstName, lastName string) (*entities.User, error) {
	strChatID := fmt.Sprint(chatID)

	user, err := uc.UserRepo.GetByChatID(strChatID)
	if err != nil {
		return nil, err
	}

	// Если пользователь уже существует
	if user != nil {
		return user, nil
	}

	// Иначе создаём нового
	newUser := &entities.User{
		ChatID:    operation.Pointer(strChatID),
		Username:  operation.Pointer(username),
		FirstName: operation.Pointer(firstName),
		LastName:  operation.Pointer(lastName),
	}

	return uc.UserRepo.Create(newUser)
}

func (uc *MedicalBotUseCase) GetUserByChatID(chatID string) (*entities.User, error) {
	user, err := uc.UserRepo.GetByChatID(chatID)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (uc *MedicalBotUseCase) UpdateUser(user *entities.User) error {
	err := uc.UserRepo.Update(user)
	if err != nil {
		return err
	}
	return nil
}

func (uc *MedicalBotUseCase) GetStatisticsExcel() (string, error) {
	stats, err := uc.SessionRepo.GetStatistics()
	if err != nil {
		return "", err
	}

	f := excelize.NewFile()
	sheet := "Статистика"
	f.SetSheetName("Sheet1", sheet)

	// Группируем сессии по пользователям
	userSessions := make(map[string][]entities.SessionStatistic)
	for _, stat := range stats {
		userKey := fmt.Sprintf("%d_%s_%s", stat.ChatID, stat.FirstName, stat.LastName)
		userSessions[userKey] = append(userSessions[userKey], stat)
	}

	// 📊 Устанавливаем ширину колонок
	uc.setColumnWidths(f, sheet)

	// ✨ Заголовок отчета
	f.SetCellValue(sheet, "A1", "Отчет по результатам тестирования")
	f.MergeCell(sheet, "A1", "M1")
	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 16, Color: "1F4E78"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"D9EAD3"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	f.SetCellStyle(sheet, "A1", "M1", titleStyle)

	// 📈 Сводная информация
	summaryRow := 3
	f.SetCellValue(sheet, fmt.Sprintf("A%d", summaryRow),
		fmt.Sprintf("Всего пользователей: %d | Всего сессий: %d", len(userSessions), len(stats)))
	f.MergeCell(sheet, fmt.Sprintf("A%d", summaryRow), fmt.Sprintf("M%d", summaryRow))

	summaryStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 11, Color: "1F4E78"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"E6F0FF"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	f.SetCellStyle(sheet, fmt.Sprintf("A%d", summaryRow), fmt.Sprintf("M%d", summaryRow), summaryStyle)

	// 🗂️ Заголовки таблицы
	headerRow := 5
	headers := []string{
		"ID Сессии", "Chat ID", "Имя", "Фамилия", "Username",
		"Тип теста", "Статус", "Создан", "Обновлен", "Отвечено",
		"Баллы", "Интерпретация", "Рекомендации",
	}

	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, headerRow)
		f.SetCellValue(sheet, cell, h)
	}

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 11, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"2F75B5"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
	})
	f.SetCellStyle(sheet, "A5", "M5", headerStyle)

	// 📝 Создаем стили для статусов
	completedStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 10, Color: "107C10"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"E2F0D9"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
		},
	})

	inProgressStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 10, Color: "E68C31"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"FFF2CC"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
		},
	})

	// Стили для чередования пользователей (базовые)
	userStyle1, _ := f.NewStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"F8F8F8"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
		},
	})

	userStyle2, _ := f.NewStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"FFFFFF"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
		},
	})

	// 📝 Заполняем данными с группировкой по пользователям
	dataStartRow := 6
	currentRow := dataStartRow

	// Сортируем пользователей для консистентного отображения
	var userKeys []string
	for key := range userSessions {
		userKeys = append(userKeys, key)
	}
	sort.Strings(userKeys)

	// Заполняем данные по каждому пользователю
	for userIndex, userKey := range userKeys {
		sessions := userSessions[userKey]

		// Определяем базовый стиль для текущего пользователя
		baseUserStyle := userStyle1
		if userIndex%2 == 0 {
			baseUserStyle = userStyle1
		} else {
			baseUserStyle = userStyle2
		}

		// Добавляем сессии пользователя
		for _, s := range sessions {
			recommendations := "Нет рекомендаций"
			if len(s.Recommendations) > 0 {
				var formatted []string
				for j, rec := range s.Recommendations {
					formatted = append(formatted, fmt.Sprintf("%d. %s", j+1, rec))
				}
				recommendations = strings.Join(formatted, "\n")
			}

			values := []interface{}{
				s.SessionID,
				s.ChatID,
				s.FirstName,
				s.LastName,
				s.Username,
				uc.formatTestType(s.TestType),
				uc.formatStatus(s.Status), // Статус с эмодзи
				s.CreatedAt.Format("02.01.2006 15:04"),
				s.UpdatedAt.Format("02.01.2006 15:04"),
				s.AnsweredQuestions,
				s.Score,
				s.Interpretation,
				recommendations,
			}

			for j, v := range values {
				cell, _ := excelize.CoordinatesToCellName(j+1, currentRow)
				f.SetCellValue(sheet, cell, v)
			}

			// Применяем стиль в зависимости от статуса
			if s.Status == "completed" {
				f.SetCellStyle(sheet, fmt.Sprintf("A%d", currentRow), fmt.Sprintf("M%d", currentRow), completedStyle)
			} else if s.Status == "in_progress" {
				f.SetCellStyle(sheet, fmt.Sprintf("A%d", currentRow), fmt.Sprintf("M%d", currentRow), inProgressStyle)
			} else {
				// Для других статусов используем базовый стиль пользователя
				f.SetCellStyle(sheet, fmt.Sprintf("A%d", currentRow), fmt.Sprintf("M%d", currentRow), baseUserStyle)
			}

			currentRow++
		}

		// Добавляем пустую строку между пользователями (кроме последнего)
		if userIndex < len(userKeys)-1 {
			currentRow++
		}
	}

	// 🔧 Настройки таблицы
	lastRow := currentRow - 1
	if lastRow >= dataStartRow {
		f.AutoFilter(sheet, fmt.Sprintf("A%d:M%d", headerRow, lastRow), []excelize.AutoFilterOptions{})
		f.SetPanes(sheet, &excelize.Panes{
			Freeze:      true,
			YSplit:      headerRow,
			TopLeftCell: "A6",
			ActivePane:  "bottomLeft",
		})
	}

	// 💾 Сохраняем файл
	fileName := fmt.Sprintf("statistics_%s.xlsx", time.Now().Format("2006-01-02_15-04-05"))
	filePath := filepath.Join(os.TempDir(), fileName)

	if err := f.SaveAs(filePath); err != nil {
		return "", fmt.Errorf("failed to save excel file: %w", err)
	}

	return filePath, nil
}

// Обновленная функция форматирования статуса с эмодзи
func (uc *MedicalBotUseCase) formatStatus(status string) string {
	statuses := map[string]string{
		"completed":   "✅ Завершен",
		"in_progress": "🔄 В процессе",
		"created":     "📝 Создан",
	}
	if formatted, ok := statuses[status]; ok {
		return formatted
	}
	return status
}

func (uc *MedicalBotUseCase) setColumnWidths(f *excelize.File, sheet string) {
	widths := []float64{
		36, 15, 15, 15, 15, // A-E
		20, 12, 16, 16, 12, // F-J
		10, 25, 40, // K-M
	}

	for i, width := range widths {
		col, _ := excelize.ColumnNumberToName(i + 1)
		f.SetColWidth(sheet, col, col, width)
	}
}

func (uc *MedicalBotUseCase) countCompletedSessions(stats []entities.SessionStatistic) int {
	count := 0
	for _, s := range stats {
		if s.Status == "completed" {
			count++
		}
	}
	return count
}

func (uc *MedicalBotUseCase) formatTestType(testType string) string {
	types := map[string]string{
		"anxiety":    "Тревожность",
		"depression": "Депрессия",
		"stress":     "Стресс",
	}
	if formatted, ok := types[testType]; ok {
		return formatted
	}
	return testType
}

func (uc *MedicalBotUseCase) calculateScore(answers []entities.Answer) int {
	score := 0
	for _, answer := range answers {
		score += answer.Value
	}
	return score
}

func (uc *MedicalBotUseCase) interpretResult(testType entities.TestType, score int) string {
	switch testType {
	case entities.TestTypeAMS:
		if score <= 26 {
			return "Отсутствие или минимальные симптомы андрогенного дефицита"
		} else if score <= 36 {
			return "Слабые симптомы андрогенного дефицита"
		} else if score <= 49 {
			return "Умеренные симптомы андрогенного дефицита"
		}
		return "Выраженные симптомы андрогенного дефицита"

	case entities.TestTypeMIEF:
		if score >= 22 {
			return "Норма - эректильная дисфункция отсутствует"
		} else if score >= 17 {
			return "Легкая эректильная дисфункция"
		} else if score >= 12 {
			return "Легкая-умеренная эректильная дисфункция"
		} else if score >= 8 {
			return "Умеренная эректильная дисфункция"
		}
		return "Тяжелая эректильная дисфункция"

	case entities.TestTypeIPSS:
		if score <= 7 {
			return "Слабая выраженность симптомов"
		} else if score <= 19 {
			return "Умеренная выраженность симптомов"
		}
		return "Выраженные симптомы"

	default:
		return "Результат определен"
	}
}

func (uc *MedicalBotUseCase) generateRecommendations(testType entities.TestType, score int) []string {
	recommendations := []string{}

	switch testType {
	case entities.TestTypeAMS:
		if score > 36 {
			recommendations = append(recommendations, "🏥 Рекомендуется консультация врача-андролога")
			recommendations = append(recommendations, "🧪 Возможно потребуется определение уровня тестостерона")
		}
		recommendations = append(recommendations, "🏃‍♂️ Регулярная физическая активность")
		recommendations = append(recommendations, "🥗 Сбалансированное питание")
		recommendations = append(recommendations, "😴 Нормализация режима сна")

	case entities.TestTypeMIEF:
		if score < 17 {
			recommendations = append(recommendations, "🏥 Рекомендуется консультация врача-уролога")
			recommendations = append(recommendations, "🧪 Обследование для выявления причин эректильной дисфункции")
		}
		recommendations = append(recommendations, "🚭 Отказ от курения")
		recommendations = append(recommendations, "🍷 Ограничение алкоголя")
		recommendations = append(recommendations, "💪 Физические упражнения для укрепления тазового дна")

	case entities.TestTypeIPSS:
		if score > 7 {
			recommendations = append(recommendations, "🏥 Рекомендуется консультация врача-уролога")
			recommendations = append(recommendations, "🧪 Дополнительное обследование предстательной железы")
		}
		recommendations = append(recommendations, "💧 Контроль потребления жидкости")
		recommendations = append(recommendations, "🚫 Ограничение кофеина и алкоголя")

	default:
		recommendations = append(recommendations, "🏥 Консультация врача для интерпретации результатов")
	}

	recommendations = append(recommendations, "⚠️ Данный тест носит справочный характер и не заменяет врачебную консультацию")

	return recommendations
}

func (uc *MedicalBotUseCase) getQuestionByStep(step int, testID int64) (*entities.Question, error) {

	q, err := uc.QuestionRepo.GetQuestionByStepAndTestID(step, testID)
	if err != nil {
		return nil, err
	}
	return q, nil
}
