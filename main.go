package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

type Anime struct {
	Results []struct {
		ID    string `json:"id"`
		Title struct {
			Romaji  string `json:"romaji"`
			English string `json:"english"`
		} `json:"title"`
		EpisodeNum int `json:"currentEpisode"`
	} `json:"results"`
}

type Watch struct {
	ProviderID string `json:"providerid"`
	Episodes   []struct {
		ID     string `json:"id"`
		Number int    `json:"number"`
		Title  string `json:"title"`
	} `json:"episodes"`
}

type WatchResponse []Watch

type Links struct {
	Sources []struct {
		Url     string `json:"url"`
		Quality string `json:"quality"`
	} `json:"sources"`
}

func (link *Links) qualityCheck(quality string) (string, string) {
	var def string
	var best string
	for i := range link.Sources {
		if link.Sources[i].Quality == quality {
			best = link.Sources[i].Url
		}
		if link.Sources[i].Quality == "default" {
			def = link.Sources[i].Url
		}
	}

	return best, def
}

func main() {
	var name string
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Search Anime: ")
	name, _ = reader.ReadString('\n')
	name = strings.TrimSpace(name)

	anify := fmt.Sprintf("https://api.anify.tv/search/anime/%s", strings.Replace(name, " ", "%20", -1))
	res, err := http.Get(anify)
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		panic("API not available")
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}

	var anime Anime
	err = json.Unmarshal(body, &anime)
	if err != nil {
		panic(err)
	}

	resultsNum := 1
	for i := range anime.Results {
		fmt.Printf("[%d] %s / %s\n", resultsNum, anime.Results[i].Title.English, anime.Results[i].Title.Romaji)
		resultsNum++
	}

	var index int
	fmt.Print("Enter your choice: ")
	fmt.Scan(&index)
	index = index - 1

	id := anime.Results[index].ID
	english := anime.Results[index].Title.English

	fmt.Println("\nID:", id)
	fmt.Println("Name:", english)
	fmt.Printf("Episodes: [%d]\n", anime.Results[index].EpisodeNum)

	// Episode code
	var episode string
	fmt.Print("Choose an episode: ")
	fmt.Scan(&episode)
	epNumber, err := strconv.Atoi(episode)
	if err != nil {
		panic(err)
	}

	mpv(id, episode, english, epNumber)
}

func mpv(id, episode, english_title string, epNumber int) {
	sources := listEpisodes(id)

	var gogoAnime int
	for i := range sources {
		if sources[i].ProviderID == "gogoanime" {
			gogoAnime = i
			break

		}
	}

	var bestUrl string
	var defUrl string
	var title string
	for i := range sources[gogoAnime].Episodes {
		if sources[gogoAnime].Episodes[i].Number == epNumber {
			bestUrl, defUrl = watch(sources[gogoAnime].ProviderID, sources[gogoAnime].Episodes[i].ID, episode, id)
			title = sources[gogoAnime].Episodes[i].Title
			break
		}
	}

	if bestUrl != "" {
		bestUrl = fmt.Sprintf("'%s'", bestUrl)
		if runtime.GOOS == "windows" {
			cmd := exec.Command("powershell.exe", "/c", "mpv", bestUrl)
			err := cmd.Run()
			if err != nil {
				panic(err)
			}
		} else if runtime.GOOS == "darwin" {
			cmd := exec.Command("powershell.exe", "/c", "iina", bestUrl)
			err := cmd.Run()
			if err != nil {
				panic(err)
			}
		}
	} else if defUrl != "" {
		defUrl = fmt.Sprintf("'%s'", defUrl)
		if runtime.GOOS == "windows" {
			cmd := exec.Command("powershell.exe", "/c", "mpv", defUrl)
			err := cmd.Run()
			if err != nil {
				panic(err)
			}
		} else if runtime.GOOS == "darwin" {
			cmd := exec.Command("/bin/sh", "/c", "iina", defUrl)
			err := cmd.Run()
			if err != nil {
				panic(err)
			}
		}
	} else {
		fmt.Println("Unable to find URL for episode :(")
		os.Exit(1)
	}

	var choice string
	fmt.Printf("\nPlaying: %s | %s\n", english_title, title)
	fmt.Printf("[n] Next Episode\n")
	fmt.Printf("[p] Previous Episode\n")
	fmt.Printf("[s] Select Episode\n")
	fmt.Printf("[q] Quit\n")
	fmt.Print("Choose: ")
	fmt.Scan(&choice)

	if choice == "n" {
		epNumber++
		episode = strconv.Itoa(epNumber)
		mpv(id, episode, english_title, epNumber)
	}
	if choice == "p" {
		epNumber--
		episode = strconv.Itoa(epNumber)
		mpv(id, episode, english_title, epNumber)
	}
	if choice == "s" {
		fmt.Print("\nChoose an episode")
		fmt.Scan(&episode)
		epNumber, err := strconv.Atoi(episode)
		if err != nil {
			panic(err)
		}
		mpv(id, episode, english_title, epNumber)
	}
	if choice == "q" {
		os.Exit(0)
	}
}

func listEpisodes(anime_id string) WatchResponse {
	url := fmt.Sprintf("https://api.anify.tv/episodes/%s", anime_id)
	res, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}

	var sources WatchResponse
	err = json.Unmarshal(body, &sources)
	if err != nil {
		panic(err)
	}

	return sources
}

func watch(provider, watchid, episodeNumber, id string) (string, string) {
	url := fmt.Sprintf("https://api.anify.tv/sources?providerId=%s&watchId=%s&episodeNumber=%s&id=%s&subType=sub", provider, watchid, episodeNumber, id)
	res, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}

	var link Links
	err = json.Unmarshal(body, &link)
	if err != nil {
		panic(err)
	}

	bestQuality, def := link.qualityCheck("1080p")

	return bestQuality, def

}
