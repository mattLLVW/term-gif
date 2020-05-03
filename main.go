package main

import (
	"encoding/json"
	"fmt"
	"github.com/mattLLVW/e.xec.sh/models"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"

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


// Send rendered gif as a response
func sendGif(w http.ResponseWriter, imgs []models.RenderedImg) {
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
		sendGif(w, models.OopsGif())
	} else {
		if models.AlreadyExist(gifId) {
			fmt.Println("already exists")
			sendGif(w, models.GetGifFromDb(gifId))
		} else {
			sendGif(w, models.InsertGif(gifId, gifUrl))
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
	models.InitDB(fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8", c.DbUser, c.DbPass, c.DbHost, c.DbPort, c.DbName))

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
