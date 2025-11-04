package main

import (
	"regexp"
	"strings"
)

/*
REFERENCE PARSING ENHANCEMENTS - Implementation Notes

This parser extracts Authors, Year, Title, and Publication Metadata from bibliographic references.

ENHANCEMENT SUMMARY (based on analysis of desired output patterns):

1. ABBREVIATION DETECTION
   - Skips periods in common abbreviations (St., Vol., No., Pt., etc.)
   - Prevents false splits on "St. Petersburg", "Vol. 5" appearing in titles
   - Function: isAbbreviation()

2. MINIMUM POSITION CHECKS
   - Markers not searched in first 20 characters (avoids premature splits)
   - Prevents finding "London" or "Vol." too early when still in title
   - Function: findFirstOfMarkersFrom() with minMarkerPosition

3. REFERENCE TYPE DETECTION
   - Identifies: Article, Book, Chapter, Online, Other
   - Applies type-specific parsing strategies
   - Function: detectReferenceType()

4. IMPROVED EDITOR PATTERN DETECTION
   - Detects chapter references: "Title // Editor (Ed.): Book"
   - Handles variants: (Ed.), (Eds), (Ed):, (Eds):
   - Stronger regex pattern: editorPatternRe

5. CONTEXTUAL CITY:PUBLISHER MATCHING
   - Looks for "City: Publisher" pattern (both capitalized)
   - Requires position > 30 chars (not in title)
   - Function: findPublisherPattern()

6. VALIDATION PASS
   - Checks title/meta length ratio (title >= 20% of total)
   - Rejects splits where meta starts with lowercase (mid-sentence)
   - Function: validateSplit()

REFERENCE STRUCTURE TYPES HANDLED:

1. Journal Articles:
   Format: Authors Year. Title // Journal. Vol.X. No.Y. P.XXX-XXX.
   Example: Abramov S.A. 2014. Ecological differentiation... // Biology Bulletin. Vol.41.

2. Books:
   Format: Authors Year. [Title]. City: Publisher. XXX p.
   Example: Bigon M. 1989. [Ecology]. Moscow: Mir. 667 p.

3. Book Chapters:
   Format: Authors Year. Title // Editor (Ed.): Book. City: Publisher. P.XX-XX.
   Example: Nash T.H. 1991. Lichens... // Hutzinger O. (Ed.): Handbook...

4. Online Resources:
   Format: Title. Year. Available from: URL Accessed Date
   Example: GBIF.org 2024. GBIF Occurrence. Available from: https://...

5. Institutional (No authors):
   Format: Title. Year. Edition. City: Publisher.
   Example: A manual of acarology. 2009. 3rd edition. Texas: Press.

KEY PARAMETERS (tunable):
- minTitleLength = 15        (chars before allowing period-split)
- minMarkerPosition = 20     (don't search markers in first 20 chars)
- maxTitleScanLength = 400   (max chars to scan for title end)
- titleMetaLengthRatio = 0.2 (title should be >= 20% of total)
*/

// year regex: 4 digits, optional letter suffix (2020a), optional span (1921-1922)
// we will store only the first 4 digits as the publication year
var yearRe = regexp.MustCompile(`\b(\d{4})([a-z])?(?:\s*[-–—]\s*\d{4})?\b`)

// detect likely author block (initials like "Z.M." or commas between names)
var initialRe = regexp.MustCompile(`[\p{L}]\.`) // letter followed by dot (unicode aware)

// editor patterns - stronger signal than general markers, checked early
// Matches: "// Author (ed.):" or "// Author (eds.)." or "// Author (Ed):"
var editorPatternRe = regexp.MustCompile(`(?i)//\s*[^/]+\s*\(\s*e(?:d|ds)\.?\s*\)\s*[:;.]`)

// Common abbreviations that should NOT trigger title/meta splits
var abbreviations = []string{
	"St", "Vol", "No", "Nos", "Pt", "P", "S", "Bd", "Ed", "Eds",
	"T", "Ch", "Art", "Ph", "Dr", "Mr", "Mrs", "Ms",
}

// Tuning parameters for parsing
const (
	minTitleLength       = 15  // Minimum characters before allowing period-based split
	minMarkerPosition    = 20  // Don't find markers in first 20 chars (avoid false positives)
	maxTitleScanLength   = 400 // Maximum characters to scan for title end
	titleMetaLengthRatio = 0.2 // Title should be at least 20% of total length
)

// markers used to split title/meta heuristically
// Order matters: more specific markers should come first

// 1. Intelligent Marker Detection
// Second sentence search: When a period is found in the title,
// markers are only searched starting from the second sentence, preventing false positives from
// city names or other words that happen to match markers in the first sentence.

