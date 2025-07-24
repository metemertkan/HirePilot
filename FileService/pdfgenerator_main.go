package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/hirepilot/shared/models"
	sharedNats "github.com/hirepilot/shared/nats"
	"github.com/jung-kurt/gofpdf"
)

func main() {
	log.Println("Starting PDF Generator service...")

	// Create output directory if it doesn't exist
	outputDir := "/app/pdfs"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Initialize shared NATS JetStream
	js := sharedNats.InitJetStream()
	if js == nil {
		log.Fatalf("Failed to initialize JetStream")
	}
	defer sharedNats.Close()

	// Subscribe to CV generation messages using shared library (specific for PDF generation)
	_, err := sharedNats.SubscribeToCVGeneratedForPDF(func(data []byte) error {
		log.Printf("Received CV generation message")

		// Parse the message to extract job data (same structure as cover letter)
		var message map[string]interface{}
		if err := json.Unmarshal(data, &message); err != nil {
			log.Printf("Error unmarshaling CV message: %v", err)
			return err
		}

		// Extract job data from the message
		jobData, ok := message["data"].(map[string]interface{})
		if !ok {
			log.Printf("Invalid CV message format")
			return fmt.Errorf("invalid message format")
		}

		// Convert to models.Job struct
		jobBytes, err := json.Marshal(jobData)
		if err != nil {
			log.Printf("Error marshaling job data: %v", err)
			return err
		}

		var job models.Job
		if err := json.Unmarshal(jobBytes, &job); err != nil {
			log.Printf("Error unmarshaling job data: %v", err)
			return err
		}

		// Check if CV exists
		if !job.CvGenerated || job.Cv == "" {
			log.Printf("Job %d does not have a CV generated", job.Id)
			return fmt.Errorf("job %d does not have a CV generated", job.Id)
		}

		log.Printf("Processing CV for Job ID: %d, Company: %s, Title: %s",
			job.Id, job.Company, job.Title)

		// Create CVData structure from job data
		cvData := sharedNats.CVData{
			JobID:     job.Id,
			Title:     job.Title,
			Company:   job.Company,
			CVContent: job.Cv,
		}

		// Generate PDF
		pdfPath, err := generateCVPDF(cvData, outputDir)
		if err != nil {
			log.Printf("Error generating CV PDF: %v", err)
			return err
		}

		log.Printf("CV PDF generated successfully: %s", pdfPath)
		return nil
	})

	if err != nil {
		log.Fatalf("Failed to subscribe to CV generated messages: %v", err)
	}

	// Subscribe to cover letter generated messages using shared library
	_, err = sharedNats.SubscribeToCoverGeneratedGeneric(func(data []byte) error {
		log.Printf("Received cover letter generation message")
		return handleCoverLetterGenerated(data, outputDir)
	})

	if err != nil {
		log.Fatalf("Failed to subscribe to cover letter generated messages: %v", err)
	}

	log.Println("PDF Generator subscribed to CV and cover letter generation messages")
	log.Println("PDF Generator is running. Press Ctrl+C to exit.")

	// Keep the service running
	select {}
}

func generateCVPDF(cvData sharedNats.CVData, outputDir string) (string, error) {
	// Create new PDF document
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetMargins(15, 15, 15) // Reduced margins from 20 to 15

	// Clean and normalize the CV content for PDF compatibility
	cleanedCVContent := cleanTextForPDF(cvData.CVContent)

	// Parse and format the CV content
	err := formatCVContent(pdf, cleanedCVContent)
	if err != nil {
		log.Printf("Error formatting CV content, falling back to simple format: %v", err)
		// Fallback to simple formatting
		pdf.SetFont("Arial", "", 10)
		addWrappedText(pdf, cleanedCVContent, 170)
	}

	// Generate filename
	safeCompany := sanitizeFilename(cvData.Company)
	safeTitle := sanitizeFilename(cvData.Title)
	filename := fmt.Sprintf("Mete_Dosemeci_CV_%s_%s.pdf", safeCompany, safeTitle)

	pdfPath := filepath.Join(outputDir, filename)

	// Save PDF
	err = pdf.OutputFileAndClose(pdfPath)
	if err != nil {
		return "", fmt.Errorf("failed to save PDF: %w", err)
	}

	return pdfPath, nil
}

