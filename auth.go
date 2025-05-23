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
	"strings"

	"github.com/99designs/keyring"
	"github.com/skratchdot/open-golang/open"
)

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
}

type ClientAuth struct {
	state         string
	RedirectURI   string
	codeVerifier  string
	codeChallenge string
	authCode      string
}

type Client struct {
	Playback     Playback
	Auth         ClientAuth
	ClientID     string
	Token        string
	RefreshToken string
	keychain     keyring.Keyring
}

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}

func getRandomValues(arr []*big.Int, limit int) []*big.Int {
	var err error
	for i := range arr {
		arr[i], err = rand.Int(rand.Reader, big.NewInt(int64(limit)))
		checkError(err)
	}
	return arr
}

func reduce(chars string, arr []*big.Int) string {
	var result string
	for i := range arr {
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
	item, err := c.keychain.Get("token")
	if err != nil {
		if err == keyring.ErrKeyNotFound {
			c.Token = ""
		} else {
			checkError(err)
		}
	} else {
		c.Token = string(item.Data)
	}

	item, err = c.keychain.Get("refreshToken")
	if err != nil {
		if err == keyring.ErrKeyNotFound {
			c.RefreshToken = ""
		} else {
			checkError(err)
		}
	} else {
		c.RefreshToken = string(item.Data)
	}

	c.Playback.client = c
}

func (c *Client) saveTokens() {
	err := c.keychain.Set(keyring.Item{
		Key:  "token",
		Data: []byte(c.Token),
	})
	checkError(err)

	err = c.keychain.Set(keyring.Item{
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
	conn.Read(buf[:])
	conn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
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
	res, err := http.Post("https://accounts.spotify.com/api/token", "application/x-www-form-urlencoded", strings.NewReader(fmt.Sprintf("client_id=%s&grant_type=authorization_code&code=%s&redirect_uri=%s&code_verifier=%s", c.ClientID, c.Auth.authCode, c.Auth.RedirectURI, c.Auth.codeVerifier)))
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

func Init() *Client {
	k, err := keyring.Open(keyring.Config{
		ServiceName:              "GoSpotti",
		KeychainName:             "GoSpotti",
		KeychainTrustApplication: true,
	})
	checkError(err)
	c := &Client{}
	c.keychain = k
	return c
}

func (c *Client) Authorize(reauth bool) {
	c.loadTokens()
	if c.Token == "" || c.RefreshToken == "" || reauth {
		fmt.Println("Authorizing...")
		c.Auth.state = generateRandomString(11)
		c.Auth.codeVerifier = generateRandomString(64)

		sha := sha256.New()
		io.WriteString(sha, c.Auth.codeVerifier)
		c.Auth.codeChallenge = base64.RawURLEncoding.EncodeToString(sha.Sum(nil))

		authURL := fmt.Sprintf("https://accounts.spotify.com/authorize?client_id=%s&response_type=code&redirect_uri=%s&code_challenge=%s&code_challenge_method=S256&scope=user-read-playback-state user-modify-playback-state&state=%s", c.ClientID, c.Auth.RedirectURI, c.Auth.codeChallenge, c.Auth.state)
		open.Run(authURL)
		c.listenForAuthCode()
		c.getAuthToken()
	}
}

func (c *Client) Reauthorize() {
	fmt.Println("Refreshing token...")
	res, err := http.Post("https://accounts.spotify.com/api/token", "application/x-www-form-urlencoded", strings.NewReader(fmt.Sprintf("client_id=%s&grant_type=refresh_token&refresh_token=%s", c.ClientID, c.RefreshToken)))
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
