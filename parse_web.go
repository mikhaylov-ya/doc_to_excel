package main

import (
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// JournalInfo contains parsed journal metadata
type JournalInfo struct {
	Volume  string
	Issue   string
	Pubdate string
	Links   []string
}

// GetJournalPage extracts journal information from DOI and fetches article links
// DOI format examples:
// - euroasentj.24.03.02 (EEJ journal, volume 24, number 3)
// - rusentj.34.3.01 (REJ journal, vol 34, number 3)
// - invertzool.22.3.01 (IZ journal, volume 22 number 3)
// - arthsel.34.3.01 (AS journal, volume 34, number 3)
func GetJournalPage(doi string) JournalInfo {
	if doi == "" {
		log.Fatalf("DOI is empty")
		return JournalInfo{}
	}

	// Parse DOI to extract journal, volume, and number
	// DOI format: [prefix/]<journal>.<volume>.<number>.<article>
	// Example: 10.15298/euroasentj.24.01.01 or euroasentj.24.01.01
	doiRegex := regexp.MustCompile(`(?:[\d.]+/)?([a-z]+)\.(\d+)\.(\d+)\.(\d+)$`)
	matches := doiRegex.FindStringSubmatch(doi)
	if len(matches) != 5 {
		log.Fatalf("Invalid DOI format: %s. Expected format: [prefix/]journal.volume.number.article (e.g., 10.15298/euroasentj.24.01.01)", doi)
		return JournalInfo{}
	}

	journalCode := matches[1]
	journalVol := strings.TrimPrefix(matches[2], "0")
	journalNum := strings.TrimPrefix(matches[3], "0")

	// Map DOI journal codes to full journal prefixes
	journalCodeMap := map[string]string{
		"euroasentj":  "EEJ",
		"rusentj":     "REJ",
		"invertzool":  "IZ",
		"arthsel":     "AS",
	}

	journalPrefix, ok := journalCodeMap[journalCode]
	if !ok {
		log.Fatalf("Unknown journal code in DOI: %s", journalCode)
		return JournalInfo{}
	}

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
	var pubdate string
	doc.Find("h1").EachWithBreak(func(i int, h1 *goquery.Selection) bool {
		if strings.Contains(h1.Text(), fmt.Sprintf("Volume %s", journalVol)) {
			// 2. Walk siblings to find <p> with Number <num>
			for s := h1.Next(); s.Length() > 0; s = s.Next() {
				if goquery.NodeName(s) == "p" &&
					strings.Contains(s.Text(), fmt.Sprintf("Number %s", journalNum)) {
					numberNode = s

					// Extract pubdate from text like "Number 3. Published on 20.06.2025"
					numberText := s.Text()
					dateRegex := regexp.MustCompile(`Published on (\d{2}\.\d{2}\.\d{4})`)
					dateMatches := dateRegex.FindStringSubmatch(numberText)
					if len(dateMatches) > 1 {
						pubdate = dateMatches[1]
					}

					return false // stop outer EachWithBreak
				}
			}
		}
		return true
	})

	if numberNode == nil {
		log.Fatalf("Could not find Volume %s Number %s", journalVol, journalNum)
		return JournalInfo{}
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

	return JournalInfo{
		Volume:  journalVol,
		Issue:   journalNum,
		Pubdate: pubdate,
		Links:   links,
	}
}
