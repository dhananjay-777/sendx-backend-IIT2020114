package main
import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/html"
)

type ResponseData struct {
	URL        string `json:"url"`
	UserStatus string `json:"userStatus"`
	Html       string `json:"html"`
}

type CrawledPage struct {
	HTML       string
	LastCrawled time.Time
}

var (
	crawledPagesMutex sync.RWMutex
	crawledPages      map[string]CrawledPage
	paidQueue         []Request
	nonPaidQueue      []Request
	priorityQueueMutex sync.Mutex
)

type Request struct {
	URL        string
	UserStatus string
}

func main() {
	crawledPages = make(map[string]CrawledPage)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			urlInput := r.URL.Query().Get("urlInput")
			userStatus := r.URL.Query().Get("userStatus")
			
			if urlInput != "" && userStatus != "" {
				// Check if the URL has been crawled in the last 60 minutes
				crawledPagesMutex.RLock()
				cachedPage, exists := crawledPages[urlInput]
				crawledPagesMutex.RUnlock()

				if exists && time.Since(cachedPage.LastCrawled).Minutes() <= 60 {
					// Serve the cached HTML page
					data := ResponseData{
						URL:        urlInput,
						UserStatus: userStatus,
						Html:       cachedPage.HTML,
					}
					fmt.Println("Already had this in cache")
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(data)
					return
				}

				request := Request{
					URL:        urlInput,
					UserStatus: userStatus,
				}
				

				// Add the request to the appropriate queue
				priorityQueueMutex.Lock()
				if userStatus == "paid" {
					paidQueue = append(paidQueue, request)
				} else {
					nonPaidQueue = append(nonPaidQueue, request)
				}
				priorityQueueMutex.Unlock()

				// Process the queue
				go processRequestQueue(w)
			}
		}
		// Handle other cases, such as serving the HTML page
		http.ServeFile(w, r, "./index.html")
	})

	fmt.Println("Server is running on :8080")
	http.ListenAndServe(":8080", nil)
}

func processRequestQueue(w http.ResponseWriter) {
	// Process paid requests first
	priorityQueueMutex.Lock()
	if len(paidQueue) > 0 {
		request := paidQueue[0]
		paidQueue = paidQueue[1:]
		priorityQueueMutex.Unlock()
		processRequest(w ,request)
	} else {
		priorityQueueMutex.Unlock()
		// If no paid requests, process non-paid requests
		if len(nonPaidQueue) > 0 {
			request := nonPaidQueue[0]
			nonPaidQueue = nonPaidQueue[1:]
			processRequest(w,request)
		}
	}
}

func processRequest(w http.ResponseWriter, request Request) {
	urlInput := request.URL
	userStatus := request.UserStatus



	// fmt.Println("hello",request)

	// Check if the URL has been crawled in the last 60 minutes
	crawledPagesMutex.RLock()
	cachedPage, exists := crawledPages[urlInput]
	crawledPagesMutex.RUnlock()

	if exists && time.Since(cachedPage.LastCrawled).Minutes() <= 60 {
		// Serve the cached HTML page directly
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cachedPage.HTML)
		return
	}

	// URL not in cache or expired, scrape and cache the URL
	htmlContent, err := scrapeURLWithRetries(urlInput, 3) // 3 retries
	if err != nil {
		fmt.Printf("Failed to scrape URL: %v for %v user. Error: %v\n", urlInput, userStatus, err)
		return
	}

	// Format the HTML content for readability
	formattedHTML, err := formatHTML(htmlContent)
	
	if err != nil {
		fmt.Printf("Failed to format HTML for URL: %v for %v user. Error: %v\n", urlInput, userStatus, err)
		return
	}
	

	// Cache the scraped HTML page
	crawledPagesMutex.Lock()
	crawledPages[urlInput] = CrawledPage{
		HTML:       formattedHTML,
		LastCrawled: time.Now(),
	}
	crawledPagesMutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cachedPage.HTML)
	
	fmt.Printf("Scraped and cached URL: %v for %v user.\n", urlInput, userStatus)
}



func scrapeURLWithRetries(url string, maxRetries int) (string, error) {
	var htmlContent string
	var err error

	for i := 0; i < maxRetries; i++ {
		htmlContent, err = scrapeURL(url)
		if err == nil {
			return htmlContent, nil
		}
		fmt.Printf("Failed to scrape URL: %v. Retrying...\n", url)
		time.Sleep(2 * time.Second) // Delay between retries
	}

	return "", err
}

func scrapeURL(url string) (string, error) {
	// Load the document from the URL
	doc, err := goquery.NewDocument(url)
	if err != nil {
		return "", err
	}

	// Select the entire HTML document
	htmlContent, err := doc.Html()
	if err != nil {
		return "", err
	}
	return htmlContent, nil
}

func formatHTML(htmlContent string) (string, error) {
	// Format the HTML content for readability
	minifier := minify.New()
	minifier.AddFunc("text/html", html.Minify)
	formattedHTML, err := minifier.String("text/html", htmlContent)
	if err != nil {
		return "", err
	}
	return formattedHTML, nil
}