// 2. Prioritized "//" Separator
// Clear marker precedence: The "//" separator is now checked first
// as the clearest marker for splitting title and meta.
// URL protection: Added logic to detect when "//" is part of a URL (http:// or https://) and skip it in those cases

// 3. Improved Marker Ordering
// Specific markers first: Reordered the publication markers to put more specific ones
// (like "available from") before generic ones (like "https://").

var publicationMarkers = []string{
	"available online", "available at", "available from", "available on", "available:",
	"online at", "online:", "internet resource:", "internet resource",
	"accessed on", "accessed", "accessed:", "retrieved", "visited",
	"proceedings of", "proceedings", "journal", "transactions", "bulletin", "annals",
	"vol.", "http://", "https://",
	// Note: City names (e.g., "Moscow:", "Berlin:") are handled by findPublisherPattern()
	// which detects the pattern "CapitalizedCity: CapitalizedPublisher" contextually
}

func pickYearIndex(ref string) (int, int, string) {
	loc := yearRe.FindStringIndex(ref)
	if loc == nil {
		return -1, -1, ""
	}
	match := ref[loc[0]:loc[1]]
	// keep only the first 4 digits as the year string
	if len(match) >= 4 {
		return loc[0], loc[1], match[:4]
	}
	return loc[0], loc[1], match
}

func isLikelyAuthorBlock(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	// initials (e.g., "Z.M.") or comma-separated names are strong signals of authors
	if initialRe.MatchString(s) {
		return true
	}
	if strings.Contains(s, ",") {
		return true
	}
	// otherwise treat as not author (institution/title)
	return false
}

func findFirstOfMarkers(s string, markers []string) int {
	return findFirstOfMarkersFrom(s, markers, 0)
}

// findFirstOfMarkersFrom finds the first occurrence of any marker, starting from minPos
// This prevents finding markers too early (e.g., in the first few words of the title)
func findFirstOfMarkersFrom(s string, markers []string, minPos int) int {
	if minPos < 0 {
		minPos = 0
	}
	if minPos >= len(s) {
		return -1
	}

	low := strings.ToLower(s)
	min := -1
	for _, m := range markers {
		// Search only from minPos onwards
		searchSpace := low[minPos:]
		idx := strings.Index(searchSpace, strings.ToLower(m))
		if idx >= 0 {
			actualIdx := minPos + idx
			if min == -1 || actualIdx < min {
				min = actualIdx
			}
		}
	}
	return min
}

// isAbbreviation checks if a period at the given position is part of a known abbreviation
func isAbbreviation(s string, periodPos int) bool {
	if periodPos < 0 || periodPos >= len(s) {
		return false
	}

	// Check each known abbreviation
	for _, abbr := range abbreviations {
		abbrLen := len(abbr)
		start := periodPos - abbrLen

		// Make sure we have enough characters before the period
		if start >= 0 && start < len(s) {
			// Extract the text before the period
			beforePeriod := s[start:periodPos]
			// Case-insensitive comparison
			if strings.EqualFold(beforePeriod, abbr) {
				return true
			}
		}
	}
	return false
}

// ReferenceType indicates the type of reference for better parsing strategy
type ReferenceType int

const (
	TypeArticle ReferenceType = iota
	TypeBook
	TypeChapter
	TypeOnline
	TypeOther
)

// detectReferenceType analyzes the text after year to determine reference type
func detectReferenceType(afterYear string) ReferenceType {
	lower := strings.ToLower(afterYear)

	// Online resources (check first as they're distinct)
	if strings.Contains(lower, "available from:") ||
		strings.Contains(lower, "available at:") ||
		strings.Contains(lower, "available online:") ||
		strings.Contains(lower, "available on:") ||
		strings.Contains(lower, "available:") ||
		strings.Contains(lower, "internet resource") ||
		strings.Contains(lower, "online at") ||
		strings.Contains(lower, "accessed on:") ||
		strings.Contains(lower, "accessed") && strings.Contains(lower, "http") {
		return TypeOnline
	}

	// Chapter - has editor notation
	if strings.Contains(lower, "(ed.)") ||
		strings.Contains(lower, "(eds)") ||
		strings.Contains(lower, "(eds.)") ||
		strings.Contains(lower, "(ed):") ||
		strings.Contains(lower, "(hrsg)") ||
		strings.Contains(lower, "(hrsg.)") ||
		strings.Contains(lower, "(eds):") {
		return TypeChapter
	}

	// Article - has // separator (not in URL)
	if idx := strings.Index(afterYear, "//"); idx != -1 {
		// Make sure it's not a URL
		if idx == 0 || (idx > 0 && afterYear[idx-1] != ':') {
			return TypeArticle
		}
	}

	// Book - has city:publisher pattern
	if findPublisherPattern(afterYear) != -1 {
		return TypeBook
	}

	return TypeOther
}

