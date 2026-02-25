package claudemd

import (
	"fmt"
	"strings"
)

const maxChunkChars = 6000

// Splitter breaks a large context document into sized chunks
// suitable for injection into agents with limited context windows.
type Splitter struct {
	MaxChars int
}

// NewSplitter creates a Splitter with the default chunk size.
func NewSplitter() *Splitter {
	return &Splitter{MaxChars: maxChunkChars}
}

// Split divides a CLAUDE.md content string into chunks of at most MaxChars.
// Each chunk is self-contained with a header indicating its part number.
func (s *Splitter) Split(content string) []string {
	if len(content) <= s.MaxChars {
		return []string{content}
	}

	lines := strings.Split(content, "\n")
	var chunks []string
	var current strings.Builder
	partNum := 1

	header := func(n int) string {
		return fmt.Sprintf("<!-- ARIA context part %d -->\n", n)
	}

	current.WriteString(header(partNum))

	for _, line := range lines {
		addition := line + "\n"
		if current.Len()+len(addition) > s.MaxChars {
			chunks = append(chunks, current.String())
			partNum++
			current.Reset()
			current.WriteString(header(partNum))
		}
		current.WriteString(addition)
	}

	if current.Len() > 0 {
		chunks = append(chunks, current.String())
	}

	return chunks
}

// FirstChunk returns only the first split chunk — useful when an agent
// only needs the task definition, not the full prior-work history.
func (s *Splitter) FirstChunk(content string) string {
	chunks := s.Split(content)
	if len(chunks) > 0 {
		return chunks[0]
	}
	return content
}
