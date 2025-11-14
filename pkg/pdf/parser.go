package pdf

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jagdpruefer/parser/pkg/models"
)

// Type aliases for convenience
type Question = models.Question
type Option = models.Option
type QuestionCatalog = models.QuestionCatalog

// CurrentTimestamp returns the current time in RFC3339 format
func CurrentTimestamp() string {
	return time.Now().Format(time.RFC3339)
}

// Parser handles PDF parsing
type Parser struct {
	pdfPath string
}

// NewParser creates a new PDF parser
func NewParser(pdfPath string) *Parser {
	return &Parser{pdfPath: pdfPath}
}

// Parse extracts questions from the PDF
func (p *Parser) Parse() (*models.QuestionCatalog, error) {
	// Extract text from PDF using pdftotext
	text, err := p.extractTextFromPDF()
	if err != nil {
		return nil, fmt.Errorf("failed to extract text from PDF: %w", err)
	}

	// Parse the extracted text
	catalog, err := p.parseText(text)
	if err != nil {
		return nil, fmt.Errorf("failed to parse text: %w", err)
	}

	return catalog, nil
}

// extractTextFromPDF uses pdftotext to extract text content
func (p *Parser) extractTextFromPDF() (string, error) {
	cmd := exec.Command("pdftotext", p.pdfPath, "-")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("pdftotext failed: %w", err)
	}
	return string(output), nil
}

// Question represents a raw parsed question with its content
type rawQuestion struct {
	number  int
	text    string
	options map[string]*models.Option
}

func newRawQuestion(number int, text string) *rawQuestion {
	return &rawQuestion{
		number:  number,
		text:    text,
		options: make(map[string]*models.Option),
	}
}