func addWrappedText(pdf *gofpdf.Fpdf, text string, width float64) {
	// Split text into lines that fit within the specified width
	lines := strings.Split(text, "\n")

	for _, line := range lines {
		if line == "" {
			pdf.Ln(2) // Reduced from 5 to 2
			continue
		}

		// Check if line fits, if not, wrap it
		for len(line) > 0 {
			// Find the maximum characters that fit in the width
			maxChars := 110 // Increased from 100 to fit more text per line
			if len(line) <= maxChars {
				pdf.Cell(0, 4, line) // Reduced height from 6 to 4
				pdf.Ln(4)            // Reduced from 6 to 4
				break
			}

			// Find last space before maxChars to avoid breaking words
			breakPoint := maxChars
			for i := maxChars - 1; i >= 0; i-- {
				if line[i] == ' ' {
					breakPoint = i
					break
				}
			}

			// Print the line segment
			pdf.Cell(0, 4, line[:breakPoint]) // Reduced height from 6 to 4
			pdf.Ln(4)                         // Reduced from 6 to 4

			// Continue with remaining text
			if breakPoint < len(line) {
				line = strings.TrimSpace(line[breakPoint:])
			} else {
				break
			}
		}
	}
}

func sanitizeFilename(filename string) string {
	// Replace invalid characters for filenames
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
		" ", "_",
	)

	sanitized := replacer.Replace(filename)

	// Limit length
	if len(sanitized) > 50 {
		sanitized = sanitized[:50]
	}

	return sanitized
}

func formatCVContent(pdf *gofpdf.Fpdf, content string) error {
	lines := strings.Split(content, "\n")

	for i, line := range lines {
		line = strings.TrimSpace(line)

		if line == "" {
			pdf.Ln(2) // Reduced from 4 to 2
			continue
		}

		// Check for different formatting patterns
		if (strings.HasPrefix(line, "**") && strings.HasSuffix(line, "**") && i == 0) ||
			(i == 0 && (strings.Contains(line, "Mete Dosemeci") || strings.Contains(line, "Mete"))) {
			// Name at the beginning of CV (first line) - handle both **Name** and plain Name formats
			name := line
			if strings.HasPrefix(name, "**") && strings.HasSuffix(name, "**") {
				name = strings.TrimPrefix(name, "**")
				name = strings.TrimSuffix(name, "**")
			}
			pdf.SetFont("Arial", "B", 14) // Reduced from 16 to 14
			pdf.Cell(0, 8, name)          // Reduced height from 10 to 8
			pdf.Ln(8)                     // Reduced spacing from 10 to 8
		} else if strings.Contains(line, " | ") && strings.Contains(line, "@") && i <= 2 {
			// Contact information line (contains email and phone) - keep on same line
			pdf.SetFont("Arial", "", 9) // Reduced from 10 to 9
			pdf.Cell(0, 5, line)        // Reduced height from 6 to 5
			pdf.Ln(6)                   // Reduced spacing from 10 to 6
		} else if strings.HasPrefix(line, "### ") {
			// Main section headings (like "PROFESSIONAL SUMMARY", "EXPERIENCE", etc.)
			heading := strings.TrimPrefix(line, "### ")
			pdf.SetFont("Arial", "B", 11) // Reduced from 13 to 11
			pdf.Ln(2)                     // Reduced from 4 to 2
			pdf.Cell(0, 5, heading)       // Reduced height from 6 to 5
			pdf.Ln(3)                     // Reduced from 5 to 3
		} else if strings.HasPrefix(line, "**") && strings.HasSuffix(line, "**") {
			// Job titles (bold text) - this should be just the job title
			jobTitle := strings.TrimPrefix(line, "**")
			jobTitle = strings.TrimSuffix(jobTitle, "**")
			
			pdf.SetFont("Arial", "B", 10)
			pdf.Ln(3) // Increased spacing before job title
			pdf.Cell(0, 5, jobTitle) // Increased height for better visibility
			pdf.Ln(6) // Increased spacing after job title to separate from company
		} else if strings.Contains(line, " | ") && (strings.Contains(line, "–") || strings.Contains(line, "-")) {
			// Company and date lines (like "Devoteam – Amsterdam, Netherlands | Dec 2023 – Present")
			pdf.SetFont("Arial", "", 9) // Reduced from 10 to 9
			pdf.Cell(0, 4, line)        // Reduced height from 5 to 4
			pdf.Ln(4)                   // Keep consistent spacing
		} else if isBulletPoint(line) {
			// Bullet points (handle all formats and malformed bullets)
			bulletText := extractBulletText(line)
			
			// Clean the bullet text to fix any encoding issues
			bulletText = cleanTextForPDF(bulletText)

			pdf.SetFont("Arial", "", 9) // Reduced from 10 to 9
			// Add simple bullet point using a dash
			pdf.Cell(8, 4, "- ") // Use simple dash as bullet point
			// Add wrapped text for the bullet point
			addBulletText(pdf, bulletText, 165) // Increased width slightly to compensate
		} else if strings.Contains(line, "---") {
			// Separator lines - add minimal space
			pdf.Ln(1) // Reduced from 3 to 1
		} else if line != "" {
			// Regular text
			pdf.SetFont("Arial", "", 9)    // Reduced from 10 to 9
			addWrappedText(pdf, line, 175) // Increased width slightly
			pdf.Ln(1)                      // Reduced from 2 to 1
		}
	}

	return nil
}

