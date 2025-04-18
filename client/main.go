package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/ItzAfroBoy/go-spotti"
	sk "github.com/ItzAfroBoy/swissknife"
)

func printData(data gospotti.PlaybackState) {
	var artists string
	for i, artist := range data.Track.Artists {
		artists += artist.Name
		if i != len(data.Track.Artists)-1 {
			artists += ", "
		}
	}
	fmt.Printf("Now playing: %s - %s\n", artists, data.Track.Name)
	fmt.Printf("Progress: %s/%s\n", (time.Duration(data.Progress) * time.Millisecond).Round(time.Second).String(), (time.Duration(data.Track.Duration) * time.Millisecond).Round(time.Second).String())
}

func clearScreen() {
	fmt.Print("\033[H\033[2J")
}

func main() {
	// reauth := flag.Bool("reauth", false, "Reauthorize the client")
	flag.Parse()

	spotti := gospotti.Client{}
	spotti.Auth.RedirectURI = "http://localhost:7171/callback"
	spotti.ClientID = "f1b6295487874fafb175fb5818c5abcf"
	spotti.Authorize(false)
	printData(spotti.Playback.GetPlaybackInfo())
	for {
		input := sk.Prompt("Spotti")
		switch input {
		case "next":
			spotti.Playback.NextTrack()
		case "prev":
			spotti.Playback.PreviousTrack()
		case "pause":
			spotti.Playback.Pause()
		case "play":
			spotti.Playback.Play()
		case "info":
			printData(spotti.Playback.GetPlaybackInfo())
		case "clear":
			clearScreen()
		case "exit":
			return
		case "help":
			fmt.Println("Commands:")
			fmt.Println("next - Skip to the next track")
			fmt.Println("prev - Go back to the previous track")
			fmt.Println("pause - Pause the current track")
			fmt.Println("play - Resume the current track")
			fmt.Println("info - Display information about the current track")
			fmt.Println("exit - Exit the program")
		default:
			fmt.Println("Invalid command. Type 'help' for a list of commands.")
		}
	}
}
