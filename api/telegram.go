package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/lnovara/workbot/types"
	"github.com/lnovara/workbot/userdb"
	"github.com/sirupsen/logrus"
)

const (
	back                   = "🔙"
	changeAccessTimeFormat = changeAccessTime + " (da %s - %s)"
	changeAccessTime       = "🕙 Modifica orario d'ingresso"
	changeLocationFormat   = changeLocation + " (da %s)"
	changeLocation         = "🌍 Modifica fuso orario"
	editSettings           = "🔧 Impostazioni"
	sendLocation           = "🌍 Invia posizione"
	workEnd                = "Uscita"
	workStart              = "Ingresso"
)

var (
	telegramBot *tgbotapi.BotAPI
)

// NewTelegramBot initializes a new Telegram bot
func NewTelegramBot(telegramToken string, debug bool) error {
	var err error
	telegramBot, err = tgbotapi.NewBotAPI(telegramToken)
	telegramBot.Debug = debug
	return err
}

// HandleBotUpdates handles bot updates
func HandleBotUpdates() {
	uc := tgbotapi.NewUpdate(0)
	uc.Timeout = 60

	updates, err := telegramBot.GetUpdatesChan(uc)
	if err != nil {
		logrus.Fatalf("Could not get bot updates chan: %s", err.Error())
	}

	for u := range updates {
		if u.Message == nil {
			logrus.Infof("nil message")
			continue
		}

		msg := u.Message

		logrus.Debugf("[%d] %s: '%s'", msg.MessageID, msg.From, msg.Text)

		var user *types.User
		user, err = userdb.GetUser(msg.From.ID)
		if err != nil && err != sql.ErrNoRows {
			logrus.Fatalf("Could not get user '%d': %s", msg.From.ID, err.Error())
		} else if err == sql.ErrNoRows {
			user = types.NewUser()
			user.Id = msg.From.ID
			user.FirstName = msg.From.FirstName
			err = userdb.InsertUser(user)
			if err != nil {
				logrus.Fatalf("Could not add user '%d': %s", msg.From.ID, err.Error())
			}
		}

		// TODO: use Command() for bot command handling
		if msg.Text == "/start" {
			msg = nil
			user.State = types.UserSetupTimezone
		} else if msg.Text == "/enter" || msg.Text == workStart {
			user.State = types.Enter
		} else if msg.Text == "/exit" || msg.Text == workEnd {
			user.State = types.Exit
		} else if msg.Text == "/settings" || msg.Text == editSettings {
			user.State = types.Settings
		} else if strings.HasPrefix(msg.Text, changeAccessTime) {
			msg = nil
			user.State = types.SetAccessTime
		} else if strings.HasPrefix(msg.Text, changeLocation) {
			user.State = types.SetTimezone
		} else if msg.Text == back {
			user.State = types.Main
		}

		err = userdb.UpdateUser(user)
		if err != nil {
			logrus.Fatalf("Could not update user '%d': %s", user.Id, err.Error())
		}
		handleMessage(user, msg)
	}
}

func handleMessage(user *types.User, msg *tgbotapi.Message) {
	switch user.State {
	case types.Main:
		handleMain(user, msg)
	case types.Enter:
		handleEnter(user, msg)
	case types.Exit:
		handleExit(user, msg)
	case types.Settings:
		handleSettings(user, msg)
	case types.SetTimezone:
		fallthrough
	case types.UserSetupTimezone:
		handleUserSetupTimeZone(user, msg)
	case types.UserSetupClientSecret:
		handleUserSetupClientSecret(user, msg)
	case types.SetAccessTime:
		fallthrough
	case types.UserSetupAccessTime:
		handleUserSetupAccessTime(user, msg)
	default:
		reply(user, "state %d unknown/unhandled. Terminating", user.State)
		logrus.Fatalf("state %d unknown", user.State)
	}
}

func handleMain(user *types.User, msg *tgbotapi.Message) {
	kb := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(workStart),
			tgbotapi.NewKeyboardButton(workEnd),
			tgbotapi.NewKeyboardButton(editSettings),
		),
	)

	mc := createReply(user, "Scegli quale operazione effettuare.")
	mc.ReplyMarkup = kb
	telegramBot.Send(mc)
}

