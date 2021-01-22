package structs

type TrackData struct {
	Current struct {
		ReleaseDate string      `json:"release_date"`
		Artist      interface{} `json:"artist"`
		Title       string      `json:"title"`
		ID          int64       `json:"id"`
		ArtID       int         `json:"art_id"`
	} `json:"current"`
	Trackinfo []struct {
		File struct {
			Mp3128 string `json:"mp3-128"`
		} `json:"file"`
		TrackNum int    `json:"track_num"`
		TrackID  int    `json:"track_id"`
		Title    string `json:"title"`
	} `json:"trackinfo"`
	Artist               string `json:"artist"`
	AlbumReleaseDate     string `json:"album_release_date"`
	ArtID                int    `json:"art_id"`
	BaseFilepath         string
	AlbumArtwork         string
	AlbumArtworkFilepath string
	CurrentTrackTitle    string
	CurrentTrackURL      string
	CurrentTrackFilepath string
}
