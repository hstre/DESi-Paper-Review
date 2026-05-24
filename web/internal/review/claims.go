package review

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

// OverclaimTerms are the paper-level overclaim terms, in canonical order.
var OverclaimTerms = []string{
	"first", "novel", "robust", "significant",
	"generalizes", "solves", "proves",
}

var overclaimRes = map[string]*regexp.Regexp{
	"first":       regexp.MustCompile(`(?i)\bfirst\b`),
	"novel":       regexp.MustCompile(`(?i)\bnovel(?:ty)?\b`),
	"robust":      regexp.MustCompile(`(?i)\brobust(?:ness|ly)?\b`),
	"significant": regexp.MustCompile(`(?i)\bsignificant(?:ly)?\b`),
	"generalizes": regexp.MustCompile(`(?i)\bgeneraliz(?:e|es|ed|ing|ation|able)\b`),
	"solves":      regexp.MustCompile(`(?i)\bsolv(?:e|es|ed|ing)\b`),
	"proves":      regexp.MustCompile(`(?i)\bprov(?:e|es|ed|ing|en)\b`),
}

// strongCategories demand evidence.
var strongCategories = map[string]bool{
	CatMain: true, CatResult: true, CatNovelty: true, CatGeneralization: true,
}

type signalGroup struct {
	category string
	res      []*regexp.Regexp
}

func compile(pats ...string) []*regexp.Regexp {
	out := make([]*regexp.Regexp, len(pats))
	for i, p := range pats {
		out[i] = regexp.MustCompile(p)
	}
	return out
}

// categorySignals are checked top to bottom; first match wins.
var categorySignals = []signalGroup{
	{CatNovelty, compile(
		`(?i)\bnovel(?:ty)?\b`, `(?i)\bfirst\b`, `(?i)\bunprecedented\b`,
		`(?i)\bfor the first time\b`)},
	{CatGeneralization, compile(
		`(?i)\bgeneraliz`, `(?i)\buniversal`, `(?i)\bacross (?:all )?(?:tasks|domains)\b`,
		`(?i)\bin general\b`, `(?i)\bany (?:task|domain|input)\b`)},
	{CatLimitation, compile(
		`(?i)\blimitation`, `(?i)\bhowever\b`, `(?i)\bcaveat`, `(?i)\bdoes not\b`,
		`(?i)\bcannot\b`, `(?i)\bfails?\b`, `(?i)\bfuture work\b`)},
	{CatMethod, compile(
		`(?i)\bwe propose\b`, `(?i)\bwe introduce\b`, `(?i)\bwe present\b`,
		`(?i)\bwe design\b`, `(?i)\bour (?:method|approach|model|algorithm)\b`,
		`(?i)\balgorithm\b`, `(?i)\bmethod(?:ology)?\b`)},
	{CatResult, compile(
		`(?i)\bresults?\b`, `(?i)\bwe achieve\b`, `(?i)\boutperform`,
		`(?i)\baccuracy\b`, `(?i)\bperformance\b`, `(?i)\bimprov`, `(?i)\d+(?:\.\d+)?\s*%`)},
	{CatEvidence, compile(
		`(?i)\bwe show\b`, `(?i)\bwe observe\b`, `(?i)\bdemonstrat`,
		`(?i)\bexperiments?\b`, `(?i)\bevidence\b`, `(?i)\bevaluat`)},
	{CatMain, compile(
		`(?i)\bprov(?:e|es|ed|ing|en)\b`, `(?i)\bsolv(?:e|es|ed|ing)\b`,
		`(?i)\bestablish`, `(?i)\bsignificant`, `(?i)\brobust\b`)},
}

var supportRe = regexp.MustCompile(
	`(?i)\d|\btable\b|\bfigure\b|\bfig\.\b|\[\d+\]|\bp\s*[<=]|\bconfidence interval\b|\bappendix\b`)

const minSentenceLen = 12

var assertiveSections = []string{"abstract", "introduction", "conclusion"}

// SplitSentences splits text into sentences deterministically.
func SplitSentences(text string) []string {
	collapsed := strings.Join(strings.Fields(text), " ")
	if collapsed == "" {
		return nil
	}
	var out []string
	var b strings.Builder
	runes := []rune(collapsed)
	for i, r := range runes {
		b.WriteRune(r)
		if r == '.' || r == '!' || r == '?' {
			if i+1 >= len(runes) || runes[i+1] == ' ' {
				if s := strings.TrimSpace(b.String()); s != "" {
					out = append(out, s)
				}
				b.Reset()
			}
		}
	}
	if s := strings.TrimSpace(b.String()); s != "" {
		out = append(out, s)
	}
	return out
}

// ClassifySentence returns the category of a sentence, or "" if none.
func ClassifySentence(sentence string) string {
	for _, g := range categorySignals {
		for _, re := range g.res {
			if re.MatchString(sentence) {
				return g.category
			}
		}
	}
	return ""
}

// OverclaimTermsIn returns overclaim terms present in text, in canonical order.
func OverclaimTermsIn(text string) []string {
	out := []string{}
	for _, term := range OverclaimTerms {
		if overclaimRes[term].MatchString(text) {
			out = append(out, term)
		}
	}
	return out
}

// HasInlineSupport reports whether a sentence carries an inline support signal.
func HasInlineSupport(sentence string) bool {
	return supportRe.MatchString(sentence)
}

// ExtractClaims extracts and classifies claims from a parsed paper.
func ExtractClaims(p ParsedPaper) []Claim {
	claims := []Claim{}
	counter := 0
	for _, sec := range p.Sections {
		secLow := strings.ToLower(sec.Title)
		assertive := false
		for _, k := range assertiveSections {
			if strings.Contains(secLow, k) {
				assertive = true
				break
			}
		}
		for _, sentence := range SplitSentences(sec.Text) {
			if utf8.RuneCountInString(sentence) < minSentenceLen {
				continue
			}
			category := ClassifySentence(sentence)
			if category == "" {
				if !assertive {
					continue
				}
				category = CatMain
			}
			counter++
			claims = append(claims, Claim{
				ClaimID:  claimID(counter),
				Category: category,
				Section:  sec.Title,
				Text:     sentence,
			})
		}
	}
	return claims
}

func claimID(n int) string {
	s := itoa3(n)
	return "C" + s
}

// itoa3 zero-pads to at least 3 digits (matches Python's C{n:03d}).
func itoa3(n int) string {
	digits := ""
	if n == 0 {
		digits = "0"
	}
	for n > 0 {
		digits = string(rune('0'+n%10)) + digits
		n /= 10
	}
	for len(digits) < 3 {
		digits = "0" + digits
	}
	return digits
}