// findPublisherPattern looks for "City: Publisher" or "City, State: Publisher"
// Returns the position where this pattern starts, or -1 if not found
func findPublisherPattern(s string) int {
	// Match: Word(s) starting with capital : Word starting with capital/uppercase
	// e.g., "Moscow: Nauka", "New York: Academic Press", "Cambridge: MIT Press"
	// Publisher name can be all caps (MIT) or capitalized (Press)
	// Updated to handle multi-word city names (e.g., "New York", "San Francisco")
	cityPubRe := regexp.MustCompile(`\b([A-Z][a-zA-Z]+(?:\s+[A-Z][a-zA-Z]+)*(?:,\s+[A-Z][a-zA-Z]+)?)\s*:\s*[A-Z]`)

	if match := cityPubRe.FindStringIndex(s); match != nil {
		beforeMatch := s[:match[0]]

		// Check if there's a "//" before this position (already in meta section)
		if strings.Contains(beforeMatch, "//") {
			return -1
		}

		// Require either:
		// 1. Position > 30 (clearly past title), OR
		// 2. Position >= 15 AND there's a period before it (end of sentence)
		if match[0] > 30 {
			return match[0]
		} else if match[0] >= 15 && strings.Contains(beforeMatch, ".") {
			// There's a period before the city:publisher pattern
			// This likely indicates end of title
			return match[0]
		}
	}
	return -1
}

// splitTitleMeta takes text after the year (already cleaned of leading ". ")
// and returns title and raw meta (no deep parsing)
func splitTitleMeta(after string) (string, string) {
	s := strings.TrimSpace(after)
	if s == "" {
		return "", ""
	}

	// Detect reference type for smarter parsing
	refType := detectReferenceType(s)

	// 1) bracketed title: start with '[' ... ']' maybe followed by a dot
	if strings.HasPrefix(s, "[") {
		// find closing bracket
		if end := strings.Index(s, "]"); end != -1 {
			titleEnd := end + 1
			// include trailing dot if present (so title ends with "].")
			if titleEnd < len(s) && s[titleEnd] == '.' {
				titleEnd++
			}
			title := strings.TrimSpace(s[:titleEnd])
			meta := s[titleEnd:]

			// Check if meta starts with // separator and remove it
			if strings.HasPrefix(strings.TrimSpace(meta), "//") {
				meta = strings.TrimSpace(meta)
				meta = meta[2:] // Skip "//"
			}

			// Remove any leading punctuation and whitespace from meta
			meta = strings.TrimPrefix(meta, ". ")

			return title, meta
		}
	}

	// 2) Check for editor pattern (chapters) - strong signal
	// Example: "Title // Author (Ed.): Book. Publisher" or "Title // Author (eds.). Book"
	if refType == TypeChapter {
		if match := editorPatternRe.FindStringIndex(s); match != nil {
			// Split at the "//" before the editor
			title := strings.TrimSpace(s[:match[0]])
			// Skip to the end of the matched pattern (after "Author (ed.):" or "Author (eds.).")
			meta := s[match[1]:]
			// Remove any leading punctuation and whitespace from meta
			meta = strings.TrimLeft(meta, " \t./")
			return title, meta
		}
	}

	// 3) Check for // separator as the clearest marker (but not in URLs)
	if idx := strings.Index(s, "//"); idx != -1 {
		// Check if this is part of a URL (http:// or https://)
		isURL := false
		if idx > 0 && s[idx-1] == ':' {
			isURL = true
		} else if idx >= 5 && strings.HasPrefix(strings.ToLower(s[idx-5:]), "http") {
			isURL = true
		}

		if !isURL {
			title := strings.TrimSpace(s[:idx])
			meta := s[idx+2:] // Skip the "//"
			// Remove any leading punctuation and whitespace from meta
			meta = strings.TrimPrefix(meta, ". ")
			// Also remove any additional leading "//" from meta if it exists
			if strings.HasPrefix(meta, "//") {
				meta = strings.TrimPrefix(meta[2:], ". ")
			}
			return title, meta
		}
	}

	// 4) Check for publisher pattern (City: Publisher) - works for any type
	// This handles books and other references with city:publisher format
	if pubIdx := findPublisherPattern(s); pubIdx != -1 {
		title := strings.TrimSpace(s[:pubIdx])
		meta := strings.TrimSpace(s[pubIdx:])
		return title, meta
	}

	// 5) Search for publication markers with improved logic
	firstPeriodIdx := strings.Index(s, ".")

	if firstPeriodIdx == -1 {
		// No period found - search for markers in the entire text, but not too early
		if idx := findFirstOfMarkersFrom(s, publicationMarkers, minMarkerPosition); idx != -1 {
			title := strings.TrimSpace(s[:idx])
			meta := strings.TrimSpace(s[idx:])
			return title, meta
		}
	} else {
		// Period found - only search for markers starting from the second sentence
		searchStart := firstPeriodIdx + 1
		// Skip spaces after the period
		for searchStart < len(s) && (s[searchStart] == ' ' || s[searchStart] == '\t') {
			searchStart++
		}

		if searchStart < len(s) {
			// Search for markers only in the text after the first sentence
			textAfterFirstSentence := s[searchStart:]
			if idx := findFirstOfMarkers(textAfterFirstSentence, publicationMarkers); idx != -1 {
				// Adjust index to be relative to the original string
				actualIdx := searchStart + idx
				title := strings.TrimSpace(s[:actualIdx])
				meta := strings.TrimSpace(s[actualIdx:])
				return title, meta
			}
		}
	}

	// 6) Fallback: look for a period followed by CAPITAL letter or '[' or '('
	// BUT: skip periods that are part of abbreviations and respect minimum title length
	found := -1
	maxScan := len(s)
	if maxScan > maxTitleScanLength {
		maxScan = maxTitleScanLength
	}

	for i := 0; i < maxScan; i++ {
		if s[i] == '.' {
			// Skip if this is an abbreviation
			if isAbbreviation(s, i) {
				continue
			}

			// Skip if title would be too short
			if i < minTitleLength {
				continue
			}

			j := i + 1
			for j < len(s) && (s[j] == ' ' || s[j] == '\t') {
				j++
			}
			if j < len(s) {
				ch := s[j]
				if (ch >= 'A' && ch <= 'Z') || ch == '[' || ch == '(' {
					found = i
					break
				}
			}
		}
	}

	if found != -1 {
		title := strings.TrimSpace(s[:found+1])
		meta := strings.TrimSpace(s[found+1:])

		// Validate the split makes sense
		if validateSplit(title, meta) {
			return title, meta
		}
	}

	// last resort: everything is title
	return s, ""
}

