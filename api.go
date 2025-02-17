package gospotti

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Playback struct {
	client *Client
}

type PlaybackState struct {
	Progress int `json:"progress_ms"`
	Track    struct {
		Artists []struct {
			Name string
		} `json:"artists"`
		Name     string `json:"name"`
		Duration int    `json:"duration_ms"`
	} `json:"item"`
}

func (Playback) handleErrors(res *http.Response, msg []byte) {
	if res.StatusCode == 429 {
		fmt.Println("Rate limit exceeded.")
	} else {
		var data map[string]interface{}
		checkError(json.Unmarshal(msg, &data))
		fmt.Println("Error:")
		fmt.Println(res.Status)
		fmt.Println(data["error"].(map[string]interface{})["message"])
	}
}

func (p Playback) GetPlaybackInfo() (data PlaybackState) {
	httpClient := &http.Client{}
	req, err := http.NewRequest("GET", "https://api.spotify.com/v1/me/player", nil)
	checkError(err)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", p.client.Token))
	res, err := httpClient.Do(req)
	checkError(err)
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode == 200 {
		checkError(json.Unmarshal(raw, &data))
	} else if res.StatusCode == 204 {
		fmt.Println("No track is currently playing.")
	} else if res.StatusCode == 401 {
		p.client.Reauthorize()
		p.GetPlaybackInfo()
	} else {
		p.handleErrors(res, raw)
	}
	return
}

func (p Playback) PreviousTrack() {
	httpClient := &http.Client{}
	req, err := http.NewRequest("POST", "https://api.spotify.com/v1/me/player/previous", nil)
	checkError(err)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", p.client.Token))
	res, err := httpClient.Do(req)
	checkError(err)
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode == 200 {
		fmt.Println("Playing previous track")
	} else if res.StatusCode == 401 {
		p.client.Reauthorize()
		p.PreviousTrack()
	} else {
		p.handleErrors(res, raw)
	}
}

func (p Playback) NextTrack() {
	httpClient := &http.Client{}
	req, err := http.NewRequest("POST", "https://api.spotify.com/v1/me/player/next", nil)
	checkError(err)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", p.client.Token))
	res, err := httpClient.Do(req)
	checkError(err)
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode == 200 {
		fmt.Println("Playing next track")
	} else if res.StatusCode == 401 {
		p.client.Reauthorize()
		p.NextTrack()
	} else {
		p.handleErrors(res, raw)
	}
}

func (p Playback) Pause() {
	httpClient := &http.Client{}
	req, err := http.NewRequest("PUT", "https://api.spotify.com/v1/me/player/pause", nil)
	checkError(err)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", p.client.Token))
	res, err := httpClient.Do(req)
	checkError(err)
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode == 200 {
		fmt.Println("Pausing playback")
	} else if res.StatusCode == 401 {
		p.client.Reauthorize()
		p.Pause()
	} else {
		p.handleErrors(res, raw)
	}
}

func (p Playback) Play() {
	httpClient := &http.Client{}
	req, err := http.NewRequest("PUT", "https://api.spotify.com/v1/me/player/play", nil)
	checkError(err)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", p.client.Token))
	res, err := httpClient.Do(req)
	checkError(err)
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode == 200 {
		fmt.Println("Starting playback")
	} else if res.StatusCode == 401 {
		p.client.Reauthorize()
		p.Play()
	} else {
		p.handleErrors(res, raw)
	}
}
