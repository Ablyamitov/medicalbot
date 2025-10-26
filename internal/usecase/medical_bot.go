package usecase

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/Ablyamitov/mamedicalbot/internal/entities"
	"github.com/Ablyamitov/mamedicalbot/internal/repositories"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type MedicalBotUseCase struct {
	TestRepo     repositories.TestRepository
	SessionRepo  repositories.SessionRepository
	QuestionRepo repositories.QuestionRepository
}

func NewMedicalBotUseCase(testRepo repositories.TestRepository, sessionRepo repositories.SessionRepository, questionRepo repositories.QuestionRepository) *MedicalBotUseCase {
	return &MedicalBotUseCase{
		TestRepo:     testRepo,
		SessionRepo:  sessionRepo,
		QuestionRepo: questionRepo,
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

	// 2️⃣ Проверяем, есть ли уже активная сессия по этому типу
	existingSession, err := uc.SessionRepo.GetActiveSessionByPatientAndType(patientID, testType)
	if err != nil {
		return nil, nil, err
	}

	// 3️⃣ Если активная сессия есть — продолжаем её
	if existingSession != nil {
		return existingSession, test, nil
	}

	// 4️⃣ Иначе создаем новую
	newSession, err := uc.SessionRepo.CreateSession(patientID, testType)
	if err != nil {
		return nil, nil, err
	}

	return newSession, test, nil
}
func (uc *MedicalBotUseCase) GetCurrentQuestion(patientID string) (*entities.Question, error) {
	session, err := uc.SessionRepo.GetSessionByPatient(patientID)
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

func (uc *MedicalBotUseCase) SubmitAnswer(patientID string, value int) (*entities.Question, error) {
	session, err := uc.SessionRepo.GetSessionByPatient(patientID)
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
	session, err := uc.SessionRepo.GetSessionByPatient(patientID)
	if err != nil {
		return nil, err
	}

	if session.Status != "completed" {
		return nil, fmt.Errorf("test not completed yet")
	}

	score := uc.calculateScore(session.Answers)
	interpretation := uc.interpretResult(session.TestType, score)
	recommendations := uc.generateRecommendations(session.TestType, score)

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
	session, err := uc.SessionRepo.GetSessionByPatient(userID)
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
	patients, err := uc.SessionRepo.GetPatientsWithUnfinishedTests()
	if err != nil {
		return fmt.Errorf("get patients: %w", err)
	}

	for _, patientID := range patients {
		// 2️⃣ Проверяем, является ли последний тест пользователя завершенным
		lastTestCompleted, err := uc.SessionRepo.IsLastTestCompleted(patientID)
		if err != nil {
			continue
		}
		if lastTestCompleted {
			// последний тест завершен — не напоминаем
			continue
		}

		// 3️⃣ Берём его незавершённые сессии, отсортированные по времени (последняя — первая)
		sessions, err := uc.SessionRepo.GetUnfinishedSessions(patientID)
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
					"start_test_"+string(lastSession.TestType),
				),
			),
		)

		chatID, err := strconv.ParseInt(lastSession.PatientID, 10, 64)
		if err != nil {
			continue
		}

		msgObj := tgbotapi.NewMessage(chatID, msg)
		msgObj.ParseMode = "Markdown"
		msgObj.ReplyMarkup = keyboard

		if _, err := bot.Send(msgObj); err != nil {
			log.Printf("failed to send reminder to %s: %v", lastSession.PatientID, err)
			continue
		}

		// 5️⃣ Обновляем UpdatedAt, чтобы не слать часто
		lastSession.UpdatedAt = time.Now()
		_ = uc.SessionRepo.UpdateSession(lastSession)
	}

	return nil
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
