package Structs;

type Song struct {

	YouTubeID string `json:"youtube_id"`

	Title string `json:"title"`
	Artists []string `json:"artists"`
	Album string `json:"album"`

	Duration Duration `json:"duration"`

	Cover string `json:"cover"`

}

type Duration struct {

	Seconds int `json:"seconds"`
	Formatted string `json:"formatted"`

}