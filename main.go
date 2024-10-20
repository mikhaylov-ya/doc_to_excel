// Воркфлоу:
// 1. Прочитать 4 эксель-файла с данными журналов
// 2. Прочитать и распарсить док-файл номера журнала
// 3. Записать новые данные в эти файлы

// Сущности

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

type Reference struct {
	authors []string
	year    int
	title   string
	meta    string
	doi     string
}

type Article struct {
	title        string
	abstract     string
	pages        string
	keywords     string
	authors      string
	affiliations string
	references   []string
	doi          string
}

func readOrDefault(arr []string, index int, defaultValue string) string {
	if index >= 0 && index < len(arr) {
		return arr[index]
	}
	return defaultValue
}

func deleteSubstring(s string) string {
	return ""
}

func findMatchingString(slice []string, regex *regexp.Regexp) (string, bool) {
	for _, str := range slice {
		if regex.MatchString(str) {
			return str, true
		}
	}
	return "", false
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
	articleSepRegex := regexp.MustCompile(`\n\s*\n`)
	yearRegex := regexp.MustCompile(`\d\d\d\d`)
	numsRegex := regexp.MustCompile(`[a-z.]\d`)
	pagesRegex := regexp.MustCompile(`(\d+)[–-—](\d+)`)
	abstractRegex := regexp.MustCompile(`(?i)abstract[.:]`)
	kwRegex := regexp.MustCompile(`(?i)key\s?words[.:]`)
	doiRegex := regexp.MustCompile(`(?i)\bdoi(?:\s|\.|:)\s?(\d[^\n]*)`)
	authSuffxRegex := regexp.MustCompile(`\d+(,)?(\*)?`)
	mailSeps := [4]string{"E-mail", "Email", "email", "e-mail"}

	arr := articleSepRegex.Split(res.Body, -1)
	fmt.Printf("Articles found: %d\n", len(arr))
	articlesNormalized := make([]Article, len(arr))
	for artIndex, art := range arr {
		normArt := Article{}
		data := strings.Split(art, "<<<")
		artMeta, artReferences := data[0], readOrDefault(data, 1, "")
		referencesArr := strings.Split(artReferences, "\n")
		normArt.references = referencesArr
		if len(abstractRegex.Split(artMeta, 2)) < 2 {
			printError(artIndex+1, fmt.Sprintf("Can't get Abstract from article data: %s", artMeta))
		}
		abstractAndKW := kwRegex.Split(abstractRegex.Split(artMeta, 2)[1], 2)
		artAbstract, artKW :=
			abstractRegex.ReplaceAllStringFunc(abstractAndKW[0], deleteSubstring),
			kwRegex.ReplaceAllStringFunc(abstractAndKW[1], deleteSubstring)
		normArt.abstract = artAbstract
		normArt.keywords = artKW

		artStrings := strings.Split(artMeta, "\n")
		doi := strings.TrimSpace(strings.TrimPrefix(doiRegex.FindString(artMeta), "doi"))
		normArt.doi = doi
		// refactor on loop
		authorsTitle := artStrings[0]
		splittedAuthorsTitleAndMeta := yearRegex.Split(authorsTitle, 2)
		if len(splittedAuthorsTitleAndMeta) < 2 {
			printError(artIndex+1, fmt.Sprintf("Something went wrong when splitting author, title and meta by year: %s", authorsTitle))
		}

		authorsRaw, titleAndMeta := splittedAuthorsTitleAndMeta[0], splittedAuthorsTitleAndMeta[1]
		splittedTitleMeta := strings.Split(titleAndMeta, "//")
		// If there's no "//" in the string, split by "/"
		if len(splittedTitleMeta) == 1 {
			splittedTitleMeta = strings.Split(titleAndMeta, "/")
		}

		title, numberMeta := splittedTitleMeta[0], splittedTitleMeta[1]
		normArt.title = strings.TrimPrefix(title, ".")
		normArt.pages = pagesRegex.FindString(numberMeta)
		// (start) ----- AUTHORS BLOCK -------
		authorsNormalized := []string{}
		for _, auth := range strings.Split(authorsRaw, ", ") {
			authorsNormalized = append(authorsNormalized, authSuffxRegex.ReplaceAllStringFunc(auth, deleteSubstring))
		}
		normArt.authors = strings.Join(authorsNormalized, ", ")
		// (end) ----- AUTHORS BLOCK -------

		// (start) ----- AFFILIATIONS BLOCK -------
		authorAffilNums := numsRegex.FindAllString(authorsRaw, -1)
		affilations := make([]string, len(authorsNormalized))
		// Fill affiliations with same value if not enumerated
		if len(authorAffilNums) == 0 {
			for j := range affilations {
				affilations[j] = strings.TrimPrefix(artStrings[1], "1")
			}
		} else {
			affiliationsNumerated := make([]string, len(authorAffilNums))
			copy(affiliationsNumerated, artStrings[1:])

			for i, affilNum := range authorAffilNums {
				normNum := string([]rune(affilNum)[1])
				idx := slices.IndexFunc(affiliationsNumerated, func(s string) bool { return strings.HasPrefix(s, normNum) })
				if idx != -1 {
					affilations[i] = strings.TrimPrefix(affiliationsNumerated[idx], normNum)
				} else {
					printError(artIndex+1, fmt.Sprintf("Affilation not found: %s", normNum))
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
		f.SetCellValue("Sheet1", pagesCell, art.pages)
		f.SetCellValue("Sheet1", authorsCell, art.authors)
		f.SetCellValue("Sheet1", affiliationsCell, art.affiliations)
		f.SetCellValue("Sheet1", titleCell, art.title)
		f.SetCellValue("Sheet1", keywordCell, art.keywords)
		f.SetCellValue("Sheet1", abstractCell, art.abstract)
		f.SetCellValue("Sheet1", numCell, artNumStr)
		f.SetCellValue("Sheet1", doiCell, art.doi)
		fmt.Printf("Article %d\n", artI+1)

		fmt.Printf("Doi found: %s\n", art.doi)

		for artRefI, ref := range art.references {
			refI += 1
			if artRefI == len(art.references)-1 {
				ref = strings.TrimSuffix(ref, ">>>")
			}
			f.SetCellValue("References", fmt.Sprintf("A%s", strconv.Itoa(refI)), ref)
			f.SetCellValue("References", fmt.Sprintf("B%s", strconv.Itoa(refI)), art.doi)
		}
	}
	// Save spreadsheet by the given path.
	if err := f.SaveAs("Book1.xlsx"); err != nil {
		fmt.Println(err)
	}
}
