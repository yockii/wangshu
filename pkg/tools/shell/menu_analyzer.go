package shell

import (
	"regexp"
	"strings"
)

// PromptPattern represents a pattern that indicates the command is waiting for input
type PromptPattern struct {
	Pattern   *regexp.Regexp
	Type      InputWaitType
	Priority  int // Higher priority patterns are checked first
}

// InputWaitType represents the type of input waiting detected
type InputWaitType string

const (
	WaitTypeContentPattern InputWaitType = "content_pattern" // Detected by content pattern
	WaitTypeOutputSilence  InputWaitType = "output_silence"  // Detected by output silence
	WaitTypeProcessState   InputWaitType = "process_state"   // Detected by process state
)

// Common prompt patterns for detecting input waiting
var PromptPatterns = []PromptPattern{
	// Yes/No confirmations - highest priority
	{
		Pattern:  regexp.MustCompile(`(?i)\(yes/no\)|\(y/n\)|\?[ \t]*$`),
		Type:     WaitTypeContentPattern,
		Priority: 100,
	},
	{
		Pattern:  regexp.MustCompile(`(?i)^(yes|no)[\?\.]?[\s]*$`),
		Type:     WaitTypeContentPattern,
		Priority: 90,
	},
	// Selection prompts
	{
		Pattern:  regexp.MustCompile(`(?i)(select|choose|pick)[\s]+(option|choice|from)[\s]*:`),
		Type:     WaitTypeContentPattern,
		Priority: 80,
	},
	// Numbered menu options - 1) Option A or 1. Option B
	{
		Pattern:  regexp.MustCompile(`^\s*\d+[\)\.]`),
		Type:     WaitTypeContentPattern,
		Priority: 70,
	},
	// Question mark at end
	{
		Pattern:  regexp.MustCompile(`\?[\s]*$`),
		Type:     WaitTypeContentPattern,
		Priority: 60,
	},
	// Common interactive prompts
	{
		Pattern:  regexp.MustCompile(`(?i)(enter|input|type)[\s]+(your\s+)?[\w\s]+:`),
		Type:     WaitTypeContentPattern,
		Priority: 50,
	},
	// Press Enter to continue
	{
		Pattern:  regexp.MustCompile(`(?i)press\s+(enter|return)\s+to\s+continue`),
		Type:     WaitTypeContentPattern,
		Priority: 40,
	},
}

// MenuOption represents a detected menu option
type MenuOption struct {
	Index       int    `json:"index"`
	Text        string `json:"text"`
	InputValue  string `json:"input_value"`  // The value to type to select this option
	Description string `json:"description,omitempty"`
}

// MenuAnalysis represents the analysis of a potential menu
type MenuAnalysis struct {
	IsMenu     bool         `json:"is_menu"`
	MenuType   MenuType     `json:"menu_type"`
	Options    []MenuOption `json:"options,omitempty"`
	Confidence float64      `json:"confidence"`
	Prompt     string       `json:"prompt,omitempty"`
}

// MenuAnalyzer analyzes terminal output to detect menus and prompts
type MenuAnalyzer struct {
	keySeq *TerminalKeySequence
}

// NewMenuAnalyzer creates a new menu analyzer
func NewMenuAnalyzer() *MenuAnalyzer {
	return &MenuAnalyzer{
		keySeq: NewTerminalKeySequence(),
	}
}

// DetectInputWaiting detects if the command is waiting for input based on output
func (ma *MenuAnalyzer) DetectInputWaiting(output string) (bool, InputWaitType, string) {
	// Check against content patterns
	for _, pp := range PromptPatterns {
		if pp.Pattern.MatchString(output) {
			// Find the matched portion
			matches := pp.Pattern.FindAllStringIndex(output, -1)
			if len(matches) > 0 {
				lastMatch := matches[len(matches)-1]
				if lastMatch[1] <= len(output) {
					matchedText := output[lastMatch[0]:lastMatch[1]]
					return true, pp.Type, matchedText
				}
			}
			return true, pp.Type, ""
		}
	}

	// Check if output ends with a common prompt pattern
	lines := strings.Split(output, "\n")
	if len(lines) > 0 {
		lastLine := strings.TrimSpace(lines[len(lines)-1])
		if lastLine != "" && !strings.HasSuffix(lastLine, ".") && !strings.HasSuffix(lastLine, "!") {
			// Last line looks like a prompt (not ending with sentence terminator)
			for _, pp := range PromptPatterns {
				if pp.Pattern.MatchString(lastLine) {
					return true, pp.Type, lastLine
				}
			}
		}
	}

	return false, WaitTypeContentPattern, ""
}

