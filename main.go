package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image/color"
	"image/gif"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/eliukblau/pixterm/pkg/ansimage"
	"github.com/mattLLVW/term-gif/models"
	"golang.org/x/time/rate"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	lol "github.com/kris-nova/lolgopher"
	"github.com/mssola/user_agent"
	"github.com/spf13/viper"
)

type config struct {
	Host      string
	Port      int
	ApiKey    string
	ApiUrl    string
	Limit     int
	DbUser    string
	DbPass    string
	DbHost    string
	DbPort    int
	DbName    string
	DbEnabled bool
}

var c config

// Create a custom visitor struct which holds the rate limiter for each
// visitor and the last time that the visitor was seen.
// alexedwards.net/blog/how-to-rate-limit-http-requests
type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// Change the map to hold values of the type visitor.
var visitors = make(map[string]*visitor)
var mu sync.Mutex

func getVisitor(ip string) *rate.Limiter {
	mu.Lock()
	defer mu.Unlock()

	v, exists := visitors[ip]
	if !exists {
		limiter := rate.NewLimiter(1, 3)
		// Include the current time when creating a new visitor.
		visitors[ip] = &visitor{limiter, time.Now()}
		return limiter
	}

	// Update the last seen time for the visitor.
	v.lastSeen = time.Now()
	return v.limiter
}

// Every minute check the map for visitors that haven't been seen for
// more than 3 minutes and delete the entries.
func cleanupVisitors() {
	for {
		time.Sleep(time.Minute)

		mu.Lock()
		for ip, v := range visitors {
			if time.Since(v.lastSeen) > 3*time.Minute {
				delete(visitors, ip)
			}
		}
		mu.Unlock()
	}
}

// Search gif on api and return download url and gif id
func searchApi(search string) (data models.Api, err error) {
	url := fmt.Sprintf("%s/v1/search?q='%s'&key=%s&media_filter=minimal&limit=%d", c.ApiUrl, search, c.ApiKey, c.Limit)
	// Search gif on api
	res, err := http.Get(url)
	if err != nil {
		log.Printf("Error while searching gif %s", err)
		return data, err
	}
	defer res.Body.Close()
	// Parse json response
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Printf("Error while reading body %s", err)
		return data, err
	}
	if err := json.Unmarshal(body, &data); err != nil {
		log.Printf("Error while unmarshalling body %s", err)
		return data, err
	}
	return data, nil
}

// If anything bad happen, be cute
func oopsGif() []models.RenderedImg {
	g := models.AnsiGif{}
	g.Oops()
	g.Render()
	return g.Rendered
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
	// Clear terminal
	fmt.Fprintf(w, "\033[2J")
}

// Check if url is valid
func isUrl(str string) bool {
	u, err := url.Parse(str)
	return err == nil && u.Scheme != "" && u.Host != ""
}

// TODO: find a way to list natively supported extension.
func isValidExtension(ext string) bool {
	switch ext {
	case ".jpeg", ".jpg", ".png", ".gif", ".webp":
		return true
	}
	return false
}

