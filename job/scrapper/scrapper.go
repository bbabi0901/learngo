package scrapper

import (
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type jobResult struct {
	id       string
	title    string
	location string
	summary  string
	rating   string
}

// Scrape indeed by term
func Scrape(term string) {
	startTime := time.Now()
	var baseURL string = "https://kr.indeed.com/jobs?q=" + term + "&limit=50"
	var jobs []jobResult
	c := make(chan []jobResult)

	totalPages := getPages(baseURL)

	for i := 0; i < totalPages; i++ {
		go getPage(baseURL, i, c)
	}

	for i := 0; i < totalPages; i++ {
		jobPage := <-c
		jobs = append(jobs, jobPage...) // append the contents of jobPage, not slice itself
	}
	writeJobs(jobs)
	endTime := time.Now()
	fmt.Println("Operating time:", endTime.Sub(startTime))
}

func writeJobs(jobs []jobResult) {
	c := make(chan []string)

	file, err := os.Create("jobs.csv ")
	checkErr(err)

	w := csv.NewWriter(file)
	defer w.Flush()

	headers := []string{"ID", "Title", "Location", "Summary", "Rating"}
	wErr := w.Write(headers)
	checkErr(wErr)

	for _, job := range jobs {
		go writeJobsDetail(job, c)
	}

	for i := 0; i < len(jobs); i++ {
		jobData := <-c
		jwErr := w.Write(jobData)
		checkErr(jwErr)
	}
}

func writeJobsDetail(job jobResult, c chan<- []string) {
	const jobURL = "https://kr.indeed.com/viewjob?jk="
	c <- []string{jobURL + job.id, job.title, job.location, job.summary, job.rating}
}

func getPage(baseURL string, page int, mainChannel chan<- []jobResult) {
	var jobs []jobResult
	c := make(chan jobResult)
	pageURL := baseURL + "&start=" + strconv.Itoa(page*50)
	resp, err := http.Get(pageURL)
	checkErr(err)
	checkCode(resp)

	defer resp.Body.Close()
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	checkErr(err)

	searchCards := doc.Find(".tapItem")
	searchCards.Each(func(i int, card *goquery.Selection) {
		go extractJobResult(card, c)

	})

	for i := 0; i < searchCards.Length(); i++ {
		job := <-c
		jobs = append(jobs, job)
	}

	mainChannel <- jobs
}

func extractJobResult(card *goquery.Selection, c chan<- jobResult) {
	id, _ := card.Attr("data-jk")
	title := CleanStrings(card.Find("h2>span").Text())
	location := CleanStrings(card.Find(".companyLocation").Text())
	summary := CleanStrings(card.Find(".job-snippet").Text())
	rating := card.Find(".ratingNumber").Text()

	c <- jobResult{
		id:       id,
		title:    title,
		location: location,
		summary:  summary,
		rating:   rating,
	}
}

// Cleans a string
func CleanStrings(str string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(str)), " ")
}

func getPages(baseURL string) (pages int) {
	pages = 0
	resp, err := http.Get(baseURL)
	checkErr(err)
	checkCode(resp)

	defer resp.Body.Close()
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	checkErr(err)

	doc.Find(".pagination").Each(func(i int, s *goquery.Selection) {
		pages = s.Find("a").Length()
	})
	return
}

func checkErr(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

func checkCode(resp *http.Response) {
	if resp.StatusCode != 200 {
		log.Fatalln("Status Code:", resp.StatusCode)
	}
}
