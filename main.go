package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/lnovara/workbot/api"
	"github.com/lnovara/workbot/userdb"
	"github.com/lnovara/workbot/version"
	"github.com/sirupsen/logrus"
)

const (
	// BANNER is printed for help/info output
	BANNER = `
                    _    _           _
__      _____  _ __| | _| |__   ___ | |_
\ \ /\ / / _ \| '__| |/ / '_ \ / _ \| __|
 \ V  V / (_) | |  |   <| |_) | (_) | |
  \_/\_/ \___/|_|  |_|\_\_.__/ \___/ \__|

 A Telegram bot that help you keep track of your working hours.
 Version: %s
 Build: %s

`
)

var (
	dbFilePath                 string
	googleAPIKey               string
	googleClientSecretFilePath string
	telegramToken              string

	debug bool
)

func init() {
	flag.StringVar(&dbFilePath, "db", "./workbot-users.sqlite3", "User database path")
	flag.StringVar(&googleAPIKey, "google-api-key", os.Getenv("GOOGLE_API_KEY"), "Google API key (or env var GOOGLE_API_KEY)")
	flag.StringVar(&googleClientSecretFilePath, "google-client-secrets", "./client_secrets.json", "Path to Google's client_secret.json file")
	flag.StringVar(&telegramToken, "telegram-token", os.Getenv("TELEGRAM_TOKEN"), "Telegram API token (or env var TELEGRAM_TOKEN)")

	flag.BoolVar(&debug, "d", false, "run in debug mode")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, fmt.Sprintf(BANNER, version.VERSION, version.GITCOMMIT))
		flag.PrintDefaults()
	}

	flag.Parse()

	if debug {
		logrus.SetLevel(logrus.DebugLevel)
	}

	if googleAPIKey == "" {
		usageAndExit("Google API key cannot be empty.", 1)
	}

	if telegramToken == "" {
		usageAndExit("Telegram API key cannot be empty.", 1)
	}
}

func main() {
	var err error

	logrus.Info("Welcome to WorkBot!!!")

	api.NewTelegramBot(telegramToken, debug)
	if err != nil {
		logrus.Fatalf("Could not initialize Telegram Bot API: %s", err.Error())
	}

	logrus.Debug("Telegram Bot API initialization done")

	err = userdb.NewUserDB(dbFilePath)
	if err != nil {
		logrus.Fatalf("Could not create user database %s: %s", dbFilePath, err.Error())
	}

	logrus.Debug("Database initialization done")

	err = api.NewMapsClient(googleAPIKey)
	if err != nil {
		logrus.Fatalf("Could not initialize Google Maps API: %s", err.Error())
	}

	logrus.Debug("Google Maps client initialization done")

	api.NewOAuthConfig(googleClientSecretFilePath)
	if err != nil {
		logrus.Fatalf("Could not initialize Google OAuth2 config: %s", err.Error())
	}

	logrus.Debug("OAuth config initialization done")

	api.HandleBotUpdates()
}

func usageAndExit(message string, exitCode int) {
	if message != "" {
		fmt.Fprintf(os.Stderr, message)
		fmt.Fprintf(os.Stderr, "\n\n")
	}

	flag.Usage()
	fmt.Fprintf(os.Stderr, "\n")
	os.Exit(exitCode)
}
