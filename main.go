// Воркфлоу:
// 1. Прочитать 4 эксель-файла с данными журналов
// 2. Прочитать и распарсить док-файл номера журнала
// 3. Записать новые данные в эти файлы
package main

import (
	"fmt"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"code.sajari.com/docconv/v2"
	"github.com/xuri/excelize/v2"
)

func deleteSubstring(s string) string {
	return ""
}

func formatPageNumbers(pages string) string {
	if pages == "" {
		return ""
	}

	// Extract the page range using a more comprehensive regex that handles all dash types
	pagesRegex := regexp.MustCompile(`(\d+)[–-—](\d+)`)
	matches := pagesRegex.FindStringSubmatch(pages)

	if len(matches) == 3 {
		startPage := matches[1]
		endPage := matches[2]

		// Convert to integers to handle leading zeros properly
		start, err1 := strconv.Atoi(startPage)
		end, err2 := strconv.Atoi(endPage)

		if err1 == nil && err2 == nil {
			// Format with leading zeros (3 digits)
			return fmt.Sprintf("%03d-%03d", start, end)
		}
	}

	return pages
}

func writeOutput(articles []string) {
	sfs, err := os.OpenFile("output.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	arts := make([]string, len(articles))
	defer sfs.Close()
	for i := 0; i < len(articles); i++ {
		var sb strings.Builder
		sb.WriteString(strconv.Itoa(i))
		sb.WriteString(" ")
		sb.WriteString(articles[i])
		arts[i] = sb.String()
	}
	_, write_err := sfs.WriteString(strings.Join(arts, "\n"))
	if write_err != nil {
		panic(write_err)
	}
}

func printError(artNum int, message string) {
	fmt.Printf("Error in article %d: %s\n", artNum, message)
}

func main() {
	docPath := os.Args[1]
	if docPath == "" {
		panic("Path to doc file is not passed")
	}
	res, err := docconv.ConvertPath(docPath)
	if err != nil {
		panic(err)
	}
	yearRegex := regexp.MustCompile(`\d\d\d\d`)
	numsRegex := regexp.MustCompile(`[[:alpha:].](\d)`)
	pagesRegex := regexp.MustCompile(`(\d+)[–-—](\d+)`)
	abstractRegex := regexp.MustCompile(`(?i)abstract[s.:]`)
	kwRegex := regexp.MustCompile(`(?i)key\s?words[.:]`)
	doiRegex := regexp.MustCompile(`(?i)\bdoi(?:\s|\.|:)\s?(\d[^\n]*)`)
	authSuffxRegex := regexp.MustCompile(`\d+(,)?(\*)?`)
	refSepRegex := regexp.MustCompile(`\r\n|\r|\n`)
	mailSeps := [4]string{"E-mail", "Email", "email", "e-mail"}

	artRefSep := regexp.MustCompile(`(?s)(.*?)<<<(.*?)>>>`)
	matches := artRefSep.FindAllStringSubmatch(res.Body, -1)

	// Collect parsed articles and references
	var articles []string
	var references []string
	for _, match := range matches {
		articles = append(articles, match[1])     // Article content before <<<
		references = append(references, match[2]) // References between <<< >>>
	}

	fmt.Printf("Articles found: %d\n", len(articles))

	articlesNormalized := make([]Article, len(articles))
	for artIndex, art := range articles {
		normArt := Article{}
		referencesArr := refSepRegex.Split(references[artIndex], -1)
		normArt.references = referencesArr
		if len(abstractRegex.Split(art, 2)) < 2 {
			printError(artIndex+1, fmt.Sprintf("Can't get Abstract from article data: %s", art))
			fmt.Print(art)
		}
		abstractAndKW := kwRegex.Split(abstractRegex.Split(art, 2)[1], 2)
		artAbstract, artKW :=
			abstractRegex.ReplaceAllStringFunc(abstractAndKW[0], deleteSubstring),
			kwRegex.ReplaceAllStringFunc(abstractAndKW[1], deleteSubstring)
		normArt.abstract = strings.TrimSpace(artAbstract)
		normArt.keywords = strings.TrimSpace(artKW)

		// Try different line ending formats
		var artStrings []string

		// First try splitting by \r\n (Windows line endings)
		artRaw := strings.Split(art, "\r\n")
		if len(artRaw) == 1 {
			// Try splitting by \n (Unix line endings)
			artRaw = strings.Split(art, "\n")
		}
		if len(artRaw) == 1 {
			// Try splitting by \r (Mac line endings)
			artRaw = strings.Split(art, "\r")
		}

		for _, str := range artRaw {
			// Trim spaces and check if the string is not empty
			if trimmed := strings.TrimSpace(str); trimmed != "" {
				artStrings = append(artStrings, trimmed) // Add valid strings to the new slice
			}
		}

		writeOutput(artStrings)
		// DOI LOOP
		var doi string
		for _, str := range artStrings {
			if doiRegex.MatchString(str) {
				doi = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(strings.TrimPrefix(str, "doi"), ":"), "."))
			}
		}
		normArt.doi = doi
		splittedAuthorsTitleAndMeta := yearRegex.Split(art, 2)
		if len(splittedAuthorsTitleAndMeta) < 2 {
			fmt.Print(art)
			printError(artIndex+1, fmt.Sprintf("Something went wrong when splitting author, title and meta by year: %s", splittedAuthorsTitleAndMeta))
		}

		authorsRaw, titleAndMeta := splittedAuthorsTitleAndMeta[0], splittedAuthorsTitleAndMeta[1]
		splittedTitleMeta := strings.Split(titleAndMeta, "//")
		// If there's no "//" in the string, split by "/"
		if len(splittedTitleMeta) == 1 {
			splittedTitleMeta = strings.Split(titleAndMeta, "/")
		}

		title, numberMeta := splittedTitleMeta[0], splittedTitleMeta[1]
		normArt.title = strings.TrimSpace(strings.TrimPrefix(title, "."))
		normArt.pages = formatPageNumbers(pagesRegex.FindString(numberMeta))
		// (start) ----- AUTHORS BLOCK -------
		authorsNormalized := []string{}
		for _, auth := range strings.Split(authorsRaw, ", ") {
			authorsNormalized = append(authorsNormalized, authSuffxRegex.ReplaceAllStringFunc(auth, deleteSubstring))
		}
		normArt.authors = strings.TrimSpace(strings.Join(authorsNormalized, ", "))
		// (end) ----- AUTHORS BLOCK -------
		// (start) ----- AFFILIATIONS BLOCK -------
		// authorsRaw is just surnames with or without number, so we extract digits here
		authorAffilNums := numsRegex.FindAllStringSubmatch(authorsRaw, -1)
		affilations := make([]string, len(authorsNormalized))
		// Fill affiliations with same value if not enumerated
		if len(authorAffilNums) == 0 {
			// Look for affiliation line - it's usually the first line that contains an email or address
			affiliationLine := ""
			for i, str := range artStrings {
				// Skip the first line (title/metadata) and look for lines with email or address patterns
				if i > 0 && (strings.Contains(str, "@") || strings.Contains(str, "E-mail") || strings.Contains(str, "Email") || strings.Contains(str, "Russia") || strings.Contains(str, "China") || strings.Contains(str, "USA")) {
					affiliationLine = str
					break
				}
			}

			if affiliationLine != "" {
				fmt.Println("Affiliation (no enumeration):", affiliationLine)
				for j := range affilations {
					affilations[j] = strings.TrimPrefix(affiliationLine, "1")
				}
			} else {
				fmt.Println("Warning: No affiliation data found for article", artIndex+1)
				for j := range affilations {
					affilations[j] = ""
				}
			}
		} else {
			affiliationsNumerated := make([]string, 0)
			if len(artStrings) > 1 {
				affiliationsNumerated = make([]string, len(artStrings)-1)
				copy(affiliationsNumerated, artStrings[1:])
			} else {
				fmt.Println("Warning: No affiliation data found for article", artIndex+1)
			}
			for i, match := range authorAffilNums {
				idx := slices.IndexFunc(affiliationsNumerated, func(s string) bool { return strings.HasPrefix(s, match[1]) })
				if idx != -1 {
					if i >= len(affilations) {
						printError(artIndex+1, "Authors have more affilation numbers than affilations")
					}
					affilations[i] = strings.TrimPrefix(affiliationsNumerated[idx], match[1])
				} else {
					printError(artIndex+1, fmt.Sprintf("Affilation not found: %s", match))
				}
			}
		}
		for i, aff := range affilations {
			for _, sep := range mailSeps {
				before, _, found := strings.Cut(aff, sep)
				if found {
					affilations[i] = before
					break
				}
			}
			affilations[i], _, _ = strings.Cut(affilations[i], ";")
			affilations[i] = strings.TrimSuffix(strings.TrimSpace(affilations[i]), ".")
		}
		normArt.affiliations = strings.Join(affilations, "; ")
		articlesNormalized[artIndex] = normArt
		// (end) ----- AFFILIATIONS BLOCK -------
	}

	f := excelize.NewFile()
	defer func() {
		if err := f.Close(); err != nil {
			fmt.Println(err)
		}
	}()
	f.NewSheet("References")
	f.NewSheet("Doi")

	var refI = 0
	for artI, art := range articlesNormalized {
		artNumStr := strconv.Itoa(artI + 1)
		pagesCell := fmt.Sprintf("A%s", artNumStr)
		authorsCell := fmt.Sprintf("B%s", artNumStr)
		affiliationsCell := fmt.Sprintf("C%s", artNumStr)
		titleCell := fmt.Sprintf("D%s", artNumStr)
		keywordCell := fmt.Sprintf("E%s", artNumStr)
		abstractCell := fmt.Sprintf("F%s", artNumStr)
		numCell := fmt.Sprintf("G%s", artNumStr)
		doiCell := fmt.Sprintf("H%s", artNumStr)

		doiSheetdoiCell := fmt.Sprintf("B%s", artNumStr)
		f.SetCellValue("Sheet1", pagesCell, art.pages)
		f.SetCellValue("Sheet1", authorsCell, art.authors)
		f.SetCellValue("Sheet1", affiliationsCell, art.affiliations)
		f.SetCellValue("Sheet1", titleCell, art.title)
		f.SetCellValue("Sheet1", keywordCell, art.keywords)
		f.SetCellValue("Sheet1", abstractCell, art.abstract)
		f.SetCellValue("Sheet1", numCell, artNumStr)
		f.SetCellValue("Sheet1", doiCell, art.doi)

		f.SetCellValue("Doi", doiSheetdoiCell, art.doi)
		fmt.Printf("Article %d: Success\n", artI+1)

		for artRefI, ref := range art.references {
			refI += 1
			if artRefI == len(art.references)-1 {
				ref = strings.TrimSuffix(ref, ">>>")
			}
			f.SetCellValue("References", fmt.Sprintf("A%s", strconv.Itoa(refI)), ref)
			authors, year, title, meta := parseReference(ref)
			f.SetCellValue("References", fmt.Sprintf("B%s", strconv.Itoa(refI)), authors)
			f.SetCellValue("References", fmt.Sprintf("C%s", strconv.Itoa(refI)), year)
			f.SetCellValue("References", fmt.Sprintf("D%s", strconv.Itoa(refI)), title)
			f.SetCellValue("References", fmt.Sprintf("E%s", strconv.Itoa(refI)), meta)
			f.SetCellValue("References", fmt.Sprintf("F%s", strconv.Itoa(refI)), art.doi)
		}
	}
	journalPathSplit := strings.SplitAfter(docPath, "/")
	journalInfo := journalPathSplit[len(journalPathSplit)-1]
	journal := strings.Replace(strings.SplitAfter(journalInfo, ".")[0], "doi.", "", 1)

	fmt.Println(journal)
	links := GetJournalPage(journal)
	for i, link := range links {
		fmt.Println(link)
		f.SetCellValue("Doi", fmt.Sprintf("C%s", strconv.Itoa(i+1)), link)
	}

	// Save spreadsheet by the given path.
	if err := f.SaveAs("Book1.xlsx"); err != nil {
		fmt.Println("Error saving spreadsheet: ", err)
	}

}
