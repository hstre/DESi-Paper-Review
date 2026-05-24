package review

import (
	"regexp"
	"strings"
)

var headerRe = regexp.MustCompile(`^(#{1,6})\s+(.*\S)\s*$`)

var knownSections = map[string]bool{
	"abstract": true, "introduction": true, "background": true,
	"related work": true, "method": true, "methods": true,
	"methodology": true, "approach": true, "experimental setup": true,
	"experiments": true, "evaluation": true, "results": true,
	"analysis": true, "discussion": true, "conclusion": true,
	"conclusions": true, "limitations": true, "future work": true,
	"references": true, "acknowledgements": true, "acknowledgments": true,
}

// ParseText parses raw paper text into a ParsedPaper.
func ParseText(text string) ParsedPaper {
	norm := strings.ReplaceAll(text, "\r\n", "\n")
	norm = strings.ReplaceAll(norm, "\r", "\n")
	lines := strings.Split(norm, "\n")
	return ParsedPaper{
		Title:    detectTitle(lines),
		Sections: splitSections(lines),
		RawText:  norm,
	}
}

func detectTitle(lines []string) string {
	for _, ln := range lines {
		if m := headerRe.FindStringSubmatch(ln); m != nil && len(m[1]) == 1 {
			return strings.TrimSpace(m[2])
		}
	}
	for _, ln := range lines {
		if strings.TrimSpace(ln) != "" {
			return strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(ln), "#"))
		}
	}
	return "Untitled"
}

func headingOf(line string) (level int, title string, ok bool) {
	if m := headerRe.FindStringSubmatch(line); m != nil {
		return len(m[1]), strings.TrimSpace(m[2]), true
	}
	s := strings.TrimSpace(line)
	if s != "" && len([]rune(s)) <= 60 {
		key := strings.ToLower(strings.TrimSpace(strings.TrimRight(s, ":")))
		if knownSections[key] {
			return 2, strings.TrimSpace(strings.TrimRight(s, ":")), true
		}
	}
	return 0, "", false
}

func splitSections(lines []string) []Section {
	sections := []Section{}
	var curTitle string
	haveTitle := false
	curLevel := 1
	var body []string

	flush := func() {
		text := strings.TrimSpace(strings.Join(body, "\n"))
		if haveTitle {
			sections = append(sections, Section{Title: curTitle, Level: curLevel, Text: text})
		} else if text != "" {
			sections = append(sections, Section{Title: "Preamble", Level: 1, Text: text})
		}
	}

	for _, ln := range lines {
		if level, title, ok := headingOf(ln); ok {
			flush()
			curLevel, curTitle, haveTitle = level, title, true
			body = nil
		} else {
			body = append(body, ln)
		}
	}
	flush()
	return sections
}

// SectionTitles returns the ordered section titles.
func (p ParsedPaper) SectionTitles() []string {
	out := make([]string, len(p.Sections))
	for i, s := range p.Sections {
		out[i] = s.Title
	}
	return out
}
