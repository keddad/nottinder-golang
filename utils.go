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
	msg := tgbotapi.NewMessage(fromId, fmt.Sprintf("It's a match! Скорее пиши [этому котику]{tg://user?id=%d}.", toId))
	bot.Send(msg)

	msg = tgbotapi.NewMessage(toId, fmt.Sprintf("It's a match! Скорее пиши [этому котику]{tg://user?id=%d}.", fromId))
	bot.Send(msg)
}

func CreateSchema(db *pg.DB) error {
	models := []interface{}{
		(*User)(nil),
		(*Pair)(nil),
	}

	for _, model := range models {
		err := db.Model(model).CreateTable(&orm.CreateTableOptions{})
		if err != nil {
			return err
		}
	}
	return nil
}

func UserRegistered(db *pg.DB, chatID int64) bool {
	user := &User{ChatId: chatID}
	err := db.Model(user).Select()

	return err == nil
}

func InsertOrUpdate(db *pg.DB, user *User) {
	db.Model(user).Update()
}

func GetPair(db *pg.DB, userId int64) (User, error) {
	var user, currentUser User
	db.Model(&currentUser).Where("chat_id == ?", userId).First()
	err := db.Model(user).Where("chat_id NOT IN (SELECT b_id FROM pair WHERE a_id = ?) AND gender IN ?", userId, (*OrientationGenderToPossibleGender[user.Orientation])[user.Gender]).First()
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
	db.Model(pair).Insert(pair)

	if !loved {
		return false
	}

	ans, _ := db.Model(Pair{}).Where("a_id == ? AND b_id == ? AND match", to, from).Exists()
	return ans
}
