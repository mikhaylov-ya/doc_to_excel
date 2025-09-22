package main

import (
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func GetJournalPage(journal string) {
	pubdateRegex := regexp.MustCompile(`\d{1,2}\.\d{2}\.\d{4}.?`)
	parts := strings.Split((journal), ".")
	journals := []string{"IZ", "AS", "REJ", "EEJ"}
	journalPrefix := strings.ToUpper(parts[0])
	// journalVol := strings.ToUpper(parts[1])
	journalNum := strings.ToUpper(parts[2])
	if !contains(journals, journalPrefix) {
		log.Fatalf("Journal %s not found in the list", journal)
		return
	}

	journalKeyCatalogMap := map[string]string{
		"IZ":  "Inv_Zool",
		"AS":  "AS",
		"REJ": "REJ",
		"EEJ": "EEJ",
	}
	journalURL := fmt.Sprintf("https://kmkjournals.com/journals/%s/%s_Index_Volumes", journalKeyCatalogMap[journalPrefix], journalPrefix)
	// 1. Fetch the webpage
	res, err := http.Get(journalURL)
	if err != nil {
		log.Fatalf("Failed to fetch page: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		log.Fatalf("Status code error: %d %s", res.StatusCode, res.Status)
	}

	// 2. Parse the HTML
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.Fatalf("Failed to parse HTML: %v", err)
	}
	doc.Find("p").Each(func(i int, s *goquery.Selection) {
		if strings.Contains(s.Text(), fmt.Sprintf("Number %s", journalNum)) {
			publicationDate := pubdateRegex.FindStringSubmatch(s.Text())
			if len(publicationDate) > 0 {
				fmt.Printf("Publication date for %s: %s\n", journal, publicationDate[0])
			} else {
				fmt.Println("Publication date not found")
			}
		}
	})
}
