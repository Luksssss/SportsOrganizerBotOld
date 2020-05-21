package main

import (
	"context"
	"log"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	pgx "github.com/jackc/pgx/v4"
)

var (
	ctx = context.Background()
)

// FromID ... вернуть id юзера
func FromID(update tgbotapi.Update) int64 {
	if update.Message != nil {
		return int64(update.Message.From.ID)
	} else if update.CallbackQuery != nil {
		return int64(update.CallbackQuery.From.ID)
	} else {
		return 0
	}
}

// selectLocation ... определить страну
func selectLocation(fromID *int64) {

	msg := tgbotapi.NewMessage(*fromID, *caseAction["fromCountry"].TextMsg)
	msg.ReplyMarkup = caseAction["fromCountry"].ReplyMarkup

	bot.Send(msg)
}

// saveKindSports ... выбор вида спорта и сохранение в бд
func saveKindSports(cq *tgbotapi.CallbackQuery, data *string, fromID *int64) string {

	var (
		idUser int64
		et     *ErrTelega
	)

	// транзакции
	tx, err := Pool.BeginTx(Ctx, pgx.TxOptions{})
	if err != nil {
		et = &ErrTelega{fromID: fromID, code: 22, err: err}
		errMessage(et)
	}
	err = tx.QueryRow(Ctx, InsertOrSelectUser, *fromID, cq.From.UserName).Scan(&idUser)

	if err == pgx.ErrNoRows {
		// не вернули id пользователя
		et = &ErrTelega{fromID: fromID, code: 23, err: err, table: "users", field: "idUser"}
		errMessage(et)
		return ""
	} else if err != nil {
		et = &ErrTelega{fromID: fromID, code: 21, err: err, table: "users", field: "idUser"}
		errMessage(et)
		return ""
	}

	idSport, err := strconv.Atoi(*data)
	if err != nil {
		et = &ErrTelega{fromID: fromID, code: 41, err: err, field: "idSport"}
		errMessage(et)

		tx.Rollback(Ctx)
		return ""
	}
	_, err = tx.Exec(Ctx, InsertOrNothingPlayer, idUser, idSport)

	err = tx.Commit(Ctx)
	if err != nil {
		et = &ErrTelega{fromID: fromID, code: 22, err: err, table: "player_info", field: "idUser, idSport"}
		errMessage(et)
	} else {
		log.Println("Пользователь добавлен/найден, id =", idUser)
	}

	return "selectAct"
}

// saveCountryCash ... сохранение страны команды в кэш
func saveCountryCash(fromID *int64, data *string) string {
	var et *ErrTelega

	// Ставим на ожидание ввода букв города от пользователей
	fromIDStr := strconv.FormatInt(*fromID, 10)
	err := Client.HSet(fromIDStr, map[string]interface{}{
		"wait_3_let_city": true,
		"id_country":      *data,
	}).Err()

	if err != nil {
		et = &ErrTelega{fromID: fromID, code: 2, err: err}
		errMessage(et)
		return ""
	}
	// время жизни ключа
	Client.Expire(fromIDStr, Extime)

	// if err != nil {
	// 	log.Println("не удалось установить время уничтожения ключа-", fromIDStr)
	// }

	return "inputFewLetter"
}

// listCity ... делаем фильтр по стране и 3 буквам от юзера и выводим эти города
func listCity(obj *map[string]string, cityLet *string, fromID *int64, fromIDStr *string) {

	var (
		msg     tgbotapi.MessageConfig
		msgText string
		rm      *tgbotapi.InlineKeyboardMarkup
		et      *ErrTelega
	)
	countryID, ok := (*obj)["id_country"]
	if !ok {
		et = &ErrTelega{fromID: fromID, code: 1, field: "idCountry"}
		errMessage(et)
		return
	}

	cid, err := strconv.Atoi(countryID)
	if err != nil {
		et = &ErrTelega{fromID: fromID, code: 41, err: err, field: "idCountry"}
		errMessage(et)
		return
	}

	// TODO: разобрать ошибку
	rm, _ = getButtons(&SelectChoiseCity, scity, 2, &ParamsSQL{&cid, strings.ToLower(*cityLet)})
	if rm.InlineKeyboard != nil {
		msgText = citySelect

		// Ставим на ожидание ввода букв города от пользователей
		Client.HDel(*fromIDStr, "wait_3_let_city")
		err := Client.HSet(*fromIDStr, "wait_city", true).Err()

		if err != nil {
			et = &ErrTelega{fromID: fromID, code: 2, err: err, field: "wait_3_let_city"}
			errMessage(et)
			return
		}

	} else {
		msgText = `Извините, но я пока не работаю в вашем городе, либо вы неверно ввели город/страну.
		Хотите попробовать ещё раз?`
		// ДА - возврат в выбору страны
		// НЕТ - ?
		rm = getButtonYesNo(sa+"1", nothing+"0")
	}

	msg = tgbotapi.NewMessage(*fromID, msgText)
	msg.ReplyMarkup = rm

	bot.Send(msg)
}

// saveLocation ... сохраняем страну и город в базу
func saveLocation(countryID *string, city *string, fromID *int64) {

	var et *ErrTelega

	idCountry, err := strconv.Atoi(*countryID)
	if err != nil {
		et = &ErrTelega{fromID: fromID, code: 41, err: err, field: "idCountry"}
		errMessage(et)
		return
	}

	idCity, err := strconv.Atoi(*city)
	if err != nil {
		et = &ErrTelega{fromID: fromID, code: 41, err: err, field: "idCity"}
		errMessage(et)
		return
	}

	_, err = Pool.Exec(Ctx, UpdateLocUsers, idCountry, idCity, *fromID)

	if err != nil {
		et = &ErrTelega{fromID: fromID, code: 41, err: err, table: "users"}
		errMessage(et)
		return
	}

	fromIDStr := strconv.FormatInt(*fromID, 10)
	Client.HDel(fromIDStr, "wait_city")

	log.Println("Страна и город пользователя обновлена, id_country =", idCountry)

	// переходим к созданию команды (таблица teams)
	// Ставим на ожидание ввода названия команды
	msg := tgbotapi.NewMessage(*fromID, newteamName)
	bot.Send(msg)

	fromIDStr = strconv.FormatInt(*fromID, 10)
	err = Client.HSet(fromIDStr, "wait_team_name", true).Err()

	if err != nil {
		et = &ErrTelega{fromID: fromID, code: 2, err: err, field: "wait_team_name"}
		errMessage(et)
		return
	}

}

// checkTeamName ... проверяем на свободность имени и передаём на следующий вопрос
func checkTeamName(obj *map[string]string, teamName *string, fromID *int64, fromIDStr *string) {
	// Ставим на ожидание ввода букв города от пользователей
	Client.HDel(*fromIDStr, "wait_team_name")
	err := Client.HSet(*fromIDStr, "team_name", true).Err()

	if err != nil {
		et := &ErrTelega{fromID: fromID, code: 2, err: err, field: "wait_team_name"}
		errMessage(et)
		return
	}

	msg := tgbotapi.NewMessage(*fromID, "Поймали.")
	bot.Send(msg)

}
