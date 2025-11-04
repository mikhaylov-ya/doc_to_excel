// Воркфлоу:
// 1. Прочитать 4 эксель-файла с данными журналов
// 2. Прочитать и распарсить док-файл номера журнала
// 3. Записать новые данные в эти файлы
package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

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
	fmt.Printf("⚠️  [Article %d] ERROR: %s\n", artNum, message)
}

func printWarning(artNum int, field string, message string) {
	fmt.Printf("⚠️  [Article %d] WARNING - %s: %s\n", artNum, field, message)
}

// processDocument converts a DOC/DOCX file to Excel format
// Returns error if processing fails
func processDocument(docPath, outputPath string) error {
	if docPath == "" {
		return fmt.Errorf("path to doc file is not provided")
	}
	res, err := docconv.ConvertPath(docPath)
	if err != nil {
		return fmt.Errorf("failed to convert document: %w", err)
	}

	// Fallback: if docconv returns empty body for .doc files, try catdoc/antiword directly
	if len(res.Body) == 0 && strings.HasSuffix(strings.ToLower(docPath), ".doc") {
		fmt.Println("Warning: docconv returned empty content, trying fallback converters...")

		// Try multiple converters in order of preference (catdoc for better encoding)
		converters := []struct {
			name string
			path string
		}{
			{"catdoc", "/usr/bin/catdoc"},
			{"catdoc", "/bin/catdoc"},
			{"wvText", "/usr/bin/wvText"},
		}

		converted := false
		var lastErr error

		for _, conv := range converters {
			if _, err := os.Stat(conv.path); err == nil {
				cmd := exec.Command(conv.path, docPath)
				output, convErr := cmd.Output()
				if convErr == nil && len(output) > 0 {
					res.Body = string(output)
					fmt.Printf("✓ Successfully converted using %s (%d bytes)\n", conv.name, len(res.Body))
					converted = true
					break
				}
				lastErr = convErr
			}
		}

		if !converted {
			if lastErr != nil {
				return fmt.Errorf("docconv returned empty content and all fallback converters failed. Last error: %v", lastErr)
			}
			return fmt.Errorf("docconv returned empty content and no fallback converters found (tried: catdoc, wvText)")
		}
	}
	yearRegex := regexp.MustCompile(`\d\d\d\d`)
	numsRegex := regexp.MustCompile(`[[:alpha:].](\d)`)
	pagesRegex := regexp.MustCompile(`(\d+)[–-—](\d+)`)
	abstractRegex := regexp.MustCompile(`(?i)abstract[s.:]`)
	kwRegex := regexp.MustCompile(`(?i)key\s*words[.:]`)
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
		// Extract abstract and keywords
		var artAbstract, artKW string
		if len(abstractRegex.Split(art, 2)) < 2 {
			printError(artIndex+1, "ABSTRACT section not found in article text")
			// Continue with empty abstract rather than failing
			artAbstract = ""
			artKW = ""
		} else {
			abstractAndKW := kwRegex.Split(abstractRegex.Split(art, 2)[1], 2)
			if len(abstractAndKW) > 0 {
				artAbstract = abstractRegex.ReplaceAllStringFunc(abstractAndKW[0], deleteSubstring)
			}
			if len(abstractAndKW) > 1 {
				artKW = kwRegex.ReplaceAllStringFunc(abstractAndKW[1], deleteSubstring)
			} else {
				// Keywords might be missing, use empty string
				artKW = ""
				printWarning(artIndex+1, "KEYWORDS", "Keywords section not found, continuing with empty keywords")
			}
		}
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
		if doi == "" {
			printWarning(artIndex+1, "DOI", "DOI not found in article")
		}
		normArt.doi = doi

		splittedAuthorsTitleAndMeta := yearRegex.Split(art, 2)
		if len(splittedAuthorsTitleAndMeta) < 2 {
			printError(artIndex+1, "YEAR: Cannot split authors/title by year pattern")
			// Continue processing with empty values
			normArt.title = ""
			normArt.pages = ""
			normArt.authors = ""
			normArt.affiliations = ""
			articlesNormalized[artIndex] = normArt
			continue
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
						printError(artIndex+1, "AFFILIATIONS: More affiliation numbers on authors than available affiliations")
					}
					affilations[i] = strings.TrimPrefix(affiliationsNumerated[idx], match[1])
				} else {
					printWarning(artIndex+1, "AFFILIATIONS", fmt.Sprintf("Affiliation number %s not found in text", match[1]))
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
		fmt.Printf("✓ [Article %d] Parsed successfully: DOI=%s, Title='%s'\n", artIndex+1, normArt.doi, normArt.title[:min(50, len(normArt.title))])
		// (end) ----- AFFILIATIONS BLOCK -------
	}

	f := excelize.NewFile()
	defer func() {
		if err := f.Close(); err != nil {
			fmt.Println(err)
		}
	}()

	// Rename Sheet1 to "articles"
	f.SetSheetName("Sheet1", "articles")

	// Create column headers
	headers := []struct {
		cell  string
		value string
	}{
		{"A1", "articles.total_number"},
		{"B1", "pubdate"},
		{"C1", "articles.volume"},
		{"D1", "articles.issue"},
		{"E1", "articles.pages"},
		{"F1", "articles.authors"},
		{"G1", "articles.affilations"},
		{"H1", "articles.title"},
		{"I1", "articles.key_words"},
		{"J1", "articles.summary"},
		{"K1", "articles.number"},
		{"L1", "articles.DOI"},
	}

	for _, h := range headers {
		f.SetCellValue("articles", h.cell, h.value)
	}

	f.NewSheet("References")
	f.NewSheet("Doi")

	// Parse web data BEFORE filling the articles sheet
	// Extract journal info from the first article's DOI
	var journalInfo JournalInfo
	if len(articlesNormalized) == 0 {
		return fmt.Errorf("no articles parsed from document")
	}
	if articlesNormalized[0].doi == "" {
		return fmt.Errorf("first article has no DOI - cannot determine journal information")
	}

	fmt.Println("Using DOI from first article:", articlesNormalized[0].doi)
	journalInfo = GetJournalPage(articlesNormalized[0].doi)
	fmt.Printf("Journal Info - Volume: %s, Issue: %s, Pubdate: %s, Articles: %d\n",
		journalInfo.Volume, journalInfo.Issue, journalInfo.Pubdate, len(journalInfo.Links))

	// Fill the Doi sheet with article links
	for i, link := range journalInfo.Links {
		fmt.Println(link)
		f.SetCellValue("Doi", fmt.Sprintf("A%s", strconv.Itoa(i+1)), link)
	}

	// STATE MANAGEMENT: Load state and handle numbering
	journalCode, err := ExtractJournalCodeFromDOI(articlesNormalized[0].doi)
	if err != nil {
		return fmt.Errorf("failed to extract journal code: %w", err)
	}

	stateManager := NewStateManager()
	state, err := stateManager.LoadState(journalCode)
	if err != nil {
		return fmt.Errorf("failed to load state: %w", err)
	}

	// Check if state is configured (starting point set)
	if !stateManager.IsConfigured(state) {
		if err := stateManager.PromptForStartingPoint(state, journalInfo.Volume, journalInfo.Issue); err != nil {
			return fmt.Errorf("failed to configure state: %w", err)
		}
	}

	// Check for duplicate issue
	var startNum, endNum int
	if existingIssue, isDuplicate := stateManager.IsIssueProcessed(state, journalInfo.Volume, journalInfo.Issue); isDuplicate {
		action, err := stateManager.PromptDuplicateAction(*existingIssue, journalCode)
		if err != nil {
			return fmt.Errorf("failed to get user choice: %w", err)
		}

		switch action {
		case SkipProcessing:
			fmt.Println("Skipping processing as requested.")
			return nil
		case ReprocessSameNumbers:
			fmt.Printf("Reprocessing with existing numbers %d-%d\n", existingIssue.StartNumber, existingIssue.EndNumber)
			startNum = existingIssue.StartNumber
			endNum = existingIssue.EndNumber
			// Remove old entry so we can add the new one
			if err := stateManager.RemoveIssue(state, journalInfo.Volume, journalInfo.Issue); err != nil {
				return fmt.Errorf("failed to remove old issue entry: %w", err)
			}
		case ReprocessNewNumbers:
			fmt.Println("Reprocessing with NEW numbers")
			// Remove old entry
			if err := stateManager.RemoveIssue(state, journalInfo.Volume, journalInfo.Issue); err != nil {
				return fmt.Errorf("failed to remove old issue entry: %w", err)
			}
			// Allocate new numbers
			startNum, endNum = stateManager.AllocateNumbers(state, len(articlesNormalized))
			fmt.Printf("Allocated new numbers: %d-%d\n", startNum, endNum)
		case Abort:
			return fmt.Errorf("processing aborted by user")
		}
	} else {
		// New issue: allocate numbers
		startNum, endNum = stateManager.AllocateNumbers(state, len(articlesNormalized))
		fmt.Printf("Allocated article numbers: %d-%d\n", startNum, endNum)
	}

	// Now fill the articles sheet with parsed data and web data
	var refI = 0
	for artI, art := range articlesNormalized {
		artNumStr := strconv.Itoa(artI + 1)
		// Row index is artI + 2 (skip header row)
		rowNum := strconv.Itoa(artI + 2)

		// Map to new column structure:
		// A: articles.total_number (from state management)
		// B: pubdate (from web)
		// C: articles.volume (from web)
		// D: articles.issue (from web)
		// E: articles.pages
		// F: articles.authors
		// G: articles.affilations
		// H: articles.title
		// I: articles.key_words
		// J: articles.summary
		// K: articles.number
		// L: articles.DOI

		// Fill total_number from allocated range
		totalNumber := startNum + artI
		f.SetCellValue("articles", fmt.Sprintf("A%s", rowNum), totalNumber)
		f.SetCellValue("articles", fmt.Sprintf("B%s", rowNum), journalInfo.Pubdate)
		f.SetCellValue("articles", fmt.Sprintf("C%s", rowNum), journalInfo.Volume)
		f.SetCellValue("articles", fmt.Sprintf("D%s", rowNum), journalInfo.Issue)
		f.SetCellValue("articles", fmt.Sprintf("E%s", rowNum), art.pages)
		f.SetCellValue("articles", fmt.Sprintf("F%s", rowNum), art.authors)
		f.SetCellValue("articles", fmt.Sprintf("G%s", rowNum), art.affiliations)
		f.SetCellValue("articles", fmt.Sprintf("H%s", rowNum), art.title)
		f.SetCellValue("articles", fmt.Sprintf("I%s", rowNum), art.keywords)
		f.SetCellValue("articles", fmt.Sprintf("J%s", rowNum), art.abstract)
		f.SetCellValue("articles", fmt.Sprintf("K%s", rowNum), artNumStr)
		f.SetCellValue("articles", fmt.Sprintf("L%s", rowNum), art.doi)
		doiSheetdoiCell := fmt.Sprintf("B%s", artNumStr)

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

	// Save spreadsheet by the given path.
	if err := f.SaveAs(outputPath); err != nil {
		return fmt.Errorf("failed to save Excel file: %w", err)
	}

	// Record processed issue in state (only after successful save)
	processedIssue := ProcessedIssue{
		Volume:        journalInfo.Volume,
		Issue:         journalInfo.Issue,
		ArticleCount:  len(articlesNormalized),
		StartNumber:   startNum,
		EndNumber:     endNum,
		Pubdate:       journalInfo.Pubdate,
		ProcessedDate: time.Now(),
	}

	if err := stateManager.RecordIssue(state, processedIssue); err != nil {
		return fmt.Errorf("failed to update state: %w", err)
	}

	fmt.Printf("\n✓ State updated: articles numbered %d-%d\n", startNum, endNum)
	fmt.Printf("✓ Excel file saved: %s\n", outputPath)

	return nil
}
