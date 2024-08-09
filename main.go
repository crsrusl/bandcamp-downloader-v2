package main

import (
	"log"
	"os"

	"github.com/crsrusl/bandcamp-downloader-v2/client"
	"github.com/crsrusl/bandcamp-downloader-v2/file"
)

func main() {
	URL := getURL()

	file := file.NewFileMngmnt()

	client := client.NewClient(file)

	errors := client.Download(URL)

	for _, err := range errors {
		log.Println(err)
		return
	}

	log.Println("Done")
}

func getURL() string {
	if len(os.Args) < 2 {
		log.Fatalf("usage: %s url\n", os.Args[0])
	}
	return os.Args[1]
}
