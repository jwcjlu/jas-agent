package graphrag

import (
	"math"
	"strings"
	"unicode"
)

var stopWords = map[string]struct{}{
	"the": {}, "a": {}, "an": {}, "and": {}, "or": {}, "is": {}, "are": {},
	"am": {}, "to": {}, "of": {}, "in": {}, "on": {}, "for": {}, "with": {},
	"this": {}, "that": {}, "these": {}, "those": {}, "be": {}, "as": {},
	"by": {}, "it": {}, "from": {}, "at": {}, "有": {}, "的": {}, "了": {},
	"和": {}, "是": {}, "在": {}, "与": {}, "及": {}, "并": {},
}

func tokenize(text string) []string {
	if text == "" {
		return nil
	}
	text = strings.ToLower(text)
	var tokens []string
	var current strings.Builder

	flushCurrent := func() {
		if current.Len() == 0 {
			return
		}
		token := current.String()
		current.Reset()
		if len(token) < 2 {
			return
		}
		if _, stop := stopWords[token]; stop {
			return
		}
		tokens = append(tokens, token)
	}

	for _, r := range text {
		switch {
		case unicode.IsDigit(r):
			current.WriteRune(r)
		case unicode.IsLetter(r):
			current.WriteRune(r)
		case unicode.Is(unicode.Han, r):
			flushCurrent()
			token := string(r)
			if _, stop := stopWords[token]; !stop {
				tokens = append(tokens, token)
			}
		default:
			flushCurrent()
		}
	}
	flushCurrent()
	return tokens
}

func semanticSimilarity(tokensA, tokensB []string) float64 {
	if len(tokensA) == 0 || len(tokensB) == 0 {
		return 0
	}
	freqA := termFrequency(tokensA)
	freqB := termFrequency(tokensB)

	var dot float64
	var normA float64
	var normB float64

	for token, fa := range freqA {
		normA += fa * fa
		if fb, ok := freqB[token]; ok {
			dot += fa * fb
		}
	}
	for _, fb := range freqB {
		normB += fb * fb
	}
	if dot == 0 || normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

func termFrequency(tokens []string) map[string]float64 {
	freq := make(map[string]float64, len(tokens))
	for _, token := range tokens {
		if token == "" {
			continue
		}
		freq[token]++
	}
	return freq
}
