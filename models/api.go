package models

// Api json models
type Api struct {
	Results []struct {
		ID                 string  `json:"id"`
		Title              string  `json:"title"`
		URL                string  `json:"url"`
		Itemurl            string  `json:"itemurl"`
		Created            float64 `json:"created"`
		ContentDescription string  `json:"content_description"`
		Flags              string  `json:"flags"`
		Hasaudio           bool    `json:"hasaudio"`
		Hascaption         bool    `json:"hascaption"`
		MediaFormats       struct {
			Preview struct {
				URL      string  `json:"url"`
				Dims     []int   `json:"dims"`
				Duration float64 `json:"duration"`
				Size     int     `json:"size"`
			} `json:"preview"`
			Gif struct {
				URL      string  `json:"url"`
				Dims     []int   `json:"dims"`
				Duration float64 `json:"duration"`
				Size     int     `json:"size"`
			} `json:"gif"`
		} `json:"media_formats"`
		AnalyticsResponsePayload string `json:"analytics_response_payload"`
		Analytics                struct {
			Onload struct {
				URL string `json:"url"`
			} `json:"onload"`
			Onclick struct {
				URL string `json:"url"`
			} `json:"onclick"`
			Onsent struct {
				URL string `json:"url"`
			} `json:"onsent"`
		} `json:"analytics"`
	} `json:"results"`
	Next string `json:"next"`
}