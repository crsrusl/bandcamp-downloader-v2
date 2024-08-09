package file

import (
	"io"
	"log"
	"os"

	"github.com/bogem/id3v2"
	"github.com/crsrusl/bandcamp-downloader-v2/entities"
)

type File interface {
	Save(path string, content io.Reader) error
	CreateDir(path string) error
	TagFile(mp3 *entities.TrackData) error
	close()
}

func NewFileMngmnt() File {
	return &fileMngmnt{}
}

type fileMngmnt struct {
	file *os.File
}

func (f *fileMngmnt) Save(path string, content io.Reader) error {
	out, osCreateError := os.Create(path)
	if osCreateError != nil {
		return osCreateError
	}

	_, ioCopyError := io.Copy(out, content)
	if ioCopyError != nil {
		return ioCopyError
	}

	f.file = out

	defer f.close()

	return nil
}

func (f *fileMngmnt) CreateDir(path string) error {
	osMkdirError := os.MkdirAll(path, 0o700)

	if osMkdirError != nil {
		return osMkdirError
	}

	return nil
}

func (f *fileMngmnt) TagFile(mp3 *entities.TrackData) error {
	tag, mp3OpenError := id3v2.Open(mp3.CurrentTrackFilepath, id3v2.Options{Parse: true, ParseFrames: nil})
	if mp3OpenError != nil {
		return mp3OpenError
	}

	artwork, readFileError := os.ReadFile(mp3.AlbumArtworkFilepath)
	if readFileError != nil {
		return readFileError
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
		return saveTagError
	}

	if tagCloseError := tag.Close(); tagCloseError != nil {
		return tagCloseError
	}

	return nil
}

func (f *fileMngmnt) close() {
	if f.file != nil {
		err := f.file.Close()
		if err != nil {
			log.Fatalln(err.Error())
		}
		return
	}

	log.Fatalln("missing file")
}
