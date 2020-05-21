package main

import (
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

// ErrTelega ... Структура для передачи параметров в функцию ошибок
type ErrTelega struct {
	fromID *int64
	code   uint16 // свой код ошибки (см. функцию errMessage)
	err    error
	table  string
	field  string
	val    interface{}
	descr  string
}

// errMessage ... обработка ошибочных ситуаций
// 1 - 19: Redis
// 20 - 40: SQL
// 41 - 60: общие ошибки (преобразование и т.д.)
// TODO: организовать уведомление об ошибках
func errMessage(et *ErrTelega) {

	var (
		msg        tgbotapi.MessageConfig
		msgText    string
		msgTextSrv = `Проблемы на стороне сервера, уже разбираемся...`
		rm         *tgbotapi.InlineKeyboardMarkup
	)

	switch et.code {
	// 1. Не удалось найти ключ пользователя в Redis
	case 1:
		msgText = `Истекло время создания команды, вам придётся начать заново.
		Начать заново?`
		msg = tgbotapi.NewMessage(*et.fromID, msgText)
		rm = getButtonYesNo(sa+"1", nothing+"0")
		msg.ReplyMarkup = rm
	// 2. не удалось добавить ключ в кэш
	case 2:
		log.Println(*et.fromID, "| 2. Не удалось добавить ключ (HGET):", et.field)
		msg = tgbotapi.NewMessage(*et.fromID, msgTextSrv)
	// 11. не удалось распарсить ключ в кэше
	case 11:
		log.Println(*et.fromID, "| 11. Не удалось распарсить ключ")
		msg = tgbotapi.NewMessage(*et.fromID, msgTextSrv)

	// 21. Проблема с вставкой данных в таблицу
	case 21:
		log.Println(*et.fromID, "| 21. Не удалось получить %s;", et.field, et.err)
		msg = tgbotapi.NewMessage(*et.fromID, msgTextSrv)
	// 22. Проблема с транзакцией
	case 22:
		log.Println(*et.fromID, "| 22. Не удалось обработать транзакцию: ", et.err)
		msg = tgbotapi.NewMessage(*et.fromID, msgTextSrv)
		log.Println("Transaction.", et.err)
	// 23. Вернулась пустая выборка (SELECT/ Update Returning id)
	case 23:
		log.Println(*et.fromID, "| 23. Не удалось найти данные в таблице: %s; ", et.field, et.err)
		msg = tgbotapi.NewMessage(*et.fromID, msgTextSrv)
		log.Println("Transaction.", et.err)
	// 24. Не удалось обновить данные (UPDATE)
	case 24:
		log.Printf("%d | 24. Не удалось обновить данные, таблица: %s; %s", *et.fromID, et.table, et.err)
		msg = tgbotapi.NewMessage(*et.fromID, msgTextSrv)
		log.Println("Transaction.", et.err)

	// 41. Проблема с преобразованием типов данных
	case 41:
		log.Printf("%d| 41. Не удалось преобразовать поле: %s; %s\n", *et.fromID, et.field, et.err)
		msg = tgbotapi.NewMessage(*et.fromID, msgTextSrv)
	default:
		log.Println(*et.fromID, "| 404. Упс, неизвестная ошибка...")
		msgText = `Упс, неизвестная ошибка...`
		msg = tgbotapi.NewMessage(*et.fromID, msgText)
	}

	bot.Send(msg)

}