func addBulletText(pdf *gofpdf.Fpdf, text string, width float64) {
	// Handle bullet point text with inline bold formatting
	if strings.Contains(text, "**") {
		addFormattedText(pdf, text, 165)
	} else {
		// Handle simple bullet point text with proper wrapping
		words := strings.Fields(text)
		if len(words) == 0 {
			pdf.Ln(3) // Reduced from 5 to 3
			return
		}

		currentLine := ""
		maxCharsPerLine := 95 // Increased from 85 to fit more text per line

		for i, word := range words {
			testLine := currentLine
			if testLine != "" {
				testLine += " "
			}
			testLine += word

			if len(testLine) <= maxCharsPerLine || currentLine == "" {
				currentLine = testLine
			} else {
				// Print current line and start new one
				pdf.Cell(0, 4, currentLine) // Reduced height from 5 to 4
				pdf.Ln(4)                   // Reduced from 5 to 4
				pdf.Cell(8, 4, "")          // Reduced indent from 10 to 8
				currentLine = word
			}

			// If this is the last word, print the line
			if i == len(words)-1 {
				pdf.Cell(0, 4, currentLine) // Reduced height from 5 to 4
				pdf.Ln(4)                   // Reduced from 6 to 4
			}
		}
	}
}

func addFormattedText(pdf *gofpdf.Fpdf, text string, width float64) {
	// Handle text with inline bold formatting (**text**)
	parts := strings.Split(text, "**")

	lineHeight := 4.0 // Reduced from 5.0 to 4.0
	maxWidth := width
	currentLineWidth := 0.0

	for i, part := range parts {
		if part == "" {
			continue
		}

		// Determine if this part should be bold (odd indices after splitting by **)
		isBold := i%2 == 1

		if isBold {
			pdf.SetFont("Arial", "B", 9) // Reduced from 10 to 9
		} else {
			pdf.SetFont("Arial", "", 9) // Reduced from 10 to 9
		}

		// Split part into words for wrapping
		words := strings.Fields(part)
		for j, word := range words {
			if j > 0 {
				word = " " + word // Add space before word (except first)
			}

			// Check if word fits on current line
			wordWidth := pdf.GetStringWidth(word)

			if currentLineWidth+wordWidth > maxWidth && currentLineWidth > 0 {
				// Move to next line
				pdf.Ln(lineHeight)
				pdf.Cell(8, lineHeight, "") // Reduced indent from 10 to 8
				currentLineWidth = 0
			}

			// Print the word
			pdf.Cell(wordWidth, lineHeight, word)
			currentLineWidth += wordWidth
		}
	}

	// Move to next line after the formatted text
	pdf.Ln(4) // Reduced from 6 to 4
}

