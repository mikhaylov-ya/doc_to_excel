package main

import (
	"regexp"
	"strings"
)

// year regex: 4 digits, optional letter suffix (2020a), optional span (1921-1922)
// we will store only the first 4 digits as the publication year
var yearRe = regexp.MustCompile(`\b(\d{4})([a-z])?(?:\s*[-–—]\s*\d{4})?\b`)

// detect likely author block (initials like "Z.M." or commas between names)
var initialRe = regexp.MustCompile(`[\p{L}]\.`) // letter followed by dot (unicode aware)

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
	"internet resource:", "internet resource",
	"accessed on", "accessed", "accessed:", "retrieved", "visited",
	"proceedings of", "proceedings", "journal", "transactions", "bulletin", "annals",
	"vol.", "http://", "https://",
	"london,",
	"berlin:", "moscow:", "leningrad:", "novosibirsk:", "irkutsk:", "cham:", "london:",
	"saint petersburg:", "st. petersburg:", "new york:", "kiev:", "kyiv:", "vladivostok:",
	"stuttgart:",
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
	low := strings.ToLower(s)
	min := -1
	for _, m := range markers {
		idx := strings.Index(low, strings.ToLower(m))
		if idx >= 0 {
			if min == -1 || idx < min {
				min = idx
			}
		}
	}
	return min
}

// splitTitleMeta takes text after the year (already cleaned of leading ". ")
// and returns title and raw meta (no deep parsing)
func splitTitleMeta(after string) (string, string) {
	s := strings.TrimSpace(after)
	if s == "" {
		return "", ""
	}

	// 1) explicit "//" split is now handled in step 3 based on period detection

	// 2) bracketed title: start with '[' ... ']' maybe followed by a dot
	if strings.HasPrefix(s, "[") {
		// find closing bracket
		if end := strings.Index(s, "]"); end != -1 {
			titleEnd := end + 1
			// include trailing dot if present (so title ends with "].")
			if titleEnd < len(s) && s[titleEnd] == '.' {
				titleEnd++
			}
			title := strings.TrimSpace(s[:titleEnd])
			meta := strings.TrimSpace(s[titleEnd:])

			// Check if meta starts with // separator and remove it
			if strings.HasPrefix(meta, "//") {
				meta = strings.TrimSpace(meta[2:])
			}

			return title, meta
		}
	}

	// 3) look for publication markers (http, Vol., Proceedings, city:, etc.)
	// First, check for // separator as the clearest marker (but not in URLs)
	if idx := strings.Index(s, "//"); idx != -1 {
		// Check if this is part of a URL (http:// or https://)
		if idx > 0 && (s[idx-1] == ':' || (idx > 4 && s[idx-5:idx-1] == "http")) {
			// This is a URL, skip this split and continue to other logic
		} else {
			title := strings.TrimSpace(s[:idx])
			meta := strings.TrimSpace(s[idx+2:]) // Skip the "//" and any spaces
			// Remove any leading "//" from meta if it exists
			if strings.HasPrefix(meta, "//") {
				meta = strings.TrimSpace(meta[2:])
			}
			return title, meta
		}
	}

	// If no // separator, check if there's a period in the title to determine search strategy
	firstPeriodIdx := strings.Index(s, ".")

	if firstPeriodIdx == -1 {
		// No period found - search for markers in the entire text
		if idx := findFirstOfMarkers(s, publicationMarkers); idx != -1 {
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

	// 4) fallback: look for a period followed by CAPITAL letter or '[' or '(' (heuristic)
	found := -1
	maxScan := len(s)
	if maxScan > 400 {
		maxScan = 400
	}
	for i := 0; i < maxScan; i++ {
		if s[i] == '.' {
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
		return title, meta
	}

	// last resort: everything is title
	return s, ""
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