// AnalyzeMenu analyzes output to detect if it contains a menu
func (ma *MenuAnalyzer) AnalyzeMenu(output string) (*MenuAnalysis, error) {
	// Try to detect numbered menu first
	if options := ma.detectNumberedMenu(output); len(options) > 0 {
		return &MenuAnalysis{
			IsMenu:     true,
			MenuType:   MenuTypeNumbered,
			Options:    options,
			Confidence: ma.calculateNumberedMenuConfidence(output, options),
		}, nil
	}

	// Try to detect box menu (requires looking at lines more carefully)
	if ma.detectBoxMenu(output) {
		// Box menus need LLM analysis to extract options
		return &MenuAnalysis{
			IsMenu:     true,
			MenuType:   MenuTypeBox,
			Confidence: 0.7,
		}, nil
	}

	// Try to detect wizard prompts
	if ma.detectWizardPrompt(output) {
		return &MenuAnalysis{
			IsMenu:     true,
			MenuType:   MenuTypeWizard,
			Confidence: 0.6,
		}, nil
	}

	return &MenuAnalysis{IsMenu: false}, nil
}

// detectNumberedMenu detects numbered menu options like "1) Option A" or "1. Option B"
func (ma *MenuAnalyzer) detectNumberedMenu(output string) []MenuOption {
	lines := strings.Split(output, "\n")
	options := make([]MenuOption, 0)

	optionPattern := regexp.MustCompile(`^\s*(\d+)[\)\.]\s*(.+?)(?:\s+(.+))?$`)

	for _, line := range lines {
		matches := optionPattern.FindStringSubmatch(line)
		if matches != nil {
			index := 0 // Will be calculated from the actual number
			if num, err := regexp.Compile(`^\d+`); err == nil {
				numMatches := num.FindString(matches[1])
				if len(numMatches) > 0 {
					// Use 0-based index
					index = len(options)
				}
			}

			option := MenuOption{
				Index:      index,
				Text:       strings.TrimSpace(matches[2]),
				InputValue: matches[1], // Use the number as input
			}

			if len(matches) > 3 {
				option.Description = strings.TrimSpace(matches[3])
			}

			options = append(options, option)
		}
	}

	return options
}

// detectBoxMenu detects box-drawn menus with visual indicators
func (ma *MenuAnalyzer) detectBoxMenu(output string) bool {
	// Look for box drawing characters
	boxChars := []string{"│", "└", "┌", "┐", "┘", "─", "┼", "┤", "├", "┬", "┴"}
	hasBoxChars := false
	for _, char := range boxChars {
		if strings.Contains(output, char) {
			hasBoxChars = true
			break
		}
	}

	if !hasBoxChars {
		return false
	}

	// Look for visual indicators like ●, ○, →, or *
	indicatorPatterns := []string{"●", "○", "→", " * ", "\x1b["} // ESC indicates cursor position
	for _, pattern := range indicatorPatterns {
		if strings.Contains(output, pattern) {
			return true
		}
	}

	return false
}

// detectWizardPrompt detects multi-step wizard prompts
func (ma *MenuAnalyzer) detectWizardPrompt(output string) bool {
	wizardKeywords := []string{
		"step", "choose", "select", "configure", "setup",
		"which would you like", "what do you want",
	}

	lowerOutput := strings.ToLower(output)
	for _, keyword := range wizardKeywords {
		if strings.Contains(lowerOutput, keyword) {
			return true
		}
	}

	return false
}

// calculateNumberedMenuConfidence calculates confidence score for numbered menu detection
func (ma *MenuAnalyzer) calculateNumberedMenuConfidence(output string, options []MenuOption) float64 {
	confidence := 0.5

	// More options = higher confidence
	if len(options) >= 2 {
		confidence += 0.2
	}
	if len(options) >= 4 {
		confidence += 0.1
	}

	// Look for menu title
	lines := strings.Split(output, "\n")
	if len(lines) > 0 {
		firstLine := strings.ToLower(lines[0])
		titleKeywords := []string{"select", "choose", "option", "menu", "pick"}
		for _, keyword := range titleKeywords {
			if strings.Contains(firstLine, keyword) {
				confidence += 0.15
				break
			}
		}
	}

	// Check for sequential numbering
	if len(options) >= 2 {
		sequential := true
		for i := 1; i < len(options); i++ {
			// Simple check: options should be in order
			if options[i].Index != options[i-1].Index+1 {
				sequential = false
				break
			}
		}
		if sequential {
			confidence += 0.1
		}
	}

	// Cap confidence at 1.0
	if confidence > 1.0 {
		confidence = 1.0
	}

	return confidence
}

// GetPromptForOption returns the appropriate prompt for a menu option
func (ma *MenuAnalyzer) GetPromptForOption(option MenuOption, menuType MenuType) string {
	return ma.keySeq.GenerateMenuSelection(option.Index, menuType)
}

// ExtractPromptFromOutput extracts the prompt/question from output
func (ma *MenuAnalyzer) ExtractPromptFromOutput(output string) string {
	lines := strings.Split(output, "\n")

	// Look for the last non-empty line that looks like a prompt
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line != "" {
			// Check if it ends with a question mark or colon
			if strings.HasSuffix(line, "?") || strings.HasSuffix(line, ":") {
				return line
			}
			// Check if it matches any prompt pattern
			for _, pp := range PromptPatterns {
				if pp.Pattern.MatchString(line) {
					return line
				}
			}
		}
	}

	// Return last non-empty line as fallback
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line != "" {
			return line
		}
	}

	return ""
}