func handleCoverLetterGenerated(data []byte, outputDir string) error {
	log.Printf("Received cover letter generation message")

	// Parse the message to extract job data
	var message map[string]interface{}
	if err := json.Unmarshal(data, &message); err != nil {
		log.Printf("Error unmarshaling cover letter message: %v", err)
		return err
	}

	// Extract job data from the message
	jobData, ok := message["data"].(map[string]interface{})
	if !ok {
		log.Printf("Invalid cover letter message format")
		return fmt.Errorf("invalid message format")
	}

	// Convert to models.Job struct
	jobBytes, err := json.Marshal(jobData)
	if err != nil {
		log.Printf("Error marshaling job data: %v", err)
		return err
	}

	var job models.Job
	if err := json.Unmarshal(jobBytes, &job); err != nil {
		log.Printf("Error unmarshaling job data: %v", err)
		return err
	}

	// Check if cover letter exists
	if job.CoverLetter == "" {
		log.Printf("Job %d does not have a cover letter", job.Id)
		return fmt.Errorf("job %d does not have a cover letter", job.Id)
	}

	log.Printf("Processing cover letter for Job ID: %d, Company: %s, Title: %s",
		job.Id, job.Company, job.Title)

	// Generate cover letter PDF
	pdfPath, err := generateCoverLetterPDF(job, outputDir)
	if err != nil {
		log.Printf("Error generating cover letter PDF: %v", err)
		return err
	}

	log.Printf("Cover letter PDF generated successfully: %s", pdfPath)
	return nil
}

