package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// StartingPoint represents where end-to-end numbering begins
type StartingPoint struct {
	Volume  int `yaml:"volume"`
	Issue   int `yaml:"issue"`
	Counter int `yaml:"counter"`
}

// ProcessedIssue represents a journal issue that has been processed
type ProcessedIssue struct {
	Volume        string    `yaml:"volume"`
	Issue         string    `yaml:"issue"`
	ArticleCount  int       `yaml:"article_count"`
	StartNumber   int       `yaml:"start_number"`
	EndNumber     int       `yaml:"end_number"`
	Pubdate       string    `yaml:"pubdate"`
	ProcessedDate time.Time `yaml:"processed_date"`
}

// JournalState represents the state file for a journal
type JournalState struct {
	JournalCode      string           `yaml:"journal_code"`
	JournalName      string           `yaml:"journal_name"`
	StartingPoint    StartingPoint    `yaml:"starting_point"`
	CurrentCounter   int              `yaml:"current_counter"`
	ProcessedIssues  []ProcessedIssue `yaml:"processed_issues"`
	MaxHistory       int              `yaml:"max_history"`
	stateFilePath    string           // Internal field, not serialized
}

// DuplicateAction represents the user's choice when a duplicate is found
type DuplicateAction int

const (
	SkipProcessing DuplicateAction = iota
	ReprocessSameNumbers
	ReprocessNewNumbers
	Abort
)

// StateManager handles all state file operations
type StateManager struct {
	stateDir string
}

// NewStateManager creates a new state manager
func NewStateManager() *StateManager {
	return &StateManager{
		stateDir: "state",
	}
}

// LoadState loads the state file for a journal
func (sm *StateManager) LoadState(journalCode string) (*JournalState, error) {
	stateFile := filepath.Join(sm.stateDir, fmt.Sprintf("%s_state.yaml", journalCode))

	data, err := os.ReadFile(stateFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var state JournalState
	if err := yaml.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse state file: %w", err)
	}

	state.stateFilePath = stateFile
	return &state, nil
}

// SaveState saves the state file for a journal (with atomic write and backup)
func (sm *StateManager) SaveState(state *JournalState) error {
	// Create backup of existing file
	if _, err := os.Stat(state.stateFilePath); err == nil {
		backupPath := state.stateFilePath + ".bak"
		if err := os.Rename(state.stateFilePath, backupPath); err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}
	}

	// Marshal to YAML
	data, err := yaml.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Write to temp file first (atomic write)
	tempFile := state.stateFilePath + ".tmp"
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	// Rename temp to actual file (atomic on most systems)
	if err := os.Rename(tempFile, state.stateFilePath); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// IsIssueProcessed checks if an issue has already been processed
func (sm *StateManager) IsIssueProcessed(state *JournalState, volume, issue string) (*ProcessedIssue, bool) {
	for i := range state.ProcessedIssues {
		if state.ProcessedIssues[i].Volume == volume && state.ProcessedIssues[i].Issue == issue {
			return &state.ProcessedIssues[i], true
		}
	}
	return nil, false
}

// IsConfigured checks if the state file has been configured with a starting point
func (sm *StateManager) IsConfigured(state *JournalState) bool {
	return state.StartingPoint.Counter > 0
}

// PromptForStartingPoint prompts the user to configure the starting point
func (sm *StateManager) PromptForStartingPoint(state *JournalState, volume, issue string) error {
	fmt.Println("\n⚠️  State file not initialized for journal", state.JournalCode)
	fmt.Println("Please configure the starting point for end-to-end numbering:")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)

	// Default to current volume/issue
	fmt.Printf("Starting Volume [%s]: ", volume)
	volInput, _ := reader.ReadString('\n')
	volInput = strings.TrimSpace(volInput)
	if volInput == "" {
		volInput = volume
	}
	startVol, err := strconv.Atoi(volInput)
	if err != nil {
		return fmt.Errorf("invalid volume number: %w", err)
	}

	fmt.Printf("Starting Issue [%s]: ", issue)
	issueInput, _ := reader.ReadString('\n')
	issueInput = strings.TrimSpace(issueInput)
	if issueInput == "" {
		issueInput = issue
	}
	startIssue, err := strconv.Atoi(issueInput)
	if err != nil {
		return fmt.Errorf("invalid issue number: %w", err)
	}

	fmt.Print("Starting Counter (first article number): ")
	counterInput, _ := reader.ReadString('\n')
	counterInput = strings.TrimSpace(counterInput)
	startCounter, err := strconv.Atoi(counterInput)
	if err != nil || startCounter <= 0 {
		return fmt.Errorf("invalid starting counter (must be > 0)")
	}

	state.StartingPoint.Volume = startVol
	state.StartingPoint.Issue = startIssue
	state.StartingPoint.Counter = startCounter
	state.CurrentCounter = startCounter

	if err := sm.SaveState(state); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	fmt.Printf("\n✓ State file configured: %s_state.yaml\n", state.JournalCode)
	fmt.Println("Continuing with processing...")
	fmt.Println()

	return nil
}

