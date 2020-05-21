package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

var (
	// Ctx ... null context
	Ctx = context.Background()
	bot *tgbotapi.BotAPI
	// Extime ... время жизни ключа в кэше redis
	Extime time.Duration
)

func main() {

	cfg, err := readConfig()
	if err != nil {
		panic(fmt.Sprintf("Configuration error: %v", err))
	}

	bot, err = tgbotapi.NewBotAPI(cfg.BotToken)
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = false
	log.Printf("Authorized on account %s", bot.Self.UserName)
	// TODO: переделать приём сообщений через вебхуки
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	err = dbConn(cfg)
	if err != nil {
		log.Fatalln(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer Pool.Close()

	err = redisConn(cfg)
	if err != nil {
		log.Fatalln(os.Stderr, "Unable to connect to redis: %v\n", err)
		os.Exit(1)
	}
	Extime = time.Duration(cfg.RedisTimeKey) * time.Minute
	defer Client.Close()

	initButtons()

	updates, err := bot.GetUpdatesChan(u)

	log.Printf("Init ... OK")

	for update := range updates {
		var thisMsg *tgbotapi.Message
		var answerMsg tgbotapi.MessageConfig
		var key string

		fromID := FromID(update)

		// ответы на кнопки
		if update.CallbackQuery != nil {

			key = answerCallback(update.CallbackQuery, &fromID)
			if key != "" {
				answerMsg = tgbotapi.NewMessage(fromID, *caseAction[key].TextMsg)
				answerMsg.ReplyMarkup = caseAction[key].ReplyMarkup
			}
		} else {

			// fmt.Printf("%T\n", answerMsg)
			thisMsg = update.Message
			if thisMsg == nil { // ignore any non-Message Updates
				continue
			}

			if thisMsg.IsCommand() {
				answerMsg = AnswerSystemCommand(thisMsg, &fromID)
			} else {
				// answerMsg = tgbotapi.NewMessage(fromID, update.Message.Text)
				AnswerTextResponse(thisMsg, &fromID)
				continue
			}

		}
		bot.Send(answerMsg)
	}
}

// AnswerSystemCommand ... Ответы на системные команды
func AnswerSystemCommand(msg *tgbotapi.Message, fromID *int64) tgbotapi.MessageConfig {

	var (
		ansMsg tgbotapi.MessageConfig
	)

	typeMsg := msg.Command()

	if typeMsg == "start" {
		// TODO: тут лезть в базу и смотреть есть ли уже инфа
		ansMsg = tgbotapi.NewMessage(*fromID, *caseAction["kindSports"].TextMsg)
		ansMsg.ReplyMarkup = caseAction["kindSports"].ReplyMarkup

	} else {
		ansMsg = tgbotapi.NewMessage(*fromID, "неизвестно что это - "+typeMsg)
	}
	return ansMsg

}

// AnswerTextResponse ... Ответы на ТЕКСТЫ пользователей
func AnswerTextResponse(msg *tgbotapi.Message, fromID *int64) string {

	var et *ErrTelega

	fromIDStr := strconv.FormatInt(*fromID, 10)

	// ищем ключ в redis
	// keyEx := Client.Exists(fromIDStr)
	// if keyEx.Val() != 1 {
	// 	log.Println("не удалось найти ключ-", fromIDStr)
	// 	errMessage(1, fromID, bot)
	// 	return ""
	// }

	obj, err := Client.HGetAll(fromIDStr).Result()

	if err != nil {
		et = &ErrTelega{fromID: fromID, code: 1}
		errMessage(et)
		return ""
	}

	var ok bool
	for _, el := range redisKeyAnsTxt {
		_, ok = obj[el.key]
		if ok {
			el.f(&obj, &msg.Text, fromID, &fromIDStr)
			break
		}
	}

	// Если ничего не нашли, то ключ "сгорел" (см. время жизни ключа)
	if !ok {
		et = &ErrTelega{fromID: fromID, code: 1}
		errMessage(et)
	}

	return ""

}

// answerCallback ... ответы на КНОПКИ
func answerCallback(cq *tgbotapi.CallbackQuery, fromID *int64) string {
	// 0 - тип ответа
	// 1 - значение
	typeData := strings.Split(cq.Data, "|")
	data := typeData[1]

	switch typeData[0] {
	case "kindSports":
		log.Println("сохранение вида спорта ...")
		return saveKindSports(cq, &data, fromID)
	case "selectAct":
		switch data {
		// Создать команду
		case "1":
			log.Println("создание команды ...")
			selectLocation(fromID)
		// Рынок свободных игроков
		case "2":
			log.Println("выйти на рынок свободных игроков ...")
			// selectLocation(fromID)
		// Найти спарринг-команду
		case "3":
			log.Println("найти спарринг-команду ...")
			// selectLocation(fromID)
		default:
			log.Println("неизвестный тип команды")
		}
	case "selectCountry":
		log.Println("сохранение страны в кэш...")
		return saveCountryCash(fromID, &data)
	default:
		// считываем из кэша
		fromIDStr := strconv.FormatInt(*fromID, 10)

		obj, err := Client.HGetAll(fromIDStr).Result()

		if err != nil {
			log.Println("не удалось распарсить ключ-", fromIDStr)
		}

		// ждем от пользователя город, после чего сохраняем местоположение
		_, ok := obj["wait_city"]

		if ok {
			countryID, ok := obj["id_country"]
			if ok {
				saveLocation(&countryID, &data, fromID)
				return ""
			}
		}
	}

	return ""
}