func cleanTextForPDF(text string) string {
	// First, let's log the problematic text for debugging
	if strings.Contains(text, "PandaDoc") {
		log.Printf("Debug - Original text with PandaDoc: %q", text)
	}
	
	// Convert text to ASCII-safe characters for PDF compatibility
	// This is more aggressive but ensures PDF rendering works correctly
	replacer := strings.NewReplacer(
		// Smart quotes - handle both proper Unicode and malformed UTF-8
		"\u201C", `"`, // U+201C LEFT DOUBLE QUOTATION MARK
		"\u201D", `"`, // U+201D RIGHT DOUBLE QUOTATION MARK
		"\u2018", "'", // U+2018 LEFT SINGLE QUOTATION MARK
		"\u2019", "'", // U+2019 RIGHT SINGLE QUOTATION MARK
		// Dashes
		"\u2013", "-", // U+2013 EN DASH
		"\u2014", "-", // U+2014 EM DASH
		// Other common problematic characters
		"\u2026", "...", // U+2026 HORIZONTAL ELLIPSIS
		"\u2122", "(TM)", // U+2122 TRADE MARK SIGN
		"\u00AE", "(R)", // U+00AE REGISTERED SIGN
		"\u00A9", "(C)", // U+00A9 COPYRIGHT SIGN
		// Non-breaking space
		"\u00A0", " ", // U+00A0 NON-BREAKING SPACE
		// Fix malformed UTF-8 sequences - using safe string literals
		"\xc3\xa2\xe2\x82\xac\xc2\xa2", "•", // Malformed bullet point
		"\xc3\xa2\xe2\x82\xac\xe2\x80\x9d", "'", // Malformed right single quote (PandaDoc's issue)
		"\xc3\xa2\xe2\x82\xac\xc5\x93", `"`, // Malformed left double quote
		"\xc3\xa2\xe2\x82\xac\xc2\x9d", `"`, // Malformed right double quote
		"\xc3\xa2\xe2\x82\xac\xe2\x80\x9c", "-", // Malformed em dash
		"\xc3\xa2\xe2\x82\xac\xc2\xa6", "...", // Malformed ellipsis
		"\xc3\xa2\xe2\x82\xac\xcb\x9c", "'", // Malformed left single quote
		// Additional hex-encoded malformed sequences for safety
		"\xe2\x80\xa2", "•", // Another hex-encoded malformed bullet
		"\xe2\x80\x99", "'", // Another hex-encoded malformed apostrophe
		"\xe2\x80\x9c", `"`, // Another hex-encoded malformed left quote
		"\xe2\x80\x9d", `"`, // Another hex-encoded malformed right quote
		"\xe2\x80\x93", "-", // Another hex-encoded malformed dash
		"\xe2\x97\x8f", "•", // Another hex-encoded bullet variant
	)

	result := replacer.Replace(text)
	
	// Use regex to catch any remaining malformed UTF-8 sequences
	// This is more aggressive and should catch patterns we might have missed
	
	// First, try to fix the most common malformed sequences using byte-level replacement
	result = strings.ReplaceAll(result, string([]byte{0xc3, 0xa2, 0xe2, 0x82, 0xac, 0xc2, 0xa2}), "•") // Malformed bullet
	result = strings.ReplaceAll(result, string([]byte{0xc3, 0xa2, 0xe2, 0x82, 0xac, 0xe2, 0x80, 0x9d}), "'") // Malformed apostrophe
	
	// Then use regex for any remaining patterns
	regexReplacements := map[string]string{
		`â€¢`: "•",  // Malformed bullet
		`â€™`: "'",  // Malformed apostrophe
		`â€œ`: `"`,  // Malformed left quote
		`â€`: `"`,   // Malformed right quote
		`â€"`: "-",  // Malformed dash
		`â—•`: "•",  // Another malformed bullet
		`â€¦`: "...", // Malformed ellipsis
		`â€˜`: "'",  // Malformed left single quote
	}
	
	for pattern, replacement := range regexReplacements {
		re := regexp.MustCompile(regexp.QuoteMeta(pattern))
		result = re.ReplaceAllString(result, replacement)
	}
	
	// Additional aggressive cleaning for any remaining non-ASCII characters
	// that might cause PDF rendering issues
	result = strings.Map(func(r rune) rune {
		// Keep ASCII printable characters (32-126)
		if r >= 32 && r <= 126 {
			return r
		}
		// Keep common whitespace characters
		if r == '\n' || r == '\r' || r == '\t' {
			return r
		}
		// Replace problematic Unicode characters with ASCII equivalents
		if r == '\u2018' || r == '\u2019' { // Smart quotes
			return '\''
		}
		if r == '\u201C' || r == '\u201D' { // Smart double quotes
			return '"'
		}
		if r == '\u2013' || r == '\u2014' { // Dashes
			return '-'
		}
		if r == '\u2026' { // Ellipsis
			return '.'
		}
		if r == '\u2022' { // Bullet point
			return '*'
		}
		// For any other non-ASCII character, replace with space
		return ' '
	}, result)
	
	// Log the result for debugging
	if strings.Contains(text, "PandaDoc") {
		log.Printf("Debug - Cleaned text with PandaDoc: %q", result)
	}
	
	return result
}

