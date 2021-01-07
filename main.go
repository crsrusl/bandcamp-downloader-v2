package main

import (
	"encoding/json"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/bogem/id3v2"
	"github.com/crsrusl/bandcamp-downloader-v2/structs"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
)

func main() {
	argsURL := os.Args[1]

	fmt.Print("Getting... ", argsURL, "\n")
	getArtistPage(argsURL)
}

func getArtistPage(url string) {
	res, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.Fatalf("status code error: %d %s", res.StatusCode, res.Status)
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	doc.Find("script[data-tralbum]").Each(func(i int, s *goquery.Selection) {
		trackDataString, _ := s.Attr("data-tralbum")

		var trackDataJson structs.TrackData
		err = json.Unmarshal([]byte(trackDataString), &trackDataJson)
		if err != nil {
			log.Fatal(err)
		}


		artist := trackDataJson.Artist
		albumTitle := trackDataJson.Current.Title
		artID := trackDataJson.Current.ArtID
		albumArtwork := fmt.Sprint("https://f4.bcbits.com/img/a",artID,"_16.jpg")
		baseFilepath := fmt.Sprint("./",removeAlphaNum(artist),"-",removeAlphaNum(albumTitle))

		err := os.Mkdir(baseFilepath, 0700)
		if err != nil {
			panic(err)
		}

		image, err := downloadFile(baseFilepath+"/"+removeAlphaNum(albumTitle)+".jpg", albumArtwork)
		if err != nil {
			log.Fatal(err)
		}

		for _, v := range trackDataJson.Trackinfo {
			title := v.Title
			filePath := baseFilepath + "/" + removeAlphaNum(artist) + "-" + removeAlphaNum(title) + ".mp3"
			url := v.File.Mp3128
			mp3, err := downloadFile(filePath, url)

			if err != nil {
				panic(err)
			}

			tag, err := id3v2.Open(mp3, id3v2.Options{Parse: true})
			if err != nil {
				log.Fatal("Error while opening mp3 file: ", err)
			}

			artwork, err := ioutil.ReadFile(image)
			if err != nil {
				log.Fatal(err)
			}

			pic := id3v2.PictureFrame{
				Encoding:    id3v2.EncodingUTF8,
				MimeType:    "image/jpeg",
				PictureType: id3v2.PTFrontCover,
				Description: "Front cover",
				Picture:     artwork,
			}

			tag.AddAttachedPicture(pic)
			tag.SetArtist(artist)
			tag.SetTitle(title)
			tag.SetAlbum(albumTitle)

			if err = tag.Save(); err != nil {
				log.Fatal("Error while saving a tag: ", err)
			}

			tag.Close()
		}

		err2 := os.Remove(image)
		if err2 != nil {
			log.Fatal(err2)
		}
	})
}

func downloadFile(filepath string, url string) (string, error) {
	fmt.Println("Downloading...", url)
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	out, err := os.Create(filepath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", err
	}
	return filepath, nil
}

func removeAlphaNum(text string) string {
	reg, err := regexp.Compile("[^a-zA-Z0-9]+")
	if err != nil {
		log.Fatal(err)
	}
	processedString := reg.ReplaceAllString(text, "")

	return processedString
}