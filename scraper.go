package main

import (
	"bytes"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type Movie struct {
	gorm.Model
	Title       string
	ReleaseDate string
	Rating      int
}

func mustSetupDB() *gorm.DB {
	// Open the SQLite database, file-based
	db, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect database")
	}

	// Automatically create the schema using the Product model
	err = db.AutoMigrate(&Movie{})
	if err != nil {
		log.Fatal("[setupDB] failed to migrate movie")
	}
	return db
}

func parseAPIResponse(responseHTML string) ([]*Movie, error) {
	newReader := strings.NewReader(responseHTML)
	doc, err := goquery.NewDocumentFromReader(newReader)
	if err != nil {
		return nil, err
	}

	movies := make([]*Movie, 0)

	doc.Find("div.card div").Each(func(index int, item *goquery.Selection) {
		title := item.Find(".content h2 a").AttrOr("title", "No title found")
		if title == "No title found" {
			return
		}

		percentageStr := item.Find(".user_score_chart").AttrOr("data-percent", "-1")

		percentage, err := strconv.Atoi(percentageStr)
		if err != nil {
			log.Println("error parsing percentage")
			return
		}

		// Find the date
		date := item.Find(".content p").Text()

		movies = append(movies, &Movie{
			Model:       gorm.Model{},
			Title:       title,
			ReleaseDate: date,
			Rating:      percentage,
		})
	})

	return movies, nil
}

func fetchDataFromAPI(page int) (string, error) {
	urlStr := "https://www.themoviedb.org/discover/movie/items"

	form := url.Values{}
	form.Add("air_date.gte", "")
	form.Add("air_date.lte", "")
	form.Add("certification", "")
	form.Add("certification_country", "DE")
	form.Add("debug", "")
	form.Add("first_air_date.gte", "")
	form.Add("first_air_date.lte", "")
	form.Add("page", strconv.Itoa(page))
	form.Add("primary_release_date.gte", "")
	form.Add("primary_release_date.lte", "")
	form.Add("region", "")
	form.Add("release_date.gte", "")
	form.Add("release_date.lte", "2024-11-19")
	form.Add("show_me", "0")
	form.Add("sort_by", "popularity.desc")
	form.Add("vote_average.gte", "0")
	form.Add("vote_average.lte", "10")
	form.Add("vote_count.gte", "0")
	form.Add("watch_region", "DE")
	form.Add("with_genres", "")
	form.Add("with_keywords", "")
	form.Add("with_networks", "")
	form.Add("with_origin_country", "")
	form.Add("with_original_language", "")
	form.Add("with_watch_monetization_types", "")
	form.Add("with_watch_providers", "")
	form.Add("with_release_type", "")
	form.Add("with_runtime.gte", "0")
	form.Add("with_runtime.lte", "400")

	req, err := http.NewRequest("POST", urlStr, bytes.NewBufferString(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("[fetchDataFromAPI] error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("Refer", "https://www.themoviedb.org/movie")
	req.Header.Set("Origin", "https://www.themoviedb.org")
	req.Header.Set("x-requested-with", "XMLHttpRequest")
	req.Header.Set("cache-control", "no-cache")
	req.Header.Set("user-agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")

	// Setup proxy if it is set in the environment variable
	proxyURL := os.Getenv("HTTP_PROXY")
	client := &http.Client{}
	if proxyURL != "" {
		proxy, err := url.Parse(proxyURL)
		if err != nil {
			return "", fmt.Errorf("[fetchDataFromAPI] error parsing proxy URL: %w", err)
		}
		client.Transport = &http.Transport{Proxy: http.ProxyURL(proxy)}
	}

	// Send the request using the configured client
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("[fetchDataFromAPI] error sending request: %w", err)
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("[fetchDataFromAPI] error reading response body: %w", err)
	}

	// Convert the body to string
	responseString := string(body)
	return responseString, err
}

func fetchAndStoreMovies(db *gorm.DB, totalPages int) {
	for page := 1; page <= totalPages; page++ {
		response, err := fetchDataFromAPI(page)
		if err != nil {
			log.Printf("Error fetching data from API for page %d: %v\n", page, err)
			continue
		}
		movies, err := parseAPIResponse(response)
		if err != nil {
			log.Printf("Error parsing API response for page %d: %v\n", page, err)
			continue
		}
		for _, movie := range movies {
			db.Create(movie)
		}
	}
}

func moviesHandler(w http.ResponseWriter, r *http.Request) {
	db := mustSetupDB()
	var movies []Movie

	// Get search query
	searchQuery := r.URL.Query().Get("q")
	if searchQuery != "" {
		db.Where("title LIKE ?", "%"+searchQuery+"%").Find(&movies)
	} else {
		db.Find(&movies)
	}

	tmpl := `
		<!DOCTYPE html>
		<html lang="en">
		<head>
			<meta charset="UTF-8">
			<meta name="viewport" content="width=device-width, initial-scale=1.0">
			<title>Movies</title>
		</head>
		<body>
			<h1>Movies</h1>
			<form method="GET" action="/">
				<input type="text" name="q" placeholder="Search by name" value="{{.SearchQuery}}">
				<input type="submit" value="Search">
			</form>
			<ul>
				{{range .Movies}}
					<li>{{.Title}} - {{.ReleaseDate}} - Rating: {{.Rating}}</li>
				{{end}}
			</ul>
		</body>
		</html>
	`

	data := struct {
		Movies      []Movie
		SearchQuery string
	}{
		Movies:      movies,
		SearchQuery: searchQuery,
	}

	t, err := template.New("movies").Parse(tmpl)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	err = t.Execute(w, data)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func clearDB(db *gorm.DB) {
	err := db.Exec("DELETE FROM movies").Error
	if err != nil {
		log.Fatal("failed to clear database")
	}
}

func main() {
	db := mustSetupDB()
	clearDB(db)
	totalPages := 5 // Set the number of pages you want to paginate through
	fetchAndStoreMovies(db, totalPages)

	http.HandleFunc("/", moviesHandler)

	log.Println("Starting server on :8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal(err)
	}
}
