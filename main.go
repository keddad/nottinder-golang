package main

import (
	"fmt"
	"github.com/go-pg/pg/v10"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"log"
	"os"
	"unicode/utf8"
)

func main() {
	userCache := make(map[int64]*User)
	userState := make(map[int64]State)
	currentProposal := make(map[int64]int64)

	token := os.Getenv("TOKEN")
	bot, _ := tgbotapi.NewBotAPI(token)

	db := pg.Connect(&pg.Options{
		Addr:     "postgres:5432",
		User:     "postgres",
		Password: "pass",
	})
	defer db.Close()
	err := CreateSchema(db)
	if err != nil {
		fmt.Print(err.Error())
		panic(err)
	}

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, _ := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil { // ignore any non-Message Updates
			continue
		}

		if update.Message.IsCommand() { // Handle commands
			state, ok := CommandToState[update.Message.Text]

			if !ok {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Такой команды нет. Может, ты где то ошибся?")
				msg.ReplyToMessageID = update.Message.MessageID
				bot.Send(msg)
				continue
			}

			if state == ReceivePair && !UserRegistered(db, update.Message.Chat.ID) {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Это команда для получения следующей пары. Ее нельзя выполнить до первичной регистрации.")
				msg.ReplyToMessageID = update.Message.MessageID
				bot.Send(msg)
				continue
			}

			userState[update.Message.Chat.ID] = state

		}

		switch userState[update.Message.Chat.ID] {
		case 0:
			if UserRegistered(db, update.Message.Chat.ID) {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Что то сломалось! Напиши /next для получения новой пары")
				userState[update.Message.Chat.ID] = ReceivePair
				msg.ReplyToMessageID = update.Message.MessageID
				bot.Send(msg)
				continue
			}
			fallthrough
		case Greet:
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Привет! Как тебя зовут? (Всю эту информацию ты сможешь поменять потом командой /change)")
			bot.Send(msg)
			userState[update.Message.Chat.ID] = GetName
		case GetName:
			name := update.Message.Text

			if len(name) == 0 {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Имя должно быть строкой")
				msg.ReplyToMessageID = update.Message.MessageID
				bot.Send(msg)
				continue
			}

			if utf8.RuneCountInString(name) > 256 {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Очень большой текст! Уместись в 256 символов")
				msg.ReplyToMessageID = update.Message.MessageID
				bot.Send(msg)
				continue
			}

			userCache[update.Message.Chat.ID] = &User{Name: name, ChatId: update.Message.Chat.ID}
			userState[update.Message.Chat.ID] = GetBio

			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Отлично! Теперь расскажи о себе")
			msg.ReplyToMessageID = update.Message.MessageID
			bot.Send(msg)

		case GetBio:
			bio := update.Message.Text

			if len(bio) == 0 {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Био должно быть строкой")
				msg.ReplyToMessageID = update.Message.MessageID
				bot.Send(msg)
				continue
			}

			if utf8.RuneCountInString(bio) > 256 {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Очень большой текст! Уместись в 256 символов")
				msg.ReplyToMessageID = update.Message.MessageID
				bot.Send(msg)
				continue
			}

			userCache[update.Message.Chat.ID].Bio = bio
			userState[update.Message.Chat.ID] = GetPhoto

			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Теперь мне нужна твоя фоточка. Если стесняешься, подойдет фото стены")
			msg.ReplyToMessageID = update.Message.MessageID
			bot.Send(msg)

		case GetPhoto:
			photo := update.Message.Photo

			if photo == nil {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Отправь мне фотографию, бака!")
				msg.ReplyToMessageID = update.Message.MessageID
				bot.Send(msg)
				continue
			}

			userCache[update.Message.Chat.ID].PhotoId = (*photo)[0].FileID
			userState[update.Message.Chat.ID] = GetGender

			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Отлично. Поговорим о поле:")
			msg.ReplyMarkup = GenderKeyboard
			bot.Send(msg)

		case GetGender:
			genderInupt := update.Message.Text

			genderOption, ok := NameToGender[genderInupt]
			if !ok {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Нажми на кнопочку, бака!")
				msg.ReplyToMessageID = update.Message.MessageID
				bot.Send(msg)
				continue
			}

			userCache[update.Message.Chat.ID].Gender = genderOption
			userState[update.Message.Chat.ID] = GetOrientation

			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "А теперь о ориентации:")
			msg.ReplyMarkup = OrientationKeyboard
			bot.Send(msg)

		case GetOrientation:
			orientationInput := update.Message.Text

			orientationOption, ok := NameToOrientation[orientationInput]
			if !ok {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Нажми на кнопочку, бака!")
				msg.ReplyToMessageID = update.Message.MessageID
				bot.Send(msg)
				continue
			}

			userCache[update.Message.Chat.ID].Orientation = orientationOption
			userState[update.Message.Chat.ID] = ReceivePair

			InsertOrUpdate(db, userCache[update.Message.Chat.ID])
			delete(userCache, update.Message.Chat.ID)
			fallthrough // После завершения регистрации переходим к поиску пары
		case ReceivePair:
			ReceivePairHandler(db, bot, &update, &userState, &currentProposal)

		case GetPairOpinion:
			opinion := update.Message.Text

			proposal, ok := currentProposal[update.Message.Chat.ID]

			if !ok {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Что то сломалось! Напиши /next для получения новой пары")
				userState[update.Message.Chat.ID] = ReceivePair
				msg.ReplyToMessageID = update.Message.MessageID
				bot.Send(msg)
				continue
			}

			if opinion != "<3" && opinion != ":(" {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Нажми на кнопочку, бака!")
				msg.ReplyToMessageID = update.Message.MessageID
				bot.Send(msg)
				continue
			}

			match := InsertPair(db, update.Message.Chat.ID, proposal, opinion == "<3")

			if match {
				HandleMatch(bot, update.Message.Chat.ID, proposal)
			}

			userState[update.Message.Chat.ID] = ReceivePair
			ReceivePairHandler(db, bot, &update, &userState, &currentProposal)
		}

	}

}
