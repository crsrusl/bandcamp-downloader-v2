package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/bogem/id3v2"
	"github.com/crsrusl/bandcamp-downloader-v2/structs"
	"github.com/inancgumus/screen"
)

func main() {
	const ArgsRequired = 2
	if len(os.Args) < ArgsRequired {
		log.Printf("usage: %s url\n", os.Args[0])
		os.Exit(1)
	}

	argsURL := os.Args[1]
	getArtistPage(argsURL)
}

func getArtistPage(url string) {
	log.Println("Getting... ", url)

	client := &http.Client{}

	req, httpRequestError := http.NewRequestWithContext(context.TODO(), "GET", url, nil)
	if httpRequestError != nil {
		log.Printf("%s", httpRequestError)

		return
	}

	res, httpGetError := client.Do(req)
	if httpGetError != nil {
		log.Printf("%s", httpGetError)

		return
	}

	defer func() {
		err := res.Body.Close()
		if err != nil {
			log.Fatal(err)
		}
	}()

	if res.StatusCode != http.StatusOK {
		log.Printf("status code error: %d %s", res.StatusCode, res.Status)

		return
	}

	doc, documentReaderError := goquery.NewDocumentFromReader(res.Body)
	if documentReaderError != nil {
		log.Printf("%s", documentReaderError)

		return
	}

	getTrackList(doc)
}

func getTrackList(doc *goquery.Document) {
	doc.Find("script[data-tralbum]").Each(func(i int, s *goquery.Selection) {
		var trackData structs.TrackData
		var wg sync.WaitGroup
		var downloads []string
		ticker := downloadStatus(&downloads)
		trackDataString, _ := s.Attr("data-tralbum")

		if jsonUnmarshalError := json.Unmarshal([]byte(trackDataString), &trackData); jsonUnmarshalError != nil {
			log.Fatal(jsonUnmarshalError)
		}

		trackData.BaseFilepath = "./" + removeAlphaNum(trackData.Artist) + removeAlphaNum(trackData.Current.Title)
		trackData.AlbumArtwork = "https://f4.bcbits.com/img/a" + strconv.Itoa(trackData.Current.ArtID) + "_16.jpg"
		trackData.AlbumArtworkFilepath = trackData.BaseFilepath + "/" + removeAlphaNum(trackData.Current.Title) + ".jpg"

		if osMkdirError := os.Mkdir(trackData.BaseFilepath, 0o700); osMkdirError != nil {
			log.Fatal(osMkdirError)
		}

		downloadImageError := downloadImage(trackData.AlbumArtworkFilepath, trackData.AlbumArtwork)
		if downloadImageError != nil {
			log.Fatal(downloadImageError)
		}

		for _, v := range trackData.Trackinfo {
			wg.Add(1)
			trackData.CurrentTrackTitle = v.Title
			trackData.CurrentTrackURL = v.File.Mp3128
			trackData.CurrentTrackFilepath = trackData.BaseFilepath +
				"/" + removeAlphaNum(trackData.Artist) +
				"-" + removeAlphaNum(trackData.CurrentTrackTitle) +
				".mp3"
			go downloadMp3(trackData, &wg, &downloads)
		}

		wg.Wait()

		if osRemoveError := os.Remove(trackData.AlbumArtworkFilepath); osRemoveError != nil {
			log.Fatal(osRemoveError)
		}

		ticker.Stop()

		log.Println("\r...Done\r")
	})
}

func downloadMp3(mp3 structs.TrackData, wg *sync.WaitGroup, downloads *[]string) {
	defer wg.Done()

	*downloads = append(*downloads, fmt.Sprintf("%s - %s", mp3.Artist, mp3.CurrentTrackTitle))

	client := &http.Client{}
	req, httpRequestError := http.NewRequestWithContext(context.TODO(), "GET", mp3.CurrentTrackURL, nil)

	if httpRequestError != nil {
		log.Printf("%s", httpRequestError)

		return
	}

	resp, httpGetError := client.Do(req)
	if httpGetError != nil {
		log.Printf("%s", httpGetError)

		return
	}

	defer func() {
		err := resp.Body.Close()
		if err != nil {
			log.Fatal(err)
		}
	}()

	out, osCreateError := os.Create(mp3.CurrentTrackFilepath)
	if osCreateError != nil {
		log.Printf("%s", osCreateError)
	}

	defer func() {
		err := out.Close()
		if err != nil {
			log.Fatal(err)
		}
	}()

	_, ioCopyError := io.Copy(out, resp.Body)
	if ioCopyError != nil {
		log.Printf("%s", ioCopyError)
	}

	if tagFileError := tagFile(mp3); tagFileError != nil {
		log.Printf("%s", tagFileError)
	}
}

func tagFile(mp3 structs.TrackData) error {
	tag, mp3OpenError := id3v2.Open(mp3.CurrentTrackFilepath, id3v2.Options{Parse: true, ParseFrames: nil})
	if mp3OpenError != nil {
		return fmt.Errorf("%w", mp3OpenError)
	}

	artwork, readFileError := ioutil.ReadFile(mp3.AlbumArtworkFilepath)
	if readFileError != nil {
		return fmt.Errorf("%w", readFileError)
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

	if saveTagError := tag.Save(); saveTagError != nil {
		return fmt.Errorf("%w", saveTagError)
	}

	if tagCloseError := tag.Close(); tagCloseError != nil {
		return fmt.Errorf("%w", tagCloseError)
	}

	return nil
}

func downloadImage(filepath string, url string) error {
	resp, httpGetError := http.Get(url)
	if httpGetError != nil {
		return fmt.Errorf("%w", httpGetError)
	}

	defer func() {
		err := resp.Body.Close()
		if err != nil {
			log.Fatal(err)
		}
	}()

	out, osCreateError := os.Create(filepath)
	if osCreateError != nil {
		return fmt.Errorf("%w", osCreateError)
	}

	defer func() {
		err := out.Close()
		if err != nil {
			log.Fatal(err)
		}
	}()

	_, ioCopyError := io.Copy(out, resp.Body)
	if ioCopyError != nil {
		return fmt.Errorf("%w", ioCopyError)
	}

	return nil
}

func removeAlphaNum(text string) string {
	reg := regexp.MustCompile("[^a-zA-Z0-9]+")

	processedString := reg.ReplaceAllString(text, "")

	return processedString
}

func downloadStatus(downloads *[]string) *time.Ticker {
	const refreshTime = 200

	rot := [4]string{"|", "/", "—", "\\"}
	rotations := len(rot) - 1
	ticker := time.NewTicker(refreshTime * time.Millisecond)
	pos := 1

	go func() {
		for range ticker.C {
			if pos > rotations {
				pos = 0
			}

			screen.Clear()
			screen.MoveTopLeft()

			for _, v := range *downloads {
				log.Println(rot[pos], " ", v)
			}
			pos++
		}
	}()

	return ticker
}
