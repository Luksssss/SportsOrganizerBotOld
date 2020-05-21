package main

import (
	"log"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	// _ "github.com/jmoiron/sqlx"
)

// Выбор вида спорта
const (
	ks      = "kindSports|"
	sa      = "selectAct|"     // выбор действия из 3 (создание команды и т.д.)
	sc      = "selectCountry|" // выбор страны и сохранение в кэш
	scity   = "selectCity|"    // выбор и сохранение страны-города
	nm      = "nameTeam|"      // выбор имени команды
	nothing = "nothing|"       // заглушка (например, для кпноки Нет)
)

// Action Структура для хранения реакций на действия
type Action struct {
	id          int8
	TextMsg     *string
	ReplyMarkup interface{}
}

type ParamsSQL struct {
	id   *int
	name string
}

// Набор КНОПОК
var (
	kindSports       tgbotapi.InlineKeyboardMarkup
	selectAction3key tgbotapi.InlineKeyboardMarkup
	fromCountry      tgbotapi.InlineKeyboardMarkup

	// caseAcrion ... структура для опросных сообщений
	// caseAction = make(map[string]*Action)

	caseAction = map[string]*Action{
		"inputFewLetter": &Action{id: 3, TextMsg: &fewLetterCity},
	}
)

func initButtons() {

	emptyParams := &ParamsSQL{}

	// Выбор вида спорта
	kindSports, err := getButtons(&SelectSports, ks, 2, emptyParams)
	if err != nil {
		log.Println("проблемы при создании кнопок kindSports")
	}
	caseAction["kindSports"] = &Action{0, &welcomeText, &kindSports}

	// выбор действия (создать команду, найти людей...)
	selectAction3key = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("1", sa+"1"),
			tgbotapi.NewInlineKeyboardButtonData("2", sa+"2"),
			tgbotapi.NewInlineKeyboardButtonData("3", sa+"3"),
		),
	)
	caseAction["selectAct"] = &Action{1, &actionSelectText, selectAction3key}

	// выбор страны команды
	fromCountry, err := getButtons(&SelectCountry, sc, 4, emptyParams)
	if err != nil {
		log.Println("проблемы при создании кнопок fromCountry")
	}
	caseAction["fromCountry"] = &Action{2, &countrySelect, &fromCountry}

}

func getButtonYesNo(prefixYes, prefixNo string) *tgbotapi.InlineKeyboardMarkup {
	// Кнопки да\нет
	уesNo := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Да", prefixYes),
			tgbotapi.NewInlineKeyboardButtonData("Нет", prefixNo),
		),
	)

	return &уesNo
}

// getButtons ... создать группу кнопок
func getButtons(sql *string, prefix string, lenLine uint8, params *ParamsSQL) (*tgbotapi.InlineKeyboardMarkup, error) {

	// TODO: как с Pool использовать sqlx ...
	var (
		id          int
		name, smile string
		i           uint8 // номер кнопки в линии
		resButtons  [][]tgbotapi.InlineKeyboardButton
		tmp         []tgbotapi.InlineKeyboardButton
		parList     []interface{}
	)

	if params.id != nil {
		parList = append(parList, *params.id)
	}

	if params.name != "" {
		parList = append(parList, params.name)
	}

	rows, err := Pool.Query(Ctx, *sql, parList...)
	if err != nil {
		log.Println("проблемы при выборке", *sql, err)
	}

	defer rows.Close()
	for rows.Next() {
		err := rows.Scan(&id, &name, &smile)
		if err != nil {
			log.Println("проблема с разбором строчки")
		}

		i++
		ikb := tgbotapi.NewInlineKeyboardButtonData(smile+" "+name, prefix+strconv.Itoa(id))
		tmp = append(tmp, ikb)

		// lenLine - количество кнопок в линию
		if lenLine <= i {
			resButtons = append(resButtons, tmp)
			tmp = nil
			i = 0
		} else {
			i++
		}

		//log.Println("строчка: ", id, name, smile)

	}
	// если не хватило на всю линию, то выводим тут
	if i > 0 {
		resButtons = append(resButtons, tmp)
	}

	newButtons := &tgbotapi.InlineKeyboardMarkup{
		InlineKeyboard: resButtons,
	}

	return newButtons, nil

}
