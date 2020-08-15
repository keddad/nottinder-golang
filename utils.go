package main

import (
	"fmt"
	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
	"github.com/go-telegram-bot-api/telegram-bot-api"
)

type State int

type User struct {
	Name, PhotoId, Bio  string
	ChatId              int64 `pg:",unique,notnull,pk"`
	Gender, Orientation int
}

type Pair struct {
	Aid   int64
	Bid   int64
	Match bool
}

func HandleMatch(bot *tgbotapi.BotAPI, fromId int64, toId int64) {
	msg := tgbotapi.NewMessage(fromId, fmt.Sprintf("It's a match! Скорее пиши [этому котику](tg://user?id=%d).", toId))
	msg.ParseMode = tgbotapi.ModeMarkdown
	bot.Send(msg)

	msg = tgbotapi.NewMessage(toId, fmt.Sprintf("It's a match! Скорее пиши [этому котику](tg://user?id=%d).", fromId))
	msg.ParseMode = tgbotapi.ModeMarkdown
	bot.Send(msg)
}

func CreateSchema(db *pg.DB) error {
	models := []interface{}{
		(*User)(nil),
		(*Pair)(nil),
	}

	for _, model := range models {
		err := db.Model(model).CreateTable(&orm.CreateTableOptions{
			IfNotExists: true,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func UserRegistered(db *pg.DB, chatID int64) bool {
	user := &User{ChatId: chatID}
	err := db.Model(user).Where("chat_id = ?::bigint", chatID).Select()

	return err == nil
}

func InsertOrUpdate(db *pg.DB, user *User) {
	if ex, _ := db.Model(user).Where("chat_id = ?::bigint", user.ChatId).Exists(); ex {
		db.Model(user).Where("chat_id = ?::bigint", user.ChatId).Update(user)
	} else {
		db.Model(user).Insert(user)
	}
}

func GetPair(db *pg.DB, userId int64) (User, error) {
	var user, currentUser User

	err := db.Model(&currentUser).Where("chat_id = ?::bigint", userId).First()
	if err != nil {
		return User{}, err
	}

	err = db.Model(&user).Where("chat_id != ? AND chat_id NOT IN (SELECT bid FROM pairs WHERE aid = ?::bigint) AND gender = ANY(ARRAY[?]::bigint[])", userId, userId, pg.In((*OrientationGenderToPossibleGender[currentUser.Orientation])[currentUser.Gender])).First()
	if err != nil {
		return User{}, err
	}
	return user, nil
}

func InsertPair(db *pg.DB, from int64, to int64, loved bool) bool {
	pair := Pair{
		Aid:   from,
		Bid:   to,
		Match: loved,
	}
	db.Model(&pair).Insert(&pair)

	if !loved {
		return false
	}

	ans, _ := db.Model(&Pair{}).Where("aid = ? AND bid = ? AND match", to, from).Exists()
	return ans
}
func ReceivePairHandler(db *pg.DB, bot *tgbotapi.BotAPI, update *tgbotapi.Update, userState *map[int64]State, currentProposal *map[int64]int64) {
	pair, err := GetPair(db, update.Message.Chat.ID)
	if err != nil {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Нет подходящих пар. Попробуй чуть позже командой /next")
		msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
		bot.Send(msg)
		return
	}

	msg := tgbotapi.NewPhotoShare(update.Message.Chat.ID, pair.PhotoId)
	msg.ReplyMarkup = PairLoveKeyboard
	msg.Caption = fmt.Sprintf("%s \n %s", pair.Name, pair.Bio)
	bot.Send(msg)

	(*userState)[update.Message.Chat.ID] = GetPairOpinion
	(*currentProposal)[update.Message.Chat.ID] = pair.ChatId
}
