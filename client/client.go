package client

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"github.com/crsrusl/bandcamp-downloader-v2/entities"
	"github.com/crsrusl/bandcamp-downloader-v2/file"
	"github.com/crsrusl/bandcamp-downloader-v2/helpers"
)

type Client interface {
	Download(url string) []error
	loadPage(url string) error
	downloadImage(filepath, url string) error
	downloadMP3(mp3 *entities.TrackData) error
}

func NewClient(file file.File) Client {
	return &bandcampClient{
		client:    &http.Client{},
		wg:        &sync.WaitGroup{},
		downloads: &[]string{},
		file:      file,
	}
}

type bandcampClient struct {
	client    *http.Client
	file      file.File
	downloads *[]string
	doc       *goquery.Document
	wg        *sync.WaitGroup
}

func (c *bandcampClient) Download(url string) []error {
	var errors []error

	var errorChan = make(chan error, 500)

	loadPageError := c.loadPage(url)
	if loadPageError != nil {
		errors = append(errors, loadPageError)
	}

	c.doc.Find("script[data-tralbum]").Each(func(i int, s *goquery.Selection) {
		var trackData = &entities.TrackData{}

		ticker := helpers.DownloadStatus(c.downloads)

		trackDataString, _ := s.Attr("data-tralbum")

		log.Printf("track data received:%s\n", trackDataString)

		if jsonUnmarshalError := json.Unmarshal([]byte(trackDataString), &trackData); jsonUnmarshalError != nil {
			log.Fatal(jsonUnmarshalError)
		}

		baseFilepath := fmt.Sprintf("./%s%s", helpers.RemoveAlphaNum(trackData.Artist), helpers.RemoveAlphaNum(trackData.Current.Title))
		albumArtwork := fmt.Sprintf("https://f4.bcbits.com/img/a%d_16.jpg", trackData.Current.ArtID)
		trackData.AlbumArtworkFilepath = fmt.Sprintf("%s/%s.jpg", baseFilepath, trackData.Current.Title)

		createError := c.file.CreateDir(baseFilepath)
		if createError != nil {
			errors = append(errors, createError)
		}

		downloadImageError := c.downloadImage(trackData.AlbumArtworkFilepath, albumArtwork)
		if downloadImageError != nil {
			errors = append(errors, downloadImageError)
		}

		for _, v := range trackData.TrackInfo {
			c.wg.Add(1)
			currentTrackData := *trackData

			currentTrackData.CurrentTrackNum = strconv.Itoa(v.TrackNum)
			currentTrackData.CurrentTrackTitle = v.Title
			currentTrackData.CurrentTrackURL = v.File.Mp3128
			currentTrackData.CurrentTrackFilepath = baseFilepath +
				"/" + helpers.RemoveAlphaNum(currentTrackData.CurrentTrackNum) +
				"-" + helpers.RemoveAlphaNum(currentTrackData.Artist) +
				"-" + helpers.RemoveAlphaNum(currentTrackData.CurrentTrackTitle) +
				".mp3"
			go func(mp3 entities.TrackData) {
				defer c.wg.Done()
				
				downloadError := c.downloadMP3(&mp3)
				if downloadError != nil {
					errorChan <- downloadError
				}
			}(currentTrackData)
		}

		c.wg.Wait()

		ticker.Stop()

		close(errorChan)

	})

	for err := range errorChan {
		errors = append(errors, err)
	}

	return errors
}

func (c *bandcampClient) loadPage(url string) error {
	log.Println("Getting... ", url)

	req, httpRequestError := http.NewRequestWithContext(context.TODO(), "GET", url, nil)
	if httpRequestError != nil {
		return httpRequestError
	}

	res, httpGetError := c.client.Do(req)
	if httpGetError != nil {
		return httpGetError
	}

	defer func() {
		err := res.Body.Close()
		if err != nil {
			log.Fatal(err)
		}
	}()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("status code error: %d %s", res.StatusCode, res.Status)
	}

	doc, documentReaderError := goquery.NewDocumentFromReader(res.Body)

	if documentReaderError != nil {
		return documentReaderError
	}

	c.doc = doc

	return nil
}

func (c *bandcampClient) downloadMP3(mp3 *entities.TrackData) error {

	*c.downloads = append(*c.downloads, fmt.Sprintf("%s - %s", mp3.Artist, mp3.CurrentTrackTitle))

	req, httpRequestError := http.NewRequestWithContext(context.TODO(), "GET", mp3.CurrentTrackURL, nil)

	if httpRequestError != nil {
		return httpRequestError
	}

	resp, httpGetError := c.client.Do(req)
	if httpGetError != nil {
		return httpGetError
	}

	defer func() {
		err := resp.Body.Close()
		if err != nil {
			log.Fatal(err)
		}
	}()

	saveError := c.file.Save(mp3.CurrentTrackFilepath, resp.Body)
	if saveError != nil {
		return saveError
	}

	tagFileError := c.file.TagFile(mp3)

	if tagFileError != nil {
		return tagFileError
	}

	return nil
}

func (c *bandcampClient) downloadImage(filepath string, url string) error {
	resp, httpGetError := http.Get(url)
	if httpGetError != nil {
		return httpGetError
	}

	defer func() {
		err := resp.Body.Close()
		if err != nil {
			log.Fatal(err)
		}
	}()

	saveError := c.file.Save(filepath, resp.Body)
	if saveError != nil {
		return saveError
	}

	return nil
}