// validateSplit checks if the title/meta split is reasonable
func validateSplit(title, meta string) bool {
	// Title shouldn't be too short relative to total length
	totalLen := len(title) + len(meta)
	if totalLen > 0 {
		titleRatio := float64(len(title)) / float64(totalLen)
		if titleRatio < titleMetaLengthRatio {
			return false
		}
	}

	// Meta shouldn't start with lowercase (indicates split mid-sentence)
	if len(meta) > 0 {
		firstChar := rune(meta[0])
		if firstChar >= 'a' && firstChar <= 'z' {
			return false
		}
	}

	return true
}

// parseReference: main orchestration; returns authors, year, title, meta(raw)
func parseReference(ref string) (string, string, string, string) {
	ref = strings.TrimSpace(ref)
	// normalize whitespace
	ref = regexp.MustCompile(`\s+`).ReplaceAllString(ref, " ")

	startIdx, endIdx, year := pickYearIndex(ref)
	if startIdx == -1 {
		// no year found — keep whole ref as title
		return "Not mentioned", "Not mentioned", ref, ""
	}

	beforeYear := strings.TrimSpace(ref[:startIdx])
	afterYear := ref[endIdx:] // keep punctuation immediately after year

	// remove leading spaces
	afterYear = strings.TrimLeft(afterYear, " \t")
	// if there's a single dot right after the year (typical "1989. Title..."), remove it + subsequent spaces
	if strings.HasPrefix(afterYear, ".") {
		afterYear = strings.TrimLeft(afterYear[1:], " \t")
	}

	// decide whether beforeYear is authors or an institution/title
	var authors string
	if beforeYear == "" || !isLikelyAuthorBlock(beforeYear) {
		authors = "Not mentioned"
	} else {
		// preserve punctuation (do not strip final dot after initial)
		authors = strings.TrimSpace(beforeYear)
	}

	// now split title/meta from the remainder
	title, meta := splitTitleMeta(afterYear)

	// if authors were an institution, move the beforeYear phrase into meta (unless meta already contains it)
	if authors == "Not mentioned" && beforeYear != "" {
		// Normalize candidate (without trailing dot for comparison)
		cand := strings.TrimSpace(strings.TrimRight(beforeYear, "."))
		if cand != "" {
			if !strings.Contains(strings.ToLower(meta), strings.ToLower(cand)) {
				meta = strings.TrimSpace(beforeYear + " " + meta)
			}
		}
	}

	// strip leading editor block from meta
	editorRe := regexp.MustCompile(`(?i)\(\s*e(ds?|d)\.?\s*\)\s*[:;]?\s*`)
	parts := editorRe.Split(meta, 2)
	if len(parts) == 2 {
		meta = strings.TrimSpace(parts[1])
	}

	return authors, year, title, meta
}
