package contextengine

// InlineDocuments writes the text files into the question.
func InlineDocuments(text string, docs []Document) (string, []Document) { return text, docs }

// PDFPages estimates how many pages a PDF has.
func PDFPages(data []byte) int { return 0 }
