package cowsay

import (
	"fmt"
	// "unicode"
	"regexp"
	"strings"
)

func LongestLine(lines []string) string {
	lenLongestLine := 0
	longestLine := ""
	for _, line := range lines {
		if len(line) > lenLongestLine {
			lenLongestLine = len(line)
			longestLine = line
		}
	}
	return longestLine
}

func BoxedStrings(lines []string) []string {
	var boxedStrings []string
	if len(lines) == 0 {
		lines = append(lines, "")
	}

	lenLongestLine := len(LongestLine(lines))
	topStr := fmt.Sprintf(" %s", strings.Repeat("_", lenLongestLine+2))
	botStr := fmt.Sprintf(" %s", strings.Repeat("-", lenLongestLine+2))
	boxedStrings = append(boxedStrings, topStr)

	switch {
	case len(lines) == 1:
		boxedStrings = append(boxedStrings, fmt.Sprintf("< %s >", lines[0]))

	case len(lines) > 1:
		for i, line := range lines {
			if i == 0 {
				boxedStrings = append(boxedStrings,
					fmt.Sprintf("/ %-*s \\", lenLongestLine, line))
			} else if i == len(lines)-1 {
				boxedStrings = append(boxedStrings,
					fmt.Sprintf("\\ %-*s /", lenLongestLine, line))
			} else {
				boxedStrings = append(boxedStrings,
					fmt.Sprintf("| %-*s |", lenLongestLine, line))
			}
		}
	}
	boxedStrings = append(boxedStrings, botStr)
	return boxedStrings
}

func Format(text string) string {
	cowFooter := `
        \   ^__^
         \  (oo)\_______
            (__)\       )\/\
                ||----w |
                ||     ||
`

	// fmt.Printf("Text was: %s\n", text)
	re := regexp.MustCompile("\\s+")
	text = re.ReplaceAllString(text, " ")
	cleanedText := []rune(text)
	// fmt.Printf("Text is now: %s\n", text)
	// fmt.Printf("len(cleanedText): %d\n", len(cleanedText))

	// Position of all whitespace characters
	var spacePos []int
	for i, v := range cleanedText {
		if v == rune(' ') {
			// fmt.Printf("Found space at index: %d\n", i)
			spacePos = append(spacePos, i)
		}
	}
	var breakPos []int
	prevBreakPos := -1
	currBreakPos := -1
	for i, v := range spacePos {
		currBreakPos = v
		lenBeforeBreak := currBreakPos - prevBreakPos
		// fmt.Printf("lenBeforeBreak: %d\n", lenBeforeBreak)
		if lenBeforeBreak > 40 {
			if i > 0 {
				breakPos = append(breakPos, spacePos[i-1])
			}
			prevBreakPos = v
		}
	}

	for _, v := range breakPos {
		// fmt.Printf("BreakPos has: %d entries\n", v)
		cleanedText[v] = rune('\n')
	}

	cleanedString := string(cleanedText)
	textParts := strings.SplitN(cleanedString, "\n", -1)

	// fmt.Printf("len(textParts): %d\n", len(textParts))
	boxedLines := BoxedStrings(textParts)

	return strings.Join(boxedLines, "\n") + cowFooter
}
