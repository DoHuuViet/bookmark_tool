package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/badoux/goscraper"
	"github.com/dustin/go-humanize"
	"github.com/julienschmidt/httprouter"
	"golang.org/x/net/html"
)

type Bookmarks struct {
	sync.RWMutex
	Sorted      []string
	BookmarkMap map[string]Bookmark
}

type Bookmark struct {
	Url      string
	Icon     string
	Category string
	Title    string
	Modified time.Time
}

var flagFile, flagPort, flagHost, flagSecret string

func (bookmarks *Bookmarks) load() {
	input, _ := ioutil.ReadFile(flagFile)
	json.Unmarshal(input, &bookmarks.BookmarkMap)
	bookmarks.sort()
}

func (bookmarks *Bookmarks) sort() {
	bookmarks.RLock()
	defer bookmarks.RUnlock()
	var keys []string
	for key, _ := range bookmarks.BookmarkMap {
		keys = append(keys, key)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(keys)))
	bookmarks.Sorted = keys
}

func (bookmarks *Bookmarks) save(url string) {
	url = parseUrl(url)
	if url == "" {
		return
	}

	if bookmarks.exists(url) {
		return
	}

	bookmarks.Lock()
	bookmarks.BookmarkMap[time.Now().String()] = Bookmark{Url: url, Icon: getIcon(url), Category: "default", Title: getTitle(url), Modified: time.Now()}
	bookmarks.Unlock()
	bookmarks.sort()
	bookmarks.saveToFile()
}

func getIcon(url string) string {
	iconUrl, err := goscraper.Scrape(url, 5)
	if err != nil {
		return ""
	}
	if strings.Contains(iconUrl.Preview.Icon, "http") {
		return iconUrl.Preview.Icon
	} else {
		return url + iconUrl.Preview.Icon
	}
}

func getTitle(url string) string {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Println(err)
		return url
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 6.1; rv:6.0) Gecko/20110814 Firefox/6.0")
	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return url
	}
	defer resp.Body.Close()
	d := html.NewTokenizer(resp.Body)
	for {
		tokenType := d.Next()
		if tokenType == html.ErrorToken {
			return ""
		}
		token := d.Token()
		if token.Data == "title" {
			d.Next()
			text := d.Text()
			if string(text) == "" {
				return url
			}
			return string(text)
		}
	}
}

func parseUrl(uri string) string {
	if strings.Contains(uri, ".") {
		if strings.Contains(uri, "favicon.ico") {
			return ""
		}
		if !strings.Contains(uri, "http") {
			return "http://" + uri
		}
		return uri
	}
	return ""
}

func (bookmarks *Bookmarks) exists(url string) bool {
	bookmarks.RLock()
	defer bookmarks.RUnlock()
	for _, bookmark := range bookmarks.BookmarkMap {
		if bookmark.Url == url {
			return true
		}
	}
	return false
}

func (bookmarks *Bookmarks) delete(key string) {
	bookmarks.Lock()
	delete(bookmarks.BookmarkMap, key)
	bookmarks.Unlock()
	bookmarks.sort()
	bookmarks.saveToFile()
}

func (bookmarks *Bookmarks) saveToFile() {
	bookmarks.RLock()
	defer bookmarks.RUnlock()
	data, err := json.Marshal(bookmarks.BookmarkMap)
	if err != nil {
		return
	}
	ioutil.WriteFile(flagFile, data, 0644)
}

func (bookmarks *Bookmarks) showBookMarks(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
	bookmarks.RLock()
	defer bookmarks.RUnlock()
	t, _ := template.New("").Funcs(template.FuncMap{"humanize": humanize.Time}).Parse(bookmarkTemplate)
	t.Execute(writer, bookmarks)
}

func checkAuth(request *http.Request) bool {
	cookie, err := request.Cookie("bookmark")
	if err != nil {
		return false
	}
	if cookie.Value != flagSecret {
		return false
	}

	return true
}

func main() {
	flag.StringVar(&flagPort, "port", "8080", "port to the webserver listens on")
	flag.StringVar(&flagFile, "file", "bookmark.json", "file to save bm")
	flag.StringVar(&flagHost, "host", "localhost", "hostname to listen on")
	flag.StringVar(&flagSecret, "secret", "secret", "secret cookie url to auth on")
	flag.Parse()

	bookmarks := Bookmarks{BookmarkMap: make(map[string]Bookmark)}
	bookmarks.load()

	router := httprouter.New()
	router.RedirectTrailingSlash = false
	router.RedirectFixedPath = false

	router.GET("/*url", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		url := params.ByName("url")[1:]
		if strings.HasPrefix(url, flagSecret) {
			http.SetCookie(writer, &http.Cookie{Name: "bookmark", Value: flagSecret, Expires: time.Now().Add(90000 * time.Hour),
				Domain: flagHost, Path: "/"})
			http.Redirect(writer, request, "bookmarks", http.StatusFound)
			return
		}

		if !checkAuth(request) {
			fmt.Fprintf(writer, "Access Denied!!!")
			return
		}
		if strings.HasPrefix(url, "remove/") {
			key := strings.Split(url, "/")
			bookmarks.delete(key[1])
			http.Redirect(writer, request, "/bookmarks", http.StatusFound)
			return
		}

		if strings.HasPrefix(url, "bookmarks") {
			bookmarks.showBookMarks(writer, request, params)
			return
		}
		bookmarks.save(url)
		http.Redirect(writer, request, "bookmarks", http.StatusFound)
	})
	fmt.Println("Starting server on: " + flagPort)
	log.Fatal(http.ListenAndServe(":"+flagPort, router))
}
