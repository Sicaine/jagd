package models

// Option represents a single answer option
type Option struct {
	Letter  string `json:"letter"`  // a, b, c, d, e, f
	Text    string `json:"text"`    // The answer text
	Correct bool   `json:"correct"` // Whether this is a correct answer
}

// Question represents a single exam question
type Question struct {
	ID       int       `json:"id"`       // Question number (1, 2, 3, ...)
	Text     string    `json:"text"`     // The question text
	Options  []Option  `json:"options"`  // List of answer options
	Category string    `json:"category"` // Category/subject (e.g., "Jagdwaffen")
}

// QuestionCatalog represents the entire collection of questions
type QuestionCatalog struct {
	Title        string       `json:"title"`        // Title of the exam
	Year         int          `json:"year"`         // Year of the exam
	State        string       `json:"state"`        // State code (e.g., "by" for Bayern)
	Subject      string       `json:"subject"`      // Main subject area
	TotalCount   int          `json:"totalCount"`   // Total number of questions
	Questions    []Question   `json:"questions"`    // List of all questions
	LastModified string       `json:"lastModified"` // When this was generated
}
