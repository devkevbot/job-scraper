package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type scrapeTask struct {
	url      string
	selector string
}

type scrapeResult struct {
	jobTitle string
	url      string
}

func (r scrapeResult) String() string {
	return fmt.Sprintf("%s (%s)", r.jobTitle, r.url)
}

func fetchDocument(ctx context.Context, url string) (*goquery.Document, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for %s: %w", url, err)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch %s: %w", url, err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d for %s", res.StatusCode, url)
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse document from %s: %w", url, err)
	}
	return doc, nil
}

func scrape(ctx context.Context, task scrapeTask, resCh chan scrapeResult, errCh chan error) {
	doc, err := fetchDocument(ctx, task.url)
	if err != nil {
		errCh <- err
		return
	}

	doc.Find(task.selector).Each(func(_ int, s *goquery.Selection) {
		job := strings.TrimSpace(s.Text())
		lowerJob := strings.ToLower(job)
		if strings.Contains(lowerJob, "software") || strings.Contains(lowerJob, "engineer") {
			resCh <- scrapeResult{url: task.url, jobTitle: job}
		}
	})
}

func main() {
	tasks := []scrapeTask{
		{url: "https://linusmediagroup.com/careers", selector: ".accordion-item__title"},
		{url: "https://vercel.com/careers", selector: "a[href^='/careers/'] p:first-of-type"},
		{url: "https://stripe.com/jobs/search", selector: ".JobsListings__link"},
	}

	var wg sync.WaitGroup
	resChan := make(chan scrapeResult, len(tasks)*10)
	errChan := make(chan error, len(tasks))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Start scraping tasks
	wg.Add(len(tasks))
	for _, task := range tasks {
		go func(task scrapeTask) {
			defer wg.Done()
			scrape(ctx, task, resChan, errChan)
		}(task)
	}

	// Close channels after all goroutines complete
	go func() {
		wg.Wait()
		close(resChan)
		close(errChan)
	}()

	var seenJobs sync.Map

	// Process results and errors
	for resChan != nil || errChan != nil {
		select {
		case result, ok := <-resChan:
			if !ok {
				resChan = nil
				continue
			}
			if _, exists := seenJobs.LoadOrStore(result.jobTitle, struct{}{}); !exists {
				fmt.Println(result)
			}
		case err, ok := <-errChan:
			if !ok {
				errChan = nil
				continue
			}
			log.Println("Error:", err)
		}
	}
}