// AllocateNumbers allocates article numbers for a new issue
func (sm *StateManager) AllocateNumbers(state *JournalState, articleCount int) (startNum, endNum int) {
	startNum = state.CurrentCounter
	endNum = state.CurrentCounter + articleCount - 1
	return startNum, endNum
}

// RecordIssue records a processed issue and updates the state
func (sm *StateManager) RecordIssue(state *JournalState, issue ProcessedIssue) error {
	// Validation: ensure counter doesn't go backwards
	if issue.EndNumber < state.CurrentCounter {
		return fmt.Errorf("counter going backwards: end number %d < current counter %d",
			issue.EndNumber, state.CurrentCounter)
	}

	// Add to processed issues
	state.ProcessedIssues = append(state.ProcessedIssues, issue)

	// Update current counter
	state.CurrentCounter = issue.EndNumber + 1

	// Trim history to max_history
	if len(state.ProcessedIssues) > state.MaxHistory {
		// Keep only the most recent issues
		state.ProcessedIssues = state.ProcessedIssues[len(state.ProcessedIssues)-state.MaxHistory:]
	}

	// Save state
	return sm.SaveState(state)
}

// RemoveIssue removes an issue from the history (for manual corrections)
func (sm *StateManager) RemoveIssue(state *JournalState, volume, issue string) error {
	newProcessedIssues := make([]ProcessedIssue, 0)
	found := false

	for _, pi := range state.ProcessedIssues {
		if pi.Volume == volume && pi.Issue == issue {
			found = true
			continue // Skip this issue
		}
		newProcessedIssues = append(newProcessedIssues, pi)
	}

	if !found {
		return fmt.Errorf("issue %s vol %s issue %s not found in history", state.JournalCode, volume, issue)
	}

	state.ProcessedIssues = newProcessedIssues
	return sm.SaveState(state)
}

// HandleDuplicateIssue handles when a duplicate issue is detected
// Default behavior: reprocess with existing numbers
func (sm *StateManager) HandleDuplicateIssue(existingIssue ProcessedIssue, journalCode string) DuplicateAction {
	fmt.Println("\n⚠️  Issue already processed!")
	fmt.Printf("Journal: %s, Volume: %s, Issue: %s\n", journalCode, existingIssue.Volume, existingIssue.Issue)
	fmt.Printf("Previously processed on: %s\n", existingIssue.ProcessedDate.Format("2006-01-02 15:04:05"))
	fmt.Printf("Articles: %d (numbers %d-%d)\n", existingIssue.ArticleCount, existingIssue.StartNumber, existingIssue.EndNumber)
	fmt.Println("→ Reprocessing with existing numbers (default behavior)")
	fmt.Println()

	return ReprocessSameNumbers
}

// ExtractJournalCodeFromDOI extracts the journal code from a DOI
func ExtractJournalCodeFromDOI(doi string) (string, error) {
	// Map DOI prefixes to journal codes
	doiToJournal := map[string]string{
		"euroasentj":  "EEJ",
		"rusentj":     "REJ",
		"invertzool":  "IZ",
		"arthsel":     "AS",
	}

	for prefix, code := range doiToJournal {
		if strings.Contains(doi, prefix) {
			return code, nil
		}
	}

	return "", fmt.Errorf("unknown journal in DOI: %s", doi)
}
