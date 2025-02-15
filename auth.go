package gospotti

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/99designs/keyring"
	"github.com/skratchdot/open-golang/open"
)

type clientAuth struct {
	state         string
	redirectURI   string
	codeVerifier  string
	codeChallenge string
	authCode      string
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
}

type Client struct {
	Playback     Playback
	Auth         clientAuth
	ClientID     string
	Token        string
	RefreshToken string
}

func checkError(err error) {
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

func getRandomValues(arr []*big.Int, limit int) []*big.Int {
	var err error
	for i := 0; i < len(arr); i++ {
		arr[i], err = rand.Int(rand.Reader, big.NewInt(int64(limit)))
		checkError(err)
	}
	return arr
}

func reduce(chars string, arr []*big.Int) string {
	var result string
	for i := 0; i < len(arr); i++ {
		result += string(chars[arr[i].Int64()])
	}
	return result
}

func generateRandomString(length int) string {
	chars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	vals := make([]*big.Int, length)
	getRandomValues(vals, len(chars))
	return reduce(chars, vals)
}

func (c *Client) loadTokens() {
	k, err := keyring.Open(keyring.Config{
		ServiceName: "GoSpotti",
	})
	if err != nil {
		return
	}
	item, err := k.Get("token")
	if err != nil {
		return
	}
	c.Token = string(item.Data)
	item, err = k.Get("refreshToken")
	if err != nil {
		return
	}
	c.RefreshToken = string(item.Data)
	c.Playback.client = c
}

func (c *Client) saveTokens() {
	k, err := keyring.Open(keyring.Config{
		ServiceName: "GoSpotti",
	})
	checkError(err)
	err = k.Set(keyring.Item{
		Key:  "token",
		Data: []byte(c.Token),
	})
	checkError(err)
	err = k.Set(keyring.Item{
		Key:  "refreshToken",
		Data: []byte(c.RefreshToken),
	})
	checkError(err)
	c.Playback.client = c
}

func (c *Client) listenForAuthCode() {
	ln, err := net.Listen("tcp", ":7171")
	checkError(err)
	defer ln.Close()
	conn, _ := ln.Accept()

	var buf [1024]byte
	conn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
	conn.Read(buf[:])
	conn.Close()
	raw, _, _ := strings.Cut(string(buf[:]), " HTTP/1.1\r\n")
	url, _ := url.Parse(raw)
	if url.Query().Get("error") == "access_denied" {
		checkError(fmt.Errorf("access denied authorization"))
	}
	c.Auth.authCode = url.Query().Get("code")
}

func (c *Client) getAuthToken() {
	fmt.Println("Getting auth token...")
	res, err := http.Post("https://accounts.spotify.com/api/token", "application/x-www-form-urlencoded", strings.NewReader(fmt.Sprintf("client_id=%s&grant_type=authorization_code&code=%s&redirect_uri=%s&code_verifier=%s", c.clientID, c.auth.authCode, c.auth.redirectURI, c.auth.codeVerifier)))
	checkError(err)
	defer res.Body.Close()
	raw, err := io.ReadAll(res.Body)
	checkError(err)
	var data tokenResponse
	checkError(json.Unmarshal(raw, &data))
	c.Token = data.AccessToken
	c.RefreshToken = data.RefreshToken
	c.saveTokens()
}

func (c *Client) Authorize(reauth bool) {
	c.loadTokens()
	if c.Token == "" || reauth {
		fmt.Println("Authorizing...")
		c.Auth.state = generateRandomString(11)
		c.Auth.codeVerifier = generateRandomString(64)

		sha := sha256.New()
		io.WriteString(sha, c.Auth.codeVerifier)
		c.Auth.codeChallenge = base64.RawURLEncoding.EncodeToString(sha.Sum(nil))

		authURL := fmt.Sprintf("https://accounts.spotify.com/authorize?client_id=%s&response_type=code&redirect_uri=%s&code_challenge=%s&code_challenge_method=S256&scope=user-read-playback-state user-modify-playback-state&state=%s", c.clientID, c.auth.redirectURI, c.auth.codeChallenge, c.auth.state)
		open.Run(authURL)
		c.listenForAuthCode()
		c.getAuthToken()
	}
}

func (c *Client) Reauthorize() {
	fmt.Println("Refreshing token...")
	res, err := http.Post("https://accounts.spotify.com/api/token", "application/x-www-form-urlencoded", strings.NewReader(fmt.Sprintf("client_id=%s&grant_type=refresh_token&refresh_token=%s", c.clientID, c.refreshToken)))
	checkError(err)
	defer res.Body.Close()
	raw, err := io.ReadAll(res.Body)
	checkError(err)
	var data tokenResponse
	checkError(json.Unmarshal(raw, &data))
	c.Token = data.AccessToken
	c.RefreshToken = data.RefreshToken
}
