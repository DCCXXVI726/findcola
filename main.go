package main

import (
	"database/sql"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	_ "github.com/lib/pq"
	"log"
	"os"
)

var (
	db  *sql.DB
	err error
)

type MyItem struct {
	Name  string  `db:"name"`
	Price float64 `db:"price"`
	Shop  string  `db:"shop"`
}

func main() {
	sqlURL := os.Getenv("DATABASE_URL")
	if sqlURL == "" {
		panic("empty DATABASE_URL")
	}
	db, err = sql.Open("postgres", sqlURL)
	if err != nil {
		panic(err)
	}
	StartBot()
}

var keyboard = tgbotapi.NewReplyKeyboard(
	tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("Найти колу")))

func findMin() string {
	rows, err := db.Query("select product.name, product.price, product.shop from product " +
		"inner join (select DISTINCT min(priceperliter) as minprice from product) minprices " +
		"on product.priceperliter = minprices.minprice " +
		"where product.sugar = false order by product.cap")
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	items := make([]MyItem, 0)
	for rows.Next() {
		var item MyItem
		err = rows.Scan(&item.Name, &item.Price, &item.Shop)
		if err != nil {
			panic(err)
		}
		items = append(items, item)
	}
	msg := "Самая дешевая кола сейчас:\n"
	for i, item := range items {
		str := fmt.Sprintf("%d. %s - %.2f в %s", i+1, item.Name, item.Price, item.Shop)
		msg = msg + str
		if i != len(items)-1 {
			msg = msg + "\n"
		}
	}
	return msg
}

func StartBot() {
	token := os.Getenv("TELEGRAM_TOKEN")
	if token == "" {
		panic("empty TELEGRAM_TOKEN")
	}
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true

	log.Printf("Authorized on account %s", bot.Self.UserName)
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, update.Message.Text)
		switch update.Message.Text {

		case "start":
			msg.ReplyMarkup = keyboard
			bot.Send(msg)
		case "Найти колу":
			msg.Text = findMin()
			bot.Send(msg)
		default:
			msg.ReplyToMessageID = update.Message.MessageID
			//log := fmt.Sprintf("%f", update.Message.Location.Longitude)
			//lat := fmt.Sprintf("%f", update.Message.Location.Latitude)
			//msg = tgbotapi.NewMessage(update.Message.Chat.ID, log+" "+lat)
			bot.Send(msg)

		}
	}
}
