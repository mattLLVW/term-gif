package models

// Api json models
type Api struct {
	WebUrl  string
	Results []Result
}

type Result struct {
	Tag   []string
	Url   string
	Id    string
	Media []MediaType
}

type MediaType struct {
	Gif GifMedia
}

type GifMedia struct {
	Url     string
	Dims    []int
	Preview string
	Size    int
}
