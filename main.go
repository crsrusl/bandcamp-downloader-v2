package main

import (
	"encoding/json"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/bogem/id3v2"
	"github.com/crsrusl/bandcamp-downloader-v2/structs"
	"github.com/inancgumus/screen"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"sync"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: %s url\n", os.Args[0])
		os.Exit(1)
	}

	argsURL := os.Args[1]
	getArtistPage(argsURL)
}

func getTrackList(doc *goquery.Document) {
	doc.Find("script[data-tralbum]").Each(func(i int, s *goquery.Selection) {
		var trackData structs.TrackData
		var wg sync.WaitGroup
		var downloads []string
		var ticker = downloadStatus(&downloads)
		var trackDataString, _ = s.Attr("data-tralbum")

		if jsonUnmarshalError := json.Unmarshal([]byte(trackDataString), &trackData); jsonUnmarshalError != nil {
			log.Fatal(jsonUnmarshalError)
		}

		trackData.BaseFilepath = "./" + removeAlphaNum(trackData.Artist) + removeAlphaNum(trackData.Current.Title)
		trackData.AlbumArtwork = "https://f4.bcbits.com/img/a" + strconv.Itoa(trackData.Current.ArtID) + "_16.jpg"
		trackData.AlbumArtworkFilepath = trackData.BaseFilepath + "/" + removeAlphaNum(trackData.Current.Title) + ".jpg"

		if osMkdirError := os.Mkdir(trackData.BaseFilepath, 0700); osMkdirError != nil {
			log.Fatal(osMkdirError)
		}

		if downloadImageError := downloadImage(trackData.AlbumArtworkFilepath, trackData.AlbumArtwork); downloadImageError != nil {
			log.Fatal(downloadImageError)
		}

		for _, v := range trackData.Trackinfo {
			wg.Add(1)
			trackData.CurrentTrackTitle = v.Title
			trackData.CurrentTrackURL = v.File.Mp3128
			trackData.CurrentTrackFilepath = trackData.BaseFilepath + "/" + removeAlphaNum(trackData.Artist) + "-" + removeAlphaNum(trackData.CurrentTrackTitle) + ".mp3"
			go downloadMp3(trackData, &wg, &downloads)
		}

		wg.Wait()

		if osRemoveError := os.Remove(trackData.AlbumArtworkFilepath); osRemoveError != nil {
			log.Fatal(osRemoveError)
		}

		ticker.Stop()

		fmt.Println("\r...Done")
	})
}

func getArtistPage(url string) {
	fmt.Println("Getting... ", url)

	res, httpGetError := http.Get(url)
	if httpGetError != nil {
		log.Fatal(httpGetError)
	}

	defer res.Body.Close()

	if res.StatusCode != 200 {
		log.Fatalf("status code error: %d %s", res.StatusCode, res.Status)
	}

	doc, documentReaderError := goquery.NewDocumentFromReader(res.Body)
	if documentReaderError != nil {
		log.Fatal(documentReaderError)
	}

	getTrackList(doc)
}

func downloadMp3(mp3 structs.TrackData, wg *sync.WaitGroup, downloads *[]string) {
	defer wg.Done()

	*downloads = append(*downloads, fmt.Sprintf("%s - %s", mp3.Artist, mp3.CurrentTrackTitle))

	resp, err := http.Get(mp3.CurrentTrackURL)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	out, err := os.Create(mp3.CurrentTrackFilepath)
	if err != nil {
		log.Fatal(err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	if tagFileError := tagFile(mp3); tagFileError != nil {
		log.Fatal(tagFileError)
	}
}

func tagFile(mp3 structs.TrackData) error {
	tag, err := id3v2.Open(mp3.CurrentTrackFilepath, id3v2.Options{Parse: true})
	if err != nil {
		return err
	}

	artwork, err := ioutil.ReadFile(mp3.AlbumArtworkFilepath)
	if err != nil {
		return err
	}

	pic := id3v2.PictureFrame{
		Encoding:    id3v2.EncodingUTF8,
		MimeType:    "image/jpeg",
		PictureType: id3v2.PTFrontCover,
		Description: "Front cover",
		Picture:     artwork,
	}

	tag.AddAttachedPicture(pic)
	tag.SetArtist(mp3.Artist)
	tag.SetTitle(mp3.CurrentTrackTitle)
	tag.SetAlbum(mp3.Current.Title)

	if err = tag.Save(); err != nil {
		return err
	}

	if err = tag.Close(); err != nil {
		return err
	}

	return nil
}

func downloadImage(filepath string, url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

func removeAlphaNum(text string) string {
	reg, err := regexp.Compile("[^a-zA-Z0-9]+")
	if err != nil {
		log.Fatal(err)
	}
	processedString := reg.ReplaceAllString(text, "")

	return processedString
}

func downloadStatus(downloads *[]string) *time.Ticker {
	rot := [4]string{"|", "/", "—", "\\"}
	ticker := time.NewTicker(200 * time.Millisecond)

	pos := 1

	go func() {
		for range ticker.C {
			if pos > 3 {
				pos = 0
			}

			screen.Clear()
			screen.MoveTopLeft()

			for _, v := range *downloads {
				fmt.Println(rot[pos], " ", v)
			}
			pos = pos + 1
		}
	}()

	return ticker
}
