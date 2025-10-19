package main

import (
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// journal format: eej24_1 (journal_volume_number)
func GetJournalPage(journal string) []string {
	parts := strings.Split((journal), "_")
	if len(parts) < 2 {
		log.Fatalf("Invalid journal format: %s. Expected format: journal_volume_number", journal)
		return []string{}
	}

	// Extract journal prefix and volume
	journalRegex := regexp.MustCompile(`^([a-zA-Z]+)(\d+)$`)
	matches := journalRegex.FindStringSubmatch(parts[0])
	if len(matches) != 3 {
		log.Fatalf("Invalid journal format: %s", journal)
		return []string{}
	}
	journalPrefix := strings.ToUpper(matches[1])
	journalVol := strings.TrimPrefix(matches[2], "0")
	journalNum := strings.TrimPrefix(parts[1], "0")

	journalKeyCatalogMap := map[string]string{
		"IZ":  "Inv_Zool",
		"AS":  "AS",
		"REJ": "REJ",
		"EEJ": "EEJ",
	}

	journalURL := fmt.Sprintf("https://kmkjournals.com/journals/%s/%s_Index_Volumes",
		journalKeyCatalogMap[journalPrefix], journalPrefix)

	// Fetch page
	res, err := http.Get(journalURL)
	if err != nil {
		log.Fatalf("Failed to fetch page: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.Fatalf("Status code error: %d %s", res.StatusCode, res.Status)
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.Fatalf("Failed to parse HTML: %v", err)
	}

	// 1. Find <h1> with Volume
	var numberNode *goquery.Selection
	doc.Find("h1").EachWithBreak(func(i int, h1 *goquery.Selection) bool {
		if strings.Contains(h1.Text(), fmt.Sprintf("Volume %s", journalVol)) {
			// 2. Walk siblings to find <p> with Number <num>
			for s := h1.Next(); s.Length() > 0; s = s.Next() {
				if goquery.NodeName(s) == "p" &&
					strings.Contains(s.Text(), fmt.Sprintf("Number %s", journalNum)) {
					numberNode = s
					return false // stop outer EachWithBreak
				}
			}
		}
		return true
	})

	if numberNode == nil {
		log.Fatalf("Could not find Volume %s Number %s", journalVol, journalNum)
		return []string{}
	}
	links := []string{}
	// 3. Collect articles until next Number/Volume
	baseURL := "https://kmkjournals.com"
	for s := numberNode.Next(); s.Length() > 0; s = s.Next() {
		if goquery.NodeName(s) == "h1" && strings.Contains(s.Text(), "Volume") {
			break
		}
		if goquery.NodeName(s) == "p" &&
			strings.Contains(s.Text(), "Number") {
			break
		}

		if goquery.NodeName(s) == "p" {
			// first <a> is article, second <a.pdf> is PDF
			linkSel := s.Find("a").First()
			fmt.Println(linkSel)
			if linkSel.Length() > 0 {
				href, _ := linkSel.Attr("href")
				if href != "" && !strings.Contains(href, ".pdf") {
					links = append(links, baseURL+href)
				}
			}
		}
	}
	return links
}
