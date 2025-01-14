package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

func main() {
	res, err := http.Get("https://linusmediagroup.com/careers")
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		log.Fatalf("status code error: %d %s", res.StatusCode, res.Status)
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	var jobs []string

	doc.Find(".accordion-item__title").Each(func(i int, s *goquery.Selection) {
		jobs = append(jobs, strings.TrimSpace(s.Text()))
	})

	for _, j := range jobs {
		fmt.Printf("%s\n", j)
	}
}
