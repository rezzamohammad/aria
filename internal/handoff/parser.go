package handoff

import (
	"bufio"
	"fmt"
	"strings"
)

// Report represents a parsed worker report.
type Report struct {
	TaskID      string
	Status      string
	Score       string
	Role        string
	CLITool     string
	CompletedAt string
	Body        string
	Fields      map[string]string
}

// ParseReport parses a YAML-frontmatter report string.
func ParseReport(content string) (*Report, error) {
	r := &Report{
		Fields: make(map[string]string),
	}

	scanner := bufio.NewScanner(strings.NewReader(content))
	inFrontmatter := false
	frontmatterDone := false
	var bodyLines []string

	for scanner.Scan() {
		line := scanner.Text()

		if !inFrontmatter && !frontmatterDone && strings.TrimSpace(line) == "---" {
			inFrontmatter = true
			continue
		}

		if inFrontmatter && strings.TrimSpace(line) == "---" {
			inFrontmatter = false
			frontmatterDone = true
			continue
		}

		if inFrontmatter {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				value = strings.Trim(value, "\"")
				r.Fields[key] = value

				switch key {
				case "task_id":
					r.TaskID = value
				case "status":
					r.Status = value
				case "score":
					r.Score = value
				case "role":
					r.Role = value
				case "cli_tool":
					r.CLITool = value
				case "completed_at":
					r.CompletedAt = value
				}
			}
		} else if frontmatterDone {
			bodyLines = append(bodyLines, line)
		}
	}

	r.Body = strings.TrimSpace(strings.Join(bodyLines, "\n"))

	if r.TaskID == "" {
		return nil, fmt.Errorf("report missing task_id field")
	}

	return r, nil
}
