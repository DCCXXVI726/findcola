package main

import (
	"database/sql"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	_ "github.com/lib/pq"
	"log"
	"net/http"
	"os"
	"time"
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

func MainHandler(resp http.ResponseWriter, _ *http.Request) {
	resp.Write([]byte("Hi there! I'm DndSpellsBot!"))
}

func main() {
	http.HandleFunc("/", MainHandler)
	go http.ListenAndServe(":"+os.Getenv("PORT"), nil)

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
	tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("Найти колу"), tgbotapi.NewKeyboardButton("Все магазины")))

func findMin() string {
	rows, err := db.Query("select product.name, product.price, product.shop, product.update_time from product " +
		"inner join (select DISTINCT min(priceperliter) as minprice from product) minprices " +
		"on product.priceperliter = minprices.minprice " +
		"where product.sugar = false order by product.cap")
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	items := make([]MyItem, 0)
	var updateTime time.Time
	for rows.Next() {
		var item MyItem
		err = rows.Scan(&item.Name, &item.Price, &item.Shop, &updateTime)
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
	msg = msg + "\nОбновлено: " + updateTime.Format("02-01-2006")
	return msg
}

func findMinParametrs(type_cola string, sugar string) string {
	query := "select product.name, product.price, product.shop, product.update_time" +
		"from product INNER JOIN (select DISTINCT min(priceperliter) as my_price, shop "
	if type_cola != "all" {
		query = query + ", curr_typecola "
	}
	if sugar != "nil" {
		query = query + ", sugar "
	}

	if type_cola == "all" && sugar == "nil" {
		query = query + "from product "
	} else {
		query = query + "from (select * from product where "
		flag := 0
		if type_cola != "all" {
			flag = 1
			query = query + "curr_typecola='" + type_cola + "' "
		}
		if sugar != "nil" {
			if flag == 1 {
				query = query + " and "
			}
			query = query + "sugar=" + sugar
		}
		query = query + ") as foo "
	}
	query = query + "group by shop "
	if type_cola != "all" {
		query = query + ", curr_typecola "
	}
	if sugar != "nil" {
		query = query + ", sugar "
	}
	query = query + "order by shop) min_price on min_price.my_price=product.priceperliter and min_price.shop=product.shop "
	if type_cola != "all" {
		query = query + "and min_price.curr_typecola=product.curr_typecola "
	}
	if sugar != "nil" {
		query = query + "and min_price.sugar=product.sugar "
	}
	query = query + "order by product.shop;"
	log.Println(query)
	rows, err := db.Query(query)
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	return rowsToStr(rows)
}

func rowsToStr(rows *sql.Rows) string {
	items := make([]MyItem, 0)
	updateTime := time.Now()
	for rows.Next() {
		var (
			item   MyItem
			upTime time.Time
		)

		err = rows.Scan(&item.Name, &item.Price, &item.Shop, &upTime)
		if err != nil {
			panic(err)
		}
		items = append(items, item)
		if updateTime.After(upTime) {
			updateTime = upTime
		}
	}
	msg := "Самая дешевая кола сейчас:\n"
	shop := ""
	k := 0
	i := 1
	for _, item := range items {
		if item.Shop != shop {
			k++
			str := fmt.Sprintf("%d. %s\n", k, item.Shop)
			msg = msg + str
			i = 1
			shop = item.Shop
		}
		str := fmt.Sprintf("%d. %s - %.2f\n", i, item.Name, item.Price)
		msg = msg + str
		i++
	}
	msg = msg + "Обновлено: " + updateTime.Format("02-01-2006")
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

	//updates := bot.GetUpdatesChan(u)
	updates := bot.ListenForWebhook("/" + bot.Token)
	for update := range updates {
		if update.Message == nil {
			continue
		}

		log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, update.Message.Text)
		msg.ReplyMarkup = keyboard
		switch update.Message.Text {
		case "/start":
			msg.Text = "Теперь можешь нажать кнопку"

			bot.Send(msg)
		case "Найти колу":
			msg.Text = findMin()
			bot.Send(msg)
		case "Все магазины":
			msg.Text = findMinParametrs("all", "nil")
			log.Println(msg.Text)
			bot.Send(msg)
		case "Все магазины cola":
			msg.Text = findMinParametrs("cola", "nil")
			bot.Send(msg)
		case "Все магазины pepsi":
			msg.Text = findMinParametrs("pepsi", "nil")
			bot.Send(msg)
		case "Все магазины cola без сахара":
			msg.Text = findMinParametrs("cola", "false")
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
