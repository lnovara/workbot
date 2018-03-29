package api

import (
	"context"
	"io/ioutil"
	"net/http"

	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	sheets "google.golang.org/api/sheets/v4"
)

var (
	oAuthConfig *oauth2.Config
)

// NewOAuthConfig initialize a new OAuth config
func NewOAuthConfig(clientSecretPath string) {
	cs, err := ioutil.ReadFile(clientSecretPath)
	if err != nil {
		logrus.Fatalf("Could not read Google client secrets from %s: %s", clientSecretPath, err.Error())
	}

	oAuthConfig, err = google.ConfigFromJSON(cs, sheets.SpreadsheetsScope)
	if err != nil {
		logrus.Fatalf("Could not initialize Google OAuth2 Client: %s", err.Error())
	}
}

func getToken(authCode string) (*oauth2.Token, error) {
	return oAuthConfig.Exchange(context.TODO(), authCode)
}

func newOAuthClientFromAuthCode(authCode string) *http.Client {
	token, err := getToken(authCode)
	if err != nil {
		logrus.Fatalf("Could not get OAuth token: %s", err.Error())
	}
	return oAuthConfig.Client(context.Background(), token)
}

func newOAuthClientFromToken(token *oauth2.Token) *http.Client {
	return oAuthConfig.Client(context.Background(), token)
}

func authCodeURL() string {
	return oAuthConfig.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
}