// parseText parses the extracted text into questions
func (p *Parser) parseText(text string) (*models.QuestionCatalog, error) {
	catalog := &models.QuestionCatalog{
		Title:        "Jagdfrageprüfer Bayern",
		Year:         2025,
		State:        "by",
		Subject:      "Jagdwaffen, Jagd- und Fanggeräte",
		Questions:    []models.Question{},
		LastModified: time.Now().Format(time.RFC3339),
	}

	lines := strings.Split(text, "\n")

	// Regex patterns
	questionOnlyPattern := regexp.MustCompile(`^\s*(\d+)\.\s*$`)
	questionWithTextPattern := regexp.MustCompile(`^\s*(\d+)\.\s+(.+)$`)
	optionLinePattern := regexp.MustCompile(`^\s*([a-f])\)\s*(.*)$`)
	optionWithXPattern := regexp.MustCompile(`^\s*X\s+([a-f])\)\s*(.*)$`)
	justXPattern := regexp.MustCompile(`^\s*X\s*$`)
	emptyLinePattern := regexp.MustCompile(`^\s*$`)

	// Raw questions keyed by number
	rawQuestions := make(map[int]*rawQuestion)
	var lastQuestionNum int
	var currentCategory string
	var nextOptionIsCorrect bool // Track if next option should be marked as correct


	// Skip header content until we find the first real question section
	// Look for pattern like "N.M" where N and M are numbers (e.g., "1.1", "3.1", "4.2")
	startIdx := 0
	sectionHeaderPattern := regexp.MustCompile(`^\s*\d+\.\d+\s+`)

	for i := 0; i < len(lines); i++ {
		trimmedLine := strings.TrimSpace(lines[i])
		if sectionHeaderPattern.MatchString(trimmedLine) {
			startIdx = i
			// Capture the category at this point
			for j := i - 1; j >= 0 && j >= i-50; j-- {
				if strings.Contains(strings.TrimSpace(lines[j]), "Sachgebiet") && strings.Contains(strings.TrimSpace(lines[j]), ":") {
					currentCategory = strings.TrimSpace(lines[j])
					break
				}
			}
			break
		}
	}

	for i := startIdx; i < len(lines); i++ {
		trimmedLine := strings.TrimSpace(lines[i])

		// Skip empty lines
		if emptyLinePattern.MatchString(trimmedLine) {
			continue
		}

		// Track category changes - look for "1. Sachgebiet:" or "1.1 Lang- und..." patterns
		if strings.Contains(trimmedLine, "Sachgebiet") && strings.Contains(trimmedLine, ":") {
			currentCategory = trimmedLine
		}

		// Skip metadata/footer lines
		if strings.Contains(trimmedLine, "Stand:") || strings.Contains(trimmedLine, "Seite") ||
			strings.Contains(trimmedLine, "Zweitkorrektor") || strings.Contains(trimmedLine, "HERAUSGEBER") {
			continue
		}

		// Check for question number only (question on next line)
		if matches := questionOnlyPattern.FindStringSubmatch(trimmedLine); matches != nil {
			qNum, err := strconv.Atoi(matches[1])
			if err != nil || qNum < 1 {
				continue
			}

			// Question text is on the next line(s)
			lastQuestionNum = qNum
			var qText string

			// Read the question text from next lines
			for j := i + 1; j < len(lines) && j < i+15; j++ {
				nextLine := strings.TrimSpace(lines[j])

				if emptyLinePattern.MatchString(nextLine) {
					continue
				}

				// Stop if we hit an option line (with or without X marker)
				if optionLinePattern.MatchString(nextLine) || optionWithXPattern.MatchString(nextLine) || justXPattern.MatchString(nextLine) {
					break
				}

				// Stop if we hit another question
				if questionOnlyPattern.MatchString(nextLine) || questionWithTextPattern.MatchString(nextLine) {
					break
				}

				qText += " " + nextLine
			}

			qText = strings.TrimSpace(qText)
			// Remove trailing "X a)" which is mistakenly included
			qText = regexp.MustCompile(`\s+X\s+[a-f]\)\s*$`).ReplaceAllString(qText, "")
			if len(qText) > 5 {
				rawQuestions[qNum] = newRawQuestion(qNum, qText)
			}
		}

		// Check for question with text on same line
		if matches := questionWithTextPattern.FindStringSubmatch(trimmedLine); matches != nil {
			qNum, err := strconv.Atoi(matches[1])
			if err != nil || qNum < 1 {
				continue
			}

			// Skip if looks like sub-section (e.g., "1.1 Lang- und Kurzwaffen")
			if !strings.ContainsRune(matches[2], '?') {
				continue
			}

			lastQuestionNum = qNum
			qText := strings.TrimSpace(matches[2])

			// Collect rest of multi-line question if needed
			for j := i + 1; j < len(lines) && j < i+10; j++ {
				nextLine := strings.TrimSpace(lines[j])

				if emptyLinePattern.MatchString(nextLine) {
					continue
				}

				// Stop at option
				if optionLinePattern.MatchString(nextLine) {
					break
				}

				// Stop at next question
				if questionOnlyPattern.MatchString(nextLine) || questionWithTextPattern.MatchString(nextLine) {
					break
				}

				qText += " " + nextLine
			}

			qText = strings.TrimSpace(qText)
			qText = strings.ReplaceAll(qText, "  ", " ")
			// Clean up - remove any trailing option markers
			qText = regexp.MustCompile(`\s+X\s+[a-f]\).*$`).ReplaceAllString(qText, "")
			qText = strings.TrimSpace(qText)
			if len(qText) > 5 {
				rawQuestions[qNum] = newRawQuestion(qNum, qText)
			}
		}

		// Check for standalone X marker (marks next option as correct)
		if lastQuestionNum > 0 && justXPattern.MatchString(trimmedLine) {
			nextOptionIsCorrect = true
			continue
		}

		// Handle options - check for "X letter)" format first
		if lastQuestionNum > 0 && rawQuestions[lastQuestionNum] != nil {
			if matches := optionWithXPattern.FindStringSubmatch(trimmedLine); matches != nil {
				letter := matches[1]
				optText := strings.TrimSpace(matches[2])

				// Collect multi-line option text
				for j := i + 1; j < len(lines) && j < i+8; j++ {
					nextLine := strings.TrimSpace(lines[j])

					if emptyLinePattern.MatchString(nextLine) {
						continue
					}

					// Stop at next option or question
					if optionLinePattern.MatchString(nextLine) || optionWithXPattern.MatchString(nextLine) {
						break
					}
					if questionOnlyPattern.MatchString(nextLine) || questionWithTextPattern.MatchString(nextLine) {
						break
					}
					if strings.Contains(nextLine, "Sachgebiet") || strings.Contains(nextLine, "Stand:") {
						break
					}

					optText += " " + nextLine
				}

				optText = strings.TrimSpace(optText)
				// Clean up - remove any trailing X or option markers
				optText = regexp.MustCompile(`\s+X\s*$`).ReplaceAllString(optText, "")
				optText = regexp.MustCompile(`\s+[a-f]\)\s*$`).ReplaceAllString(optText, "")
				optText = strings.TrimSpace(optText)

				rawQuestions[lastQuestionNum].options[letter] = &models.Option{
					Letter:  letter,
					Text:    optText,
					Correct: true,
				}
				nextOptionIsCorrect = false
			} else if matches := optionLinePattern.FindStringSubmatch(trimmedLine); matches != nil {
				letter := matches[1]
				optText := strings.TrimSpace(matches[2])

				// Collect multi-line option text
				for j := i + 1; j < len(lines) && j < i+8; j++ {
					nextLine := strings.TrimSpace(lines[j])

					if emptyLinePattern.MatchString(nextLine) {
						continue
					}

					// Stop at next option or question
					if optionLinePattern.MatchString(nextLine) || optionWithXPattern.MatchString(nextLine) {
						break
					}
					if questionOnlyPattern.MatchString(nextLine) || questionWithTextPattern.MatchString(nextLine) {
						break
					}
					if strings.Contains(nextLine, "Sachgebiet") || strings.Contains(nextLine, "Stand:") {
						break
					}

					optText += " " + nextLine
				}

				optText = strings.TrimSpace(optText)
				// Clean up - remove any trailing markers
				optText = regexp.MustCompile(`\s+X\s*$`).ReplaceAllString(optText, "")
				optText = regexp.MustCompile(`\s+[a-f]\)\s*$`).ReplaceAllString(optText, "")
				optText = strings.TrimSpace(optText)

				isCorrect := nextOptionIsCorrect
				nextOptionIsCorrect = false // Reset after using

				rawQuestions[lastQuestionNum].options[letter] = &models.Option{
					Letter:  letter,
					Text:    optText,
					Correct: isCorrect,
				}
			}
		}
	}

	// Sort question numbers and build final catalog
	questionNums := make([]int, 0, len(rawQuestions))
	for num := range rawQuestions {
		questionNums = append(questionNums, num)
	}

	// Simple bubble sort
	for i := 0; i < len(questionNums); i++ {
		for j := i + 1; j < len(questionNums); j++ {
			if questionNums[j] < questionNums[i] {
				questionNums[i], questionNums[j] = questionNums[j], questionNums[i]
			}
		}
	}

	// Build final catalog
	for _, num := range questionNums {
		rq := rawQuestions[num]

		if len(rq.text) > 5 && len(rq.options) > 0 {
			// Convert options map to slice in order
			var opts []models.Option
			letters := []string{"a", "b", "c", "d", "e", "f"}
			for _, letter := range letters {
				if opt, exists := rq.options[letter]; exists {
					opts = append(opts, *opt)
				}
			}

			q := models.Question{
				ID:       rq.number,
				Text:     rq.text,
				Options:  opts,
				Category: currentCategory,
			}
			catalog.Questions = append(catalog.Questions, q)
		}
	}

	catalog.TotalCount = len(catalog.Questions)
	return catalog, nil
}

// ParseFile is a convenience function that takes a filename and returns the parsed catalog
func ParseFile(filename string) (*models.QuestionCatalog, error) {
	parser := NewParser(filename)
	return parser.Parse()
}
