package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jagdpruefer/parser/pkg/pdf"
)

func main() {
	// Command-line flags
	inputPDF := flag.String("input", "", "Path to the PDF file to parse (or use -batch for multiple files)")
	outputJSON := flag.String("output", "", "Path to output JSON file (optional, defaults to questions.json)")
	batch := flag.Bool("batch", false, "Process all sg*.pdf files in current directory")
	verbose := flag.Bool("verbose", false, "Enable verbose output")
	dir := flag.String("dir", ".", "Directory to search for PDF files (used with -batch)")

	flag.Parse()

	if *batch {
		processBatch(*dir, *outputJSON, *verbose)
	} else {
		processSingle(*inputPDF, *outputJSON, *verbose)
	}
}

func processSingle(inputPDF, outputJSON string, verbose bool) {
	// Validate input
	if inputPDF == "" {
		fmt.Fprintf(os.Stderr, "Error: input file required\n")
		fmt.Fprintf(os.Stderr, "Usage: parser -input <pdf-file> [-output <json-file>] [-verbose]\n")
		os.Exit(1)
	}

	// Check if file exists
	if _, err := os.Stat(inputPDF); err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot read input file: %v\n", err)
		os.Exit(1)
	}

	// Set default output path
	if outputJSON == "" {
		outputJSON = filepath.Join(filepath.Dir(inputPDF), "questions.json")
	}

	if verbose {
		fmt.Printf("Parsing PDF: %s\n", inputPDF)
	}

	// Parse the PDF
	catalog, err := pdf.ParseFile(inputPDF)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing PDF: %v\n", err)
		os.Exit(1)
	}

	if verbose {
		fmt.Printf("Successfully parsed %d questions\n", catalog.TotalCount)
		fmt.Printf("Title: %s\n", catalog.Title)
		fmt.Printf("Year: %d\n", catalog.Year)
		fmt.Printf("State: %s\n", catalog.State)
	}

	writeCatalog(catalog, outputJSON, verbose)
}

func processBatch(dir, outputJSON string, verbose bool) {
	// Find all sg*.pdf files
	entries, err := os.ReadDir(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading directory: %v\n", err)
		os.Exit(1)
	}

	// Collect sg PDF files
	type sgFile struct {
		sgNum int
		path  string
	}

	var sgFiles []sgFile
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".pdf") {
			// Check if it matches pattern fragekatalog_*_sg*.pdf
			if strings.Contains(entry.Name(), "_sg") {
				// Extract SG number
				parts := strings.Split(entry.Name(), "_sg")
				if len(parts) == 2 {
					sgNumStr := strings.TrimSuffix(parts[1], ".pdf")
					var sgNum int
					fmt.Sscanf(sgNumStr, "%d", &sgNum)
					if sgNum > 0 {
						sgFiles = append(sgFiles, sgFile{sgNum, filepath.Join(dir, entry.Name())})
					}
				}
			}
		}
	}

	if len(sgFiles) == 0 {
		fmt.Fprintf(os.Stderr, "Error: no sg*.pdf files found in %s\n", dir)
		os.Exit(1)
	}

	// Sort by SG number
	sort.Slice(sgFiles, func(i, j int) bool {
		return sgFiles[i].sgNum < sgFiles[j].sgNum
	})

	if verbose {
		fmt.Printf("Found %d PDF files to process:\n", len(sgFiles))
		for _, f := range sgFiles {
			fmt.Printf("  SG%d: %s\n", f.sgNum, f.path)
		}
	}

	// Parse all files and merge
	mergedCatalog := &pdf.QuestionCatalog{
		Title:      "Jagdfrageprüfer Bayern - Alle Sachgebiete",
		Year:       2025,
		State:      "by",
		Subject:    "Alle Sachgebiete (SG 1-6)",
		Questions:  []pdf.Question{},
		LastModified: "",
	}

	totalParsed := 0

	for _, sgFile := range sgFiles {
		if verbose {
			fmt.Printf("\nProcessing SG%d: %s\n", sgFile.sgNum, sgFile.path)
		}

		catalog, err := pdf.ParseFile(sgFile.path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing %s: %v\n", sgFile.path, err)
			continue
		}

		if verbose {
			fmt.Printf("  Parsed %d questions\n", catalog.TotalCount)
		}

		mergedCatalog.Questions = append(mergedCatalog.Questions, catalog.Questions...)
		totalParsed += catalog.TotalCount
	}

	mergedCatalog.TotalCount = len(mergedCatalog.Questions)
	mergedCatalog.LastModified = pdf.CurrentTimestamp()

	if verbose {
		fmt.Printf("\nTotal questions parsed: %d\n", totalParsed)
	}

	// Set output path
	if outputJSON == "" {
		outputJSON = filepath.Join(dir, "questions.json")
	}

	writeCatalog(mergedCatalog, outputJSON, verbose)
}

func writeCatalog(catalog *pdf.QuestionCatalog, outputPath string, verbose bool) {
	// Marshal to JSON
	jsonData, err := json.MarshalIndent(catalog, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling JSON: %v\n", err)
		os.Exit(1)
	}

	// Write to file
	err = os.WriteFile(outputPath, jsonData, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing output file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully wrote %d questions to: %s\n", catalog.TotalCount, outputPath)

	if verbose {
		// Show statistics
		correctCount := 0
		for _, q := range catalog.Questions {
			for _, opt := range q.Options {
				if opt.Correct {
					correctCount++
				}
			}
		}

		fmt.Printf("\nStatistics:\n")
		fmt.Printf("  Total Questions: %d\n", catalog.TotalCount)
		fmt.Printf("  Total Correct Answers: %d\n", correctCount)
		fmt.Printf("  Title: %s\n", catalog.Title)

		// Show first question as sample
		if len(catalog.Questions) > 0 {
			fmt.Printf("\n--- Sample Question ---\n")
			q := catalog.Questions[0]
			fmt.Printf("Q%d: %s\n", q.ID, q.Text)
			fmt.Printf("Category: %s\n", q.Category)
			fmt.Printf("Options:\n")
			for _, opt := range q.Options {
				correct := ""
				if opt.Correct {
					correct = " [CORRECT]"
				}
				fmt.Printf("  %s) %s%s\n", opt.Letter, opt.Text, correct)
			}
		}
	}
}