func handleEnter(user *types.User, msg *tgbotapi.Message) {
	err := appendEnterTime(user, time.Unix(int64(msg.Date), 0))
	if err != nil {
		if err == errAlreadyEnter {
			reply(user, "Oggi hai già effettuato l'ingresso, quante volte vuoi entrare?! Vai a lavorare!")
		} else {
			logrus.Fatal(err)
		}
	}
	// FIXME: give feedback about successful insertion and theoretical exit time
	user.State = types.Main
	userdb.UpdateUser(user)
	handleMessage(user, nil)
}

func handleExit(user *types.User, msg *tgbotapi.Message) {
	err := appendExitTime(user, time.Unix(int64(msg.Date), 0))
	if err != nil {
		if err == errNoEnter {
			reply(user, "Oggi non hai ancora effettuato l'ingresso. Devi entrare prima di poter uscire, no?!")
		} else if err == errAlreadyExit {
			reply(user, "Oggi hai già effettuato l'uscita, quante volte vuoi uscire?! Goditi la serata!")
		} else {
			logrus.Fatal(err)
		}
	} else {
		reply(user, "Uscita effettuata con successo. Buona serata!")
	}
	user.State = types.Main
	userdb.UpdateUser(user)
	handleMessage(user, nil)
}

func handleSetAccessTime(user *types.User, msg *tgbotapi.Message) {
	logrus.Panic("HandleSetAccessTime not implemented!")
}

func handleSetTimeZone(user *types.User, msg *tgbotapi.Message) {
	logrus.Panic("HandleSetTimeZone not implemented!")
}

func handleSettings(user *types.User, msg *tgbotapi.Message) {
	kb := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(back),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(fmt.Sprintf(changeAccessTimeFormat, user.AccessStart.Format("15:04"), user.AccessEnd.Format("15:04"))),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(fmt.Sprintf(changeLocationFormat, user.TimeZone)),
		),
	)
	kb.OneTimeKeyboard = true
	mc := createReply(user, "Quali impostazioni vuoi modificare?")
	mc.ReplyMarkup = kb
	telegramBot.Send(mc)
}

func handleUserSetupAccessTime(user *types.User, msg *tgbotapi.Message) {
	kb := tgbotapi.NewReplyKeyboard()
	kb.OneTimeKeyboard = true
	start := time.Date(2000, time.January, 1, 8, 0, 0, 0, time.UTC)
	for i := 0; i < 5; i++ {
		end := start.Add(30 * time.Minute)
		kb.Keyboard = append(kb.Keyboard, tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(fmt.Sprintf("%s - %s", start.Format("15:04"), end.Format("15:04"))),
		))
		start = start.Add(30 * time.Minute)
	}

	errMsg := createReply(user, "Non riesco a capire cosa hai scritto, prova di nuovo.")
	errMsg.ReplyMarkup = kb

	if msg == nil {
		var mc tgbotapi.MessageConfig
		if user.State == types.UserSetupAccessTime {
			mc = createReply(user, "Per ultimo, scegli il tuo orario d'ingresso.")
		} else if user.State == types.SetAccessTime {
			mc = createReply(user, "Per favore, scegli il tuo orario d'ingresso.")
		} else {
			logrus.Panicf("Unexpected state '%d'", user.State)
		}
		mc.ReplyMarkup = kb
		telegramBot.Send(mc)
	} else {
		patt := "(.*) - (.*)"
		m, err := regexp.MatchString(patt, msg.Text)
		if err != nil {
			logrus.Fatal(err)
		}
		if !m {
			telegramBot.Send(errMsg)
			return
		}
		re := regexp.MustCompile(patt)
		matches := re.FindStringSubmatch(msg.Text)
		accessStart := matches[1]
		accessEnd := matches[2]
		accessStartTime, err := time.Parse("15:04", accessStart)
		if err != nil {
			telegramBot.Send(errMsg)
			return
		}
		accessEndTime, err := time.Parse("15:04", accessEnd)
		if err != nil {
			telegramBot.Send(errMsg)
			return
		}
		reply(user, "Grazie! Hai scelto la fascia oraria %s.", msg.Text)
		if user.State == types.UserSetupAccessTime {
			reply(user, "Congratulazioni %s, ora puoi iniziare ad usare Workbot!", user.FirstName)
		}
		user.AccessStart = accessStartTime
		user.AccessEnd = accessEndTime
		user.State = types.Main
		userdb.UpdateUser(user)
		handleMessage(user, nil)
	}
}

