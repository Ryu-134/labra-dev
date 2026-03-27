package services

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

var oauthConfig = &oauth2.Config{}

// Note having verifier as a global is extremly dangerous, will move to db when I get to it
var verifier string

func InitOauth(gh_client, gh_secret string) {
	oauthConfig = &oauth2.Config{
		ClientID:     gh_client,
		ClientSecret: gh_secret,
		Scopes:       []string{"repo", "user"},
		Endpoint:     github.Endpoint,
		RedirectURL:  "http://localhost:8080/v1/callback",
	}
}

func Authenticate(w http.ResponseWriter, r *http.Request) error {
	fmt.Println("CLIENT ID:", oauthConfig.ClientID)
	verifier = oauth2.GenerateVerifier()

	state, err := generateState()
	if err != nil {
		return err
	}

	cookieState := http.Cookie{
		Name:    "oauthstate",
		Value:   state,
		Expires: time.Now().Add(30 * time.Second),
	}

	http.SetCookie(w, &cookieState)
	url := oauthConfig.AuthCodeURL(state, oauth2.S256ChallengeOption(verifier))

	http.Redirect(w, r, url, http.StatusTemporaryRedirect)

	return nil
}

func Callback(w http.ResponseWriter, r *http.Request) ([]byte, error) {
	ctx := context.Background()

	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	if code == "" {
		return nil, fmt.Errorf("error: code is unavaliable")
	}

	oauthState, _ := r.Cookie("oauthstate")
	if state != oauthState.Value {
		return nil, fmt.Errorf("error: state does not match")
	}

	tok, err := oauthConfig.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		return nil, err
	}

	client := oauthConfig.Client(ctx, tok)

	resp, err := client.Get("https://api.github.com/user")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func generateState() (string, error) {
	state := make([]byte, 24)
	fmt.Println("GEN")
	if _, err := io.ReadFull(rand.Reader, state); err != nil {
		return "", err
	}
	fmt.Println("COMPLETE")

	return base64.StdEncoding.EncodeToString(state), nil
}
