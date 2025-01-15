package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
)

type config struct {
	url string
	cb  func(cfg config, doc *goquery.Document, ch chan result)
}

type result struct {
	jobTitle string
	url      string
}

func (r result) String() string {
	return fmt.Sprintf("%s (%s)", r.jobTitle, r.url)
}

func scrape(cfg config, wg *sync.WaitGroup, ch chan result, errCh chan error) {
	defer wg.Done()

	res, err := http.Get(cfg.url)
	if err != nil {
		errCh <- fmt.Errorf("failed to fetch %s: %w", cfg.url, err)
		return
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		errCh <- fmt.Errorf("status code error: %d %s", res.StatusCode, res.Status)
		return
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		errCh <- fmt.Errorf("failed to parse document from %s: %w", cfg.url, err)
		return
	}

	cfg.cb(cfg, doc, ch)
}

func main() {
	configs := []config{
		{
			url: "https://linusmediagroup.com/careers",
			cb: func(cfg config, doc *goquery.Document, ch chan result) {
				doc.Find(".accordion-item__title").Each(func(i int, s *goquery.Selection) {
					job := strings.TrimSpace(s.Text())
					if strings.Contains(strings.ToLower(job), "software") {
						ch <- result{url: cfg.url, jobTitle: job}
					}
				})
			},
		},
		{
			url: "https://vercel.com/careers",
			cb: func(cfg config, doc *goquery.Document, ch chan result) {
				doc.Find("a[href^='/careers/'] p:first-of-type").Each(func(i int, s *goquery.Selection) {
					job := strings.TrimSpace(s.Text())
					if strings.Contains(strings.ToLower(job), "software") {
						ch <- result{url: cfg.url, jobTitle: job}
					}
				})
			},
		},
	}

	var wg sync.WaitGroup
	ch := make(chan result, len(configs)*10)
	errCh := make(chan error, len(configs))

	wg.Add(len(configs))

	for _, cfg := range configs {
		go scrape(cfg, &wg, ch, errCh)
	}

	go func() {
		wg.Wait()
		close(ch)
		close(errCh)
	}()

	for result := range ch {
		fmt.Println(result)
	}

	for err := range errCh {
		log.Println("Error:", err)
	}
}
