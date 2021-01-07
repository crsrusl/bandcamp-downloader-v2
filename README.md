## Bandcamp downloader V2

The original Bandcamp downloader worked well, but I wanted something which would work without requiring the nodejs to be installed on the users' system.

Version 2 of the Bandcamp downloader is written entirely in Golang and can be compiled to run as a standalone on a range of different platforms, e.g. Windows, Mac, Linux.

To get started, clone this repo and run the application, being sure to include the URL you want to download the music from `go run main.go [url]`.

The application can download from most Bandcamp urls, including:
- Album Urls e.g. https://[artist].bandcamp.com/album/[album]
- Track Urls e.g. https://[artist].bandcamp.com/track/[name]

To have the application run as standalone command line tool, first build for your platform `go build -o ./bcd main.go`, and then place the application into your `$GOPATH/bin`. You will now be able to easily download tracks to your working directory by running `bcd [url]` from your command line.

Cheers