package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
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
	Intro struct {
		Start int `json:"start"`
		End   int `json:"end"`
	} `json:"intro"`
}

func main() {
	var name string
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Search Anime: ")
	name, _ = reader.ReadString('\n')
	name = strings.TrimSpace(name)

	anify := fmt.Sprintf("https://api.anify.tv/search/anime/%s", strings.Replace(name, " ", "%20", 1))
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

	sources := listEpisodes(id)

	var gogoAnime int
	for i := range sources {
		if sources[i].ProviderID == "gogoanime" {
			gogoAnime = i
			break
		}
	}

	var url string
	for i := range sources[gogoAnime].Episodes {
		if sources[gogoAnime].Episodes[i].Number == epNumber {
			url = watch(sources[gogoAnime].ProviderID, sources[gogoAnime].Episodes[i].ID, episode, id)
			fmt.Println(url)
			break
		}
	}

	cmdStruct := exec.Command("pwsh.exe", "/c", "mpv", url)
	out, err := cmdStruct.Output()
	if err != nil {
		panic(err)
	}

	fmt.Println(out)

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

func watch(provider, watchid, episodeNumber, id string) string {
	res, err := http.Get("https://api.anify.tv/sources?providerId=" + provider + "&watchId=" + watchid + "&episodeNumber=" + episodeNumber + "&id=" + id + "&subType=sub")
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

	var bestQuality string

	for i := range link.Sources {
		if link.Sources[i].Quality == "1080p" {
			bestQuality = link.Sources[i].Url
			break
		}
	}
	return bestQuality
}
