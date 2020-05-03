package main

import (
	"database/sql"
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
	_ "github.com/go-sql-driver/mysql"
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
	Id    string
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

type RenderedImg struct {
	Output string
	Delay  int
}

type config struct {
	Host   string
	Port   int
	ApiKey string
	Limit  int
	DbUser string
	DbPass string
	DbHost string
	DbPort int
	DbName string
}

var c config

var db *sql.DB

func initDb() (*sql.DB) {
	db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8", c.DbUser, c.DbPass, c.DbHost, c.DbPort, c.DbName))
	if err != nil {
		log.Fatalf("sql.Open error: %s", err)
	}
	// Open doesn't open a connection. Validate DSN data:
	if err = db.Ping(); err != nil {
		log.Fatalf("db.Ping error: %s", err)
	}
	return db
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

func alreadyExist(id string) (exist bool) {
	db := initDb()
	err := db.QueryRow("SELECT IF(COUNT(*),'true','false') FROM gif WHERE tenor_id=?", id).Scan(&exist)
	if err != nil {
		log.Println("can't check if already exist: ", err)
		return false
	}
	return exist
}

func insertGif(id string, url string) (imgs []RenderedImg) {
	db := initDb()
	res, err := http.Get(url)
	// TODO: err
	defer res.Body.Close()
	g, err := gif.DecodeAll(res.Body)
	// TODO: err

	stmt, err := db.Prepare("INSERT INTO gif (tenor_id) VALUES (?)")
	if err != nil {
		log.Println(err)
	}
	_, err = stmt.Exec(id)
	checkErr(err)

	imgs = renderGif(g)
	for i, srcImg := range imgs {
		stmt, err := db.Prepare("INSERT INTO gif_data (frame_nb, delay, frame, gif_id) VALUES (?, ?, ?, ?)")
		if err != nil {
			log.Println(err)
		}
		_, err = stmt.Exec(i, srcImg.Delay, srcImg.Output, id)
		checkErr(err)
	}
	return imgs
}

func getGifFromDb(id string) (imgs []RenderedImg) {
	db := initDb()

	rows, err := db.Query("SELECT delay, frame FROM gif_data WHERE gif_id =? ORDER BY frame_nb", id)
	if err != nil {
		log.Println(err)
	}
	for rows.Next() {
		var img RenderedImg
		err = rows.Scan(&img.Delay, &img.Output)
		imgs = append(imgs, img)
	}
	return imgs
}

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
func searchGif(search string) (gifUrl string, gifId string, err error) {
	url := fmt.Sprintf("https://api.tenor.com/v1/search?q='%s'&key=%s&media_filter=minimal&limit=%d", search, c.ApiKey, c.Limit)
	// Search gif on tenor
	res, err := http.Get(url)
	if err != nil {
		log.Printf("Error while searching gif %s", err)
		return "","", err
	}
	defer res.Body.Close()
	// Parse json response
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Printf("Error while reading body %s", err)
		return "","", err
	}
	var data Tenor
	if err := json.Unmarshal(body, &data); err != nil {
		log.Printf("Error while unmarshalling body %s", err)
		return "","", err
	}
	rand.Seed(time.Now().Unix())
	randNb := rand.Intn(c.Limit)
	gifUrl = data.Results[randNb].Media[0].Gif.Url
	return gifUrl, data.Results[randNb].Id, nil
}

// If anything bad happen, be cute
func oopsGif() ([]RenderedImg) {
	oopsFile, err := os.Open("static/img/oops.gif")
	defer oopsFile.Close()
	if err != nil {
		// Something is really wrong, just stop everything
		panic("can't even open rescue gif")
	}
	g, err := gif.DecodeAll(oopsFile)
	if err != nil {
		// Something is really wrong, just stop everything
		panic("can't even decode rescue gif")
	}
	return renderGif(g)
}

// Split gif, transform to ansi code and return a slice of images:delay
func renderGif(g *gif.GIF) (imgs []RenderedImg) {
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

	for i, srcImg := range g.Image {
		delay := g.Delay[i]
		draw.Draw(overpaintImage, overpaintImage.Bounds(), srcImg, image.ZP, draw.Over)
		pix, _ := ansimage.NewScaledFromImage(overpaintImage, sfy*ty, sfx*tx, mc, sm, dm)
		pix.SetMaxProcs(runtime.NumCPU())
		renderedGif := pix.Render()
		imgs = append(imgs, RenderedImg{Delay: delay, Output: renderedGif})
	}
	return imgs
}

// Send rendered gif as a response
func sendGif(w http.ResponseWriter, imgs []RenderedImg) {
	// Clear terminal and position cursor
	fmt.Fprintf(w, "\033[2J\033[1;1H")

	for _, srcImg := range imgs {
		time.Sleep(time.Duration(srcImg.Delay*10) * time.Millisecond)
		fmt.Fprintf(w, srcImg.Output)
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
	gifUrl, gifId, err := searchGif(search)
	if err != nil {
		sendGif(w, oopsGif())
	} else {
		if alreadyExist(gifId) {
			fmt.Println("already exists")
			sendGif(w, getGifFromDb(gifId))
		} else {
			sendGif(w, insertGif(gifId, gifUrl))
		}
	}
}

// Serve Vue.js files
func indexHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "static/index.html")
}

// Check if request is made from a CLI
func isValidBrowser(browser string) bool {
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
	if !isValidBrowser(name) {
		indexHandler(w, r)
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
	r.PathPrefix("/").HandlerFunc(indexHandler)

	err := http.ListenAndServe(fmt.Sprintf("%s:%d", c.Host, c.Port), handlers.RecoveryHandler()(r))
	if err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}
