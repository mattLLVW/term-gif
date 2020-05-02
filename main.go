package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/eliukblau/pixterm/pkg/ansimage"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/mssola/user_agent"
	"github.com/spf13/viper"
)

type Tenor struct {
	WebUrl  string
	Results []Result
}

type Result struct {
	Tag   []string
	Url   string
	Media []MediaType
}

type MediaType struct {
	Gif Gif
}

type Gif struct {
	Url     string
	Dims    []int
	Preview string
	Size    int
}

type config struct {
	Host   string
	Port   int
	apiKey string
	Limit  int
}

var c config

// Get max Gif dimensions.
func getGifDimensions(gif *gif.GIF) (x, y int) {
	var lowestX int
	var lowestY int
	var highestX int
	var highestY int

	for _, img := range gif.Image {
		if img.Rect.Min.X < lowestX {
			lowestX = img.Rect.Min.X
		}
		if img.Rect.Min.Y < lowestY {
			lowestY = img.Rect.Min.Y
		}
		if img.Rect.Max.X > highestX {
			highestX = img.Rect.Max.X
		}
		if img.Rect.Max.Y > highestY {
			highestY = img.Rect.Max.Y
		}
	}

	return highestX - lowestX, highestY - lowestY
}

// Search gif on tenor and return download url
func searchGif(search string) (gifUrl string, err error) {
	url := fmt.Sprintf("https://api.tenor.com/v1/search?q='%s'&key=%s&media_filter=minimal&limit=%d", search, c.apiKey, c.Limit)
	// Search gif on tenor
	res, err := http.Get(url)
	if err != nil {
		log.Printf("Error while searching gif %s", err)
		return "", err
	}
	defer res.Body.Close()
	// Parse json response
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Printf("Error while reading body %s", err)
		return "", err
	}
	var data Tenor
	if err := json.Unmarshal(body, &data); err != nil {
		log.Printf("Error while unmarshalling body %s", err)
		return "", err
	}
	rand.Seed(time.Now().Unix())
	gifUrl = data.Results[rand.Intn(c.Limit)].Media[0].Gif.Url
	return gifUrl, nil
}

// If anything bad happen, be cute
func oopsGif() (g *gif.GIF) {
	oopsFile, err := os.Open("static/img/oops.gif")
	defer oopsFile.Close()
	if err != nil {
		// Something is really wrong, just stop everything
		panic("can't even open rescue gif")
	}
	g, err = gif.DecodeAll(oopsFile)
	if err != nil {
		// Something is really wrong, just stop everything
		panic("can't even decode rescue gif")
	}
	return
}

// Split gif images and write response according to timing
func sendGif(w http.ResponseWriter, g *gif.GIF) {
	// https://stackoverflow.com/a/33296596/8135079
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Error while searching gif %s", r)
		}
	}()

	imgWidth, imgHeight := getGifDimensions(g)

	overpaintImage := image.NewRGBA(image.Rect(0, 0, imgWidth, imgHeight))
	draw.Draw(overpaintImage, overpaintImage.Bounds(), g.Image[0], image.ZP, draw.Src)

	// set image scale factor for ANSIPixel grid
	tx, ty := 30, 9
	sfy, sfx := ansimage.BlockSizeY, ansimage.BlockSizeX
	mc := color.RGBA{0x00, 0x00, 0x00, 0xff}
	dm := ansimage.DitheringMode(0)
	sm := ansimage.ScaleMode(2)
	// Clear terminal and position cursor
	fmt.Fprintf(w, "\033[2J\033[1;1H")

	for i, srcImg := range g.Image {
		delay := g.Delay[i]
		draw.Draw(overpaintImage, overpaintImage.Bounds(), srcImg, image.ZP, draw.Over)
		pix, _ := ansimage.NewScaledFromImage(overpaintImage, sfy*ty, sfx*tx, mc, sm, dm)
		pix.SetMaxProcs(runtime.NumCPU())
		renderedGif := pix.Render()
		time.Sleep(time.Duration(delay*10) * time.Millisecond)
		fmt.Fprintf(w, renderedGif)
		// Reposition cursor
		fmt.Fprintf(w, "\033[1;1H")
	}
}

// If requested from Cli, search for gif, encode in ansi and return result
func wildcardHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	search := vars["search"]
	strings.ReplaceAll(search, "_", " ")

	// Search gif on tenor and return download url
	gifUrl, err := searchGif(search)
	if err != nil {
		sendGif(w, oopsGif())
	} else {
		res, err := http.Get(gifUrl)
		defer res.Body.Close()
		if err != nil {
			sendGif(w, oopsGif())
		} else {
			g, err := gif.DecodeAll(res.Body)
			if err != nil {
				sendGif(w, oopsGif())
			} else {
				sendGif(w, g)
			}
		}
	}
}

// Serve Vue.js files
func IndexHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "static/index.html")
}

// Check if request is made from a CLI
func IsValidBrowser(browser string) bool {
	switch browser {
	case
		"curl",
		"Wget",
		"HTTPie":
		return true
	}
	return false
}

// Return proper handler depending on cli or browser
func conditionalHandler(w http.ResponseWriter, r *http.Request) {
	ua := user_agent.New(r.Header.Get("User-Agent"))
	name, _ := ua.Browser()
	if !IsValidBrowser(name) {
		IndexHandler(w, r)
	} else {
		wildcardHandler(w, r)
	}
}

func main() {
	log.Println("starting server...")
	// Find and read the config file
	viper.SetConfigFile(".env")
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error while reading config file %s", err)
	}
	if err := viper.Unmarshal(&c); err != nil {
		log.Fatalf("Unable to unmarshal config %s", err)
	}

	r := mux.NewRouter()

	r.PathPrefix("/static").Handler(http.StripPrefix("/static", http.FileServer(http.Dir("./static"))))
	r.PathPrefix("/favicon.ico").Handler(http.FileServer(http.Dir("static/")))
	r.HandleFunc("/{search}", conditionalHandler)
	r.PathPrefix("/").HandlerFunc(IndexHandler)

	err := http.ListenAndServe(fmt.Sprintf("%s:%d", c.Host, c.Port), handlers.RecoveryHandler()(r))
	if err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}
