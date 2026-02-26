package shell

import (
	"fmt"
	"strings"
)

// ANSI escape sequences for terminal control
const (
	CursorUp    = "\x1b[A"
	CursorDown  = "\x1b[B"
	CursorRight = "\x1b[C"
	CursorLeft  = "\x1b[D"
	Enter       = "\n"
	Escape      = "\x1b"
	Tab         = "\t"
)

// MenuType represents different types of menus
type MenuType string

const (
	MenuTypeNumbered MenuType = "numbered" // 1) Option A, 2) Option B
	MenuTypeBox      MenuType = "box"      // Box-drawn menu with arrow keys
	MenuTypeWizard   MenuType = "wizard"   // Multi-step wizard
)

// TerminalKeySequence generates terminal key sequences
type TerminalKeySequence struct{}

// NewTerminalKeySequence creates a new terminal key sequence generator
func NewTerminalKeySequence() *TerminalKeySequence {
	return &TerminalKeySequence{}
}

// GenerateArrowKeys generates arrow key sequences for menu navigation
func (tks *TerminalKeySequence) GenerateArrowKeys(direction string, count int) string {
	switch direction {
	case "up":
		return strings.Repeat(CursorUp, count)
	case "down":
		return strings.Repeat(CursorDown, count)
	case "left":
		return strings.Repeat(CursorLeft, count)
	case "right":
		return strings.Repeat(CursorRight, count)
	}
	return ""
}

// GenerateMenuSelection generates the input sequence for selecting a menu option
func (tks *TerminalKeySequence) GenerateMenuSelection(optionIndex int, menuType MenuType) string {
	switch menuType {
	case MenuTypeNumbered:
		// For numbered menus, just type the number and press enter
		return fmt.Sprintf("%d%s", optionIndex+1, Enter)
	case MenuTypeBox:
		// For box menus, use arrow keys to navigate and press enter
		return tks.GenerateArrowKeys("down", optionIndex) + Enter
	case MenuTypeWizard:
		// For wizards, typically use numbered selection
		return fmt.Sprintf("%d%s", optionIndex+1, Enter)
	default:
		return fmt.Sprintf("%d%s", optionIndex+1, Enter)
	}
}

// GenerateYesNo generates input for yes/no prompts
func (tks *TerminalKeySequence) GenerateYesNo(yes bool) string {
	if yes {
		return "y" + Enter
	}
	return "n" + Enter
}

// GenerateConfirmation generates input for confirmation prompts (Y/n)
func (tks *TerminalKeySequence) GenerateConfirmation(confirm bool) string {
	if confirm {
		return Enter // Default is usually yes, just press enter
	}
	return "n" + Enter
}

// GenerateTextInput generates text input with enter
func (tks *TerminalKeySequence) GenerateTextInput(text string) string {
	return text + Enter
}

// GeneratePasswordInput generates password input (usually just the text + enter)
func (tks *TerminalKeySequence) GeneratePasswordInput(password string) string {
	return password + Enter
}

// ParseArrowKey parses LLM suggested key names into ANSI sequences
func (tks *TerminalKeySequence) ParseArrowKey(keyName string) (string, error) {
	switch strings.ToUpper(keyName) {
	case "UP", "ARROW_UP", "CURSOR_UP":
		return CursorUp, nil
	case "DOWN", "ARROW_DOWN", "CURSOR_DOWN":
		return CursorDown, nil
	case "LEFT", "ARROW_LEFT", "CURSOR_LEFT":
		return CursorLeft, nil
	case "RIGHT", "ARROW_RIGHT", "CURSOR_RIGHT":
		return CursorRight, nil
	case "ENTER", "RETURN":
		return Enter, nil
	case "ESC", "ESCAPE":
		return Escape, nil
	case "TAB":
		return Tab, nil
	default:
		return "", fmt.Errorf("unknown key name: %s", keyName)
	}
}

// GenerateFromSuggestion generates input from an LLM suggestion
func (tks *TerminalKeySequence) GenerateFromSuggestion(suggestion string, inputType string) string {
	switch inputType {
	case "arrow":
		// Try to parse as arrow key
		if seq, err := tks.ParseArrowKey(suggestion); err == nil {
			return seq
		}
		// Fall through to text if not a recognized arrow key
		fallthrough
	case "text", "enter":
		if suggestion == "ENTER" || suggestion == "Enter" {
			return Enter
		}
		return suggestion + Enter
	default:
		return suggestion + Enter
	}
}