func handleUserSetupClientSecret(user *types.User, msg *tgbotapi.Message) {
	if msg == nil {
		reply(user, "Per registrare i tuoi orari lavorativi, ho bisogno che tu mi dia l'autorizzazione per accedere a Google Sheets.")
		reply(user, "Per favore, visita il link e inviami il codice d'autorizzazione: %s", authCodeURL())
	} else {
		token, err := getToken(msg.Text)
		if err != nil {
			reply(user, "Non è stato possibile ottenere il codice di autorizzazione. Riprova.")
			handleMessage(user, nil)
			return
		}
		clientSecret, err := json.Marshal(token)
		if err != nil {
			logrus.Fatalf("Could not marshal client secret: %s", err.Error())
		}
		user.ClientSecret = clientSecret
		err = userdb.UpdateUser(user)
		if err != nil {
			logrus.Fatalf("Could not update user '%d': %s", user.Id, err.Error())
		}
		reply(user, "Autorizzazione avvenuta con successo!")
		reply(user, "Adesso creo un nuovo foglio di calcolo sul tuo Google Drive.")
		reply(user, "Potrebbe volerci qualche secondo...")
		newSheetsClient(user)
		sheetId, sheetUrl, err := createSpreadsheet(user)
		if err != nil {
			logrus.Fatalf("Could not create spreadsheet: %s", err.Error())
		}
		reply(user, "Fatto!")
		reply(user, "Troverai i tuoi orari lavorativi registrati all'indirizzo: %s", sheetUrl)
		user.SheetId = sheetId
		user.State = types.UserSetupAccessTime
		userdb.UpdateUser(user)
		handleMessage(user, nil)
	}
}

func handleUserSetupTimeZone(user *types.User, msg *tgbotapi.Message) {
	var mc tgbotapi.MessageConfig

	kb := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButtonLocation(sendLocation),
		),
	)
	kb.OneTimeKeyboard = true

	if msg == nil {
		reply(user, "Ciao %s!", user.FirstName)
		mc = createReply(user, "Per iniziare, iniviami la tua posizione, così che possa determinare il tuo fuso orario!")
		mc.ReplyMarkup = kb
		telegramBot.Send(mc)
	} else if msg.Location == nil {
		mc = createReply(user, "Per favore, inviami la tua posizione.")
		mc.ReplyMarkup = kb
		telegramBot.Send(mc)
	} else {
		tzId, err := timezone(msg.Location.Latitude, msg.Location.Longitude)
		if err != nil {
			logrus.Fatalf("Could not get timezone: %s", err.Error())
		}
		user.TimeZone = tzId
		if user.State == types.SetTimezone {
			user.State = types.Main
		} else if user.State == types.UserSetupTimezone {
			user.State = types.UserSetupClientSecret
		} else {
			logrus.Panicf("Unexpected state '%d'", user.State)
		}
		err = userdb.UpdateUser(user)
		if err != nil {
			logrus.Fatalf("Could not update user %d: %s", user.Id, err.Error())
		}
		reply(user, "Grazie! Il tuo fuso orario è '%s'.", tzId)
		handleMessage(user, nil)
	}
}

func createReply(user *types.User, format string, data ...interface{}) tgbotapi.MessageConfig {
	return tgbotapi.NewMessage(int64(user.Id), fmt.Sprintf(format, data...))
}

func reply(user *types.User, format string, data ...interface{}) {
	_, err := telegramBot.Send(createReply(user, format, data...))
	if err != nil {
		logrus.Fatalf("Could not send reply to '%d': %s", user.Id, err.Error())
	}
}
