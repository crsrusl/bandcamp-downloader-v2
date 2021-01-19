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

		mp3 := structs.Mp3struct{
			Artist:     trackDataJson.Artist,
			AlbumTitle: trackDataJson.Current.Title,
			ArtID:      trackDataJson.Current.ArtID,
			AlbumArtwork: fmt.Sprint("https://f4.bcbits.com/img/a", trackDataJson.Current.ArtID, "_16.jpg"),
			BaseFilepath: fmt.Sprint("./", removeAlphaNum(trackDataJson.Artist), "-", removeAlphaNum(trackDataJson.Current.Title)),
		}

		err := os.Mkdir(mp3.BaseFilepath, 0700)
		if err != nil {
			log.Fatal(err)
		}

		mp3.Image, err = downloadImage(mp3.BaseFilepath+"/"+removeAlphaNum(mp3.AlbumTitle)+".jpg", mp3.AlbumArtwork)
		if err != nil {
			log.Fatal(err)
		}

		var wg sync.WaitGroup

		for _, v := range trackDataJson.Trackinfo {
			wg.Add(1)
			mp3.Title = v.Title
			filePath := mp3.BaseFilepath + "/" + removeAlphaNum(mp3.Artist) + "-" + removeAlphaNum(mp3.Title) + ".mp3"
			go downloadMp3(filePath, v.File.Mp3128, mp3, &wg)
		}

		wg.Wait()

		err = os.Remove(mp3.Image)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("...Done")
	})
}

func downloadMp3(filepath string, url string, mp3 structs.Mp3struct, wg *sync.WaitGroup) {
	defer wg.Done()

	fmt.Println("Downloading...", mp3.Artist, " - ", mp3.Title)

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
		log.Fatal(err)
	}

	artwork, err := ioutil.ReadFile(mp3.Image)
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
	tag.SetArtist(mp3.Artist)
	tag.SetTitle(mp3.Title)
	tag.SetAlbum(mp3.AlbumTitle)

	if err = tag.Save(); err != nil {
		log.Fatal(err)
	}

	if err = tag.Close(); err != nil {
		log.Fatal(err)
	}
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
