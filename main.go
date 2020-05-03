package main

import (
	"encoding/json"
	"fmt"
	"github.com/mattLLVW/e.xec.sh/models"
	"golang.org/x/time/rate"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/mssola/user_agent"
	"github.com/spf13/viper"
)

type config struct {
	Host      string
	Port      int
	ApiKey    string
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
// taken from alexedwards.net/blog/how-to-rate-limit-http-requests
type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// Change the the map to hold values of the type visitor.
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

func limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.Header.Get("X-Real-Ip")
		if ip != "" {
			limiter := getVisitor(ip)
			if limiter.Allow() == false {
				http.Error(w, http.StatusText(429), http.StatusTooManyRequests)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// Search gif on api and return download url and gif id
func searchApi(search string) (data models.Api, err error) {
	url := fmt.Sprintf("https://api.tenor.com/v1/search?q='%s'&key=%s&media_filter=minimal&limit=%d", search, c.ApiKey, c.Limit)
	// Search gif on tenor
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

// If requested from Cli, search for gif, encode in ansi and return result
func wildcardHandler(w http.ResponseWriter, r *http.Request) {
	// Extract search terms from url and format them
	search := mux.Vars(r)["search"]
	strings.ReplaceAll(search, "_", " ")
	qry := r.URL.Query()

	// Search gif on api and return api data
	apiData, err := searchApi(search)

	if err != nil || len(apiData.Results) == 0 {
		log.Println("api results len", len(apiData.Results), "err", err)
		sendGif(w, models.OopsGif())
	} else {
		// If enabled, return a random gif
		rand.Seed(time.Now().Unix())
		randNb := rand.Intn(c.Limit)
		if len(apiData.Results) < c.Limit {
			// Must be a specific search so just return the only result
			randNb = 0
		}
		gifUrl := apiData.Results[randNb].Media[0].Gif.Url
		gifId := apiData.Results[randNb].Id
		if models.AlreadyExist(gifId) {
			// If we already have gif rendered locally try returning it
			log.Println("fetching", gifId, "from database")
			g, err := models.GetGifFromDb(gifId, qry.Get("rev") != "")
			if err != nil {
				log.Println("error while fetching", gifId, "from database")
				sendGif(w, models.OopsGif())
			} else {
				sendGif(w, g)
			}
		} else {
			// If we don't have gif locally, store it in database and return it
			log.Println("inserting", gifId, "into database")
			g, err := models.InsertGif(gifId, gifUrl, qry.Get("rev") != "")
			if err != nil {
				log.Println("error while inserting", gifId, "into database")
				sendGif(w, models.OopsGif())
			} else {
				sendGif(w, g)
			}
		}
	}
}

// Serve static files
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

// Run a background goroutine to remove old entries from the visitors map.
func init() {
	go cleanupVisitors()
}

func main() {
	// Open logging file and output logs to both std out and app.log
	f, err := os.OpenFile("app.log", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	mw := io.MultiWriter(os.Stdout, f)
	log.SetOutput(mw)

	// Find and read the config file
	viper.SetConfigFile(".env")
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error while reading config file %s", err)
	}
	if err := viper.Unmarshal(&c); err != nil {
		log.Fatalf("Unable to unmarshal config %s", err)
	}

	// Init database using environ variables
	models.InitDB(fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8", c.DbUser, c.DbPass, c.DbHost, c.DbPort, c.DbName))

	// Route to static site or ascii gif based on user agent
	r := mux.NewRouter()
	r.PathPrefix("/static").Handler(http.StripPrefix("/static", http.FileServer(http.Dir("./static")))).Methods("GET")
	r.HandleFunc("/{search}", conditionalHandler).Methods("GET")
	r.PathPrefix("/").HandlerFunc(indexHandler).Methods("GET")

	log.Println("starting server...")
	err = http.ListenAndServe(fmt.Sprintf("%s:%d", c.Host, c.Port), handlers.RecoveryHandler()(handlers.LoggingHandler(mw, limit(r))))
	if err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}
