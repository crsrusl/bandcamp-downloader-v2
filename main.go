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
	"sync"
)

type mp3struct struct {
	artist       string
	title        string
	albumTitle   string
	artID        int
	image        string
	albumArtwork string
	baseFilepath string
}

func main() {
	argsURL := os.Args[1]
	getArtistPage(argsURL)
}

func getArtistPage(url string) {
	fmt.Println("Getting... ", url)

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

		mp3 := mp3struct{
			artist:     trackDataJson.Artist,
			albumTitle: trackDataJson.Current.Title,
			artID:      trackDataJson.Current.ArtID,
		}

		mp3.albumArtwork = fmt.Sprint("https://f4.bcbits.com/img/a", mp3.artID, "_16.jpg")
		mp3.baseFilepath = fmt.Sprint("./", removeAlphaNum(mp3.artist), "-", removeAlphaNum(mp3.albumTitle))

		err := os.Mkdir(mp3.baseFilepath, 0700)
		if err != nil {
			panic(err)
		}

		image, err := downloadImage(mp3.baseFilepath+"/"+removeAlphaNum(mp3.albumTitle)+".jpg", mp3.albumArtwork)
		if err != nil {
			log.Fatal(err)
		}

		mp3.image = image

		var wg sync.WaitGroup

		for _, v := range trackDataJson.Trackinfo {
			wg.Add(1)
			mp3.title = v.Title
			filePath := mp3.baseFilepath + "/" + removeAlphaNum(mp3.artist) + "-" + removeAlphaNum(mp3.title) + ".mp3"
			url := v.File.Mp3128
			go downloadMp3(filePath, url, mp3, &wg)
		}

		wg.Wait()

		err2 := os.Remove(mp3.image)
		if err2 != nil {
			log.Fatal(err2)
		}
	})
}

func downloadMp3(filepath string, url string, mp3 mp3struct, wg *sync.WaitGroup) {
	defer wg.Done()

	fmt.Println("Downloading...", mp3.artist, " - ", mp3.title)
	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	out, err := os.Create(filepath)
	if err != nil {
		log.Fatal(err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	tag, err := id3v2.Open(filepath, id3v2.Options{Parse: true})
	if err != nil {
		log.Fatal("Error while opening mp3 file: ", err)
	}

	artwork, err := ioutil.ReadFile(mp3.image)
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
	tag.SetArtist(mp3.artist)
	tag.SetTitle(mp3.title)
	tag.SetAlbum(mp3.albumTitle)

	if err = tag.Save(); err != nil {
		log.Fatal("Error while saving a tag: ", err)
	}

	tag.Close()
}

func downloadImage(filepath string, url string) (string, error) {
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