// If requested from Cli, search for gif, encode in ansi and return result
func wildcardHandler(w http.ResponseWriter, r *http.Request) {
	// Extract search terms and url query from url and format them
	qry := r.URL.Query()
	preview := qry.Get("img")
	url := qry.Get("url")
	reverse := qry.Get("rev")
	vars := mux.Vars(r)
	search, ok := vars["search"]

	// If there's no search terms and nothing to fetch, just return doc.
	if !ok && url == "" {
		landing := `                                  
       _  __
  __ _(_)/ _|__  ___   _ _________   _   _ __ _   _ _ __
 / _  | | |_ \ \/ / | | |_  /_  / | | | | '__| | | | '_ \
| (_| | |  _| >  <| |_| |/ / / /| |_| |_| |  | |_| | | | |
 \__, |_|_|(_)_/\_\\__, /___/___|\__, (_)_|   \__,_|_| |_|
 |___/             |___/         |___/

Search and display gifs in your terminal

	USAGE:
		curl gif.xyzzy.run/<your_search_terms_separated_by_an_underscore>
		curl gif.xyzzy.run?url=https://<the_greatest_gif_or_image_of_all_times.jpeg>
	
	EXAMPLE:
		curl gif.xyzzy.run/spongebob_magic
		curl "gif.xyzzy.run?url=https://gif.xyzzy.run/static/img/mgc.gif" #inception :)


	You can also reverse the gif if you want, i.e:

		curl "gif.xyzzy.run/mind_blown?rev=true"
	
	Or just display a preview image of the gif, i.e:

		curl "gif.xyzzy.run/wow?img=true"


Powered By Tenor

if you like this project, please consider sponsoring it: https://github.com/sponsors/mattLLVW

`
		lw := lol.Writer{
			Output:    w,
			ColorMode: lol.ColorModeTrueColor,
		}
		lw.Write([]byte(landing))
		return
	}
	// If user submit url, check that it's valid with a valid extension before downloading.
	if url != "" {
		ext := filepath.Ext(url)
		if !isUrl(url) || !isValidExtension(ext) {
			sendGif(w, oopsGif())
			return
		}
		res, err := http.Get(url)
		if err != nil {
			sendGif(w, oopsGif())
			return
		}
		defer res.Body.Close()
		body, err := ioutil.ReadAll(res.Body)

		// Use the net/http package's handy DetectContentType function.
		contentType := http.DetectContentType(body)

		if contentType == "image/gif" {
			g := models.AnsiGif{}
			g.Gif, err = gif.DecodeAll(bytes.NewBuffer(body))
			if err != nil {
				sendGif(w, oopsGif())
				return
			}
			if preview != "" {
				// Clear terminal and position cursor
				fmt.Fprintf(w, "\033[2J\033[1;1H")
				img := g.Preview()
				fmt.Fprintf(w, img)
				return
			}
			g.Render()
			if reverse != "" {
				g.Reverse()
			}
			sendGif(w, g.Rendered)
			return
		}

		if strings.HasPrefix(contentType, "image") {
			// Clear terminal and position cursor
			fmt.Fprintf(w, "\033[2J\033[1;1H")
			// set image scale factor for ANSIPixel grid, background color and scale mode
			tx, ty := 30, 9
			sfy, sfx := ansimage.BlockSizeY, ansimage.BlockSizeX
			mc := color.RGBA{0x00, 0x00, 0x00, 0xff}
			dm := ansimage.DitheringMode(0)
			sm := ansimage.ScaleMode(2)
			pix, _ := ansimage.NewScaledFromReader(bytes.NewBuffer(body), sfy*ty, sfx*tx, mc, sm, dm)
			pix.SetMaxProcs(runtime.NumCPU())
			res := pix.Render()
			fmt.Fprintf(w, res)
			return
		}
		// If the detected content type is not a valid image be cute.
		sendGif(w, oopsGif())
		return
	}

	// Search gif on api and return api data
	strings.ReplaceAll(search, "_", " ")
	apiData, err := searchApi(search)

	if err != nil || len(apiData.Results) == 0 {
		log.Println("api results len", len(apiData.Results), "err", err)
		sendGif(w, oopsGif())
		return
	}
	// If enabled, return a random gif
	rand.Seed(time.Now().Unix())
	randNb := rand.Intn(c.Limit)
	if len(apiData.Results) < c.Limit {
		// Must be a specific search so just return the only result
		randNb = 0
	}
	gifUrl := apiData.Results[randNb].Media[0].Gif.Url
	gifId := apiData.Results[randNb].Id
	gifPreview := apiData.Results[randNb].Media[0].Gif.Preview
	if preview != "" {
		// Clear terminal and position cursor
		fmt.Fprintf(w, "\033[2J\033[1;1H")
		img := models.GetPreview(gifPreview)
		fmt.Fprintf(w, img)
		return
	}
	if c.DbEnabled && models.AlreadyExist(gifId) {
		// If we already have gif rendered locally try returning it
		log.Println("fetching", gifId, "from database")
		g, err := models.GetGifFromDb(gifId, qry.Get("rev") != "")
		if err != nil {
			log.Println("error while fetching", gifId, "from database")
			sendGif(w, oopsGif())
			return
		}
		sendGif(w, g)
		return
	}
	// Render fetched gif
	g := models.AnsiGif{}
	err = g.Get(gifUrl)
	if err != nil {
		log.Println("error getting", gifId)
		g.Oops()
		g.Render()
		sendGif(w, g.Rendered)
		return
	}
	g.Render()
	if c.DbEnabled {
		// Store rendered gif in db.
		log.Println("inserting", gifId, "into database")
		err = g.Insert(gifId)
		if err != nil {
			log.Println("error while inserting", gifId, "into database")
			g.Oops()
			g.Render()
			sendGif(w, g.Rendered)
			return
		}
	}
	if reverse != "" {
		g.Reverse()
	}
	sendGif(w, g.Rendered)
	return
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
	// If static site, just return it
	if !isValidBrowser(name) {
		http.ServeFile(w, r, "static/index.html")
	} else {
		// Apply limiter if we are curling and have a real ip
		ip := r.Header.Get("X-Real-Ip")
		if ip != "" {
			limiter := getVisitor(ip)
			if limiter.Allow() == false {
				http.Error(w, http.StatusText(429), http.StatusTooManyRequests)
				return
			}
		}
		wildcardHandler(w, r)
	}
}

// Run a background goroutine to remove old entries from the visitors map.
func init() {
	go cleanupVisitors()
}

func main() {
	log.SetOutput(os.Stdout)

	// Find and read the config file
	viper.SetConfigFile(".env")
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error while reading config file %s", err)
	}
	if err := viper.Unmarshal(&c); err != nil {
		log.Fatalf("Unable to unmarshal config %s", err)
	}

	if c.DbEnabled {
		// Init database using environ variables
		models.InitDB(fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8", c.DbUser, c.DbPass, c.DbHost, c.DbPort, c.DbName))
	}

	// Route to static site or ascii gif based on user agent
	r := mux.NewRouter()
	r.PathPrefix("/static").Handler(http.StripPrefix("/static", http.FileServer(http.Dir("./static")))).Methods("GET")
	r.HandleFunc("/{search}", conditionalHandler).Methods("GET")
	r.PathPrefix("/").HandlerFunc(conditionalHandler).Methods("GET")

	log.Printf("starting server on %s:%d\n", c.Host, c.Port)
	err := http.ListenAndServe(fmt.Sprintf("%s:%d", c.Host, c.Port), handlers.RecoveryHandler()(handlers.LoggingHandler(os.Stdout, r)))
	if err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}
