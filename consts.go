package main

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"

var OrientationGenderToPossibleGender = map[int]*map[int][]int{
	Straight: {
		Male:    {Female},
		Female:  {Male},
		Unknown: {Male, Female, Unknown},
	},
	Gay: {
		Male:    {Male},
		Female:  {Female},
		Unknown: {Male, Female, Unknown},
	},
	Bi: {
		Male:    {Male, Female, Unknown},
		Female:  {Male, Female, Unknown},
		Unknown: {Male, Female, Unknown},
	},
}

const (
	Greet State = 1 + iota
	GetName
	GetBio
	GetPhoto
	GetGender
	GetOrientation
	ReceivePair
	GetPairOpinion
)

const ( // Orientation
	Straight = 1 + iota
	Gay
	Bi
)

const ( // Gender
	Male = 1 + iota
	Female
	Unknown
)

var CommandToState = map[string]State{
	"/start":  Greet,
	"/change": Greet,
	"/next":   ReceivePair,
}

var NameToGender = map[string]int{
	"Парень":     Male,
	"Девушка":    Female,
	"¯\\_(ツ)_/¯": Unknown,
}

var NameToOrientation = map[string]int{
	"Натурал":       Straight,
	"Гей/Лесбиянка": Gay,
	"Если он милый, то какая разница?": Bi,
}

var GenderKeyboard = tgbotapi.NewReplyKeyboard(
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("Парень"),
		tgbotapi.NewKeyboardButton("Девушка"),
		tgbotapi.NewKeyboardButton("¯\\_(ツ)_/¯"),
	),
)

var OrientationKeyboard = tgbotapi.NewReplyKeyboard(
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("Натурал"),
	), tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("Гей/Лесбиянка"),
	), tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("Если он милый, то какая разница?"),
	),
)

var PairLoveKeyboard = tgbotapi.NewReplyKeyboard(
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("<3"),
		tgbotapi.NewKeyboardButton(":("),
	),
)