// isBulletPoint checks if a line is a bullet point (handles all formats and malformed bullets)
func isBulletPoint(line string) bool {
	if len(line) == 0 {
		return false
	}
	
	// Check for safe bullet point patterns first
	safePrefixes := []string{
		"*   ",    // Standard markdown bullet
		"●\t",     // Unicode bullet with tab
		"• ",      // Unicode bullet with space
		"* ",      // Simple asterisk bullet
	}
	
	for _, prefix := range safePrefixes {
		if strings.HasPrefix(line, prefix) {
			return true
		}
	}
	
	// Check for malformed UTF-8 sequences that represent bullets
	malformedBulletPrefixes := []string{
		"â€¢",     // Common malformed bullet sequence
		"â—•",     // Another malformed bullet variant
		"â€¢ ",    // Malformed bullet with space
		"â€¢\t",   // Malformed bullet with tab
	}
	
	for _, prefix := range malformedBulletPrefixes {
		if strings.HasPrefix(line, prefix) {
			return true
		}
	}
	
	// Check for hex-encoded malformed UTF-8 sequences
	if len(line) >= 3 {
		// Check if line starts with malformed UTF-8 bullet sequences
		if strings.HasPrefix(line, "\xc3\xa2") { // Starts with malformed UTF-8 sequence
			return true
		}
	}
	
	// Check for Unicode bullet points using rune conversion
	runes := []rune(line)
	if len(runes) > 0 {
		firstRune := runes[0]
		if firstRune == 0x2022 || firstRune == 0x2023 { // Unicode bullet points
			return true
		}
	}
	
	// Check for standard bullet characters
	firstChar := string(line[0])
	if firstChar == "•" || firstChar == "*" {
		return true
	}
	
	return false
}

// extractBulletText extracts the text content from a bullet point line
func extractBulletText(line string) string {
	// Handle malformed UTF-8 sequences that represent bullets
	malformedBulletPrefixes := []string{
		"â€¢",     // Common malformed bullet sequence
		"â—•",     // Another malformed bullet variant
		"â€¢ ",    // Malformed bullet with space
		"â€¢\t",   // Malformed bullet with tab
	}
	
	for _, prefix := range malformedBulletPrefixes {
		if strings.HasPrefix(line, prefix) {
			return strings.TrimLeft(strings.TrimPrefix(line, prefix), " \t")
		}
	}
	
	// Use safe bullet prefixes
	safeBulletPrefixes := []string{
		"*   ",    // Standard markdown bullet
		"●\t",     // Unicode bullet with tab
		"• ",      // Unicode bullet with space
		"* ",      // Simple asterisk bullet
	}
	
	for _, prefix := range safeBulletPrefixes {
		if strings.HasPrefix(line, prefix) {
			return strings.TrimPrefix(line, prefix)
		}
	}
	
	// Handle malformed UTF-8 sequences using safe hex detection
	if len(line) > 0 {
		// Check for malformed bullet characters using safe detection
		if strings.HasPrefix(line, "\xc3\xa2") { // Starts with malformed UTF-8
			// Find the first space or tab after the malformed sequence
			for i := 1; i < len(line); i++ {
				if line[i] == ' ' || line[i] == '\t' {
					return strings.TrimLeft(line[i:], " \t")
				}
			}
			// If no space/tab found, try to extract after first few bytes
			if len(line) > 3 {
				return strings.TrimLeft(line[3:], " \t")
			}
		}
		
		// Check for standard bullet characters
		firstChar := string(line[0])
		if firstChar == "•" || firstChar == "*" {
			// Remove the bullet character and any following whitespace
			remaining := strings.TrimLeft(line[1:], " \t")
			return remaining
		}
	}
	
	return line
}

func generateCoverLetterPDF(job models.Job, outputDir string) (string, error) {
	// Create new PDF document
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	// Clean and normalize the cover letter text for PDF compatibility
	cleanedCoverLetter := cleanTextForPDF(job.CoverLetter)

	// Add cover letter content with text wrapping
	pdf.SetFont("Arial", "", 10)
	addWrappedText(pdf, cleanedCoverLetter, 190)

	// Generate filename
	safeCompany := sanitizeFilename(job.Company)
	safeTitle := sanitizeFilename(job.Title)
	filename := fmt.Sprintf("CoverLetter_%s_%s.pdf",
		safeCompany, safeTitle)

	pdfPath := filepath.Join(outputDir, filename)

	// Save PDF
	err := pdf.OutputFileAndClose(pdfPath)
	if err != nil {
		return "", fmt.Errorf("failed to save cover letter PDF: %w", err)
	}

	return pdfPath, nil
}
