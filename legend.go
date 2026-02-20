package asciigraph

import (
	"unicode/utf8"

	"github.com/pancsta/tcell-v2"
)

// createLegendItem calculates the length of a legend item (box + space + text).
// Used for layout calculations when rendering to tcell.Screen.
func createLegendItem(text string, color tcell.Color) int {
	// Add 2 for box (■) and space
	return utf8.RuneCountInString(text) + 2
}
