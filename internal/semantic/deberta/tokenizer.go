package deberta

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode"
)

// WordPiece tokenizer with greedy longest-match-first segmentation.
type WordPiece struct {
	vocab           map[string]struct{}
	unk             string
	doLowerCase     bool
	maxCharsPerWord int
}

// TokenCount returns the number of wordpiece tokens for a given text.
func (wp *WordPiece) TokenCount(text string) int {
	tokens := wp.tokenize(text)
	return len(tokens)
}

func (wp *WordPiece) tokenize(text string) []string {
	// Basic tokenization: whitespace and punctuation splitting + lowercasing
	basic := basicTokens(text, wp.doLowerCase)
	out := make([]string, 0, len(basic))
	for _, w := range basic {
		out = append(out, wp.wordpiece(w)...)
	}
	return out
}

func (wp *WordPiece) wordpiece(word string) []string {
	if word == "" {
		return nil
	}
	// Guard against exceptionally long words
	n := 0
	for range word { // rune count
		n++
	}
	if wp.maxCharsPerWord > 0 && n > wp.maxCharsPerWord {
		return []string{wp.unk}
	}
	// Greedy longest-match-first
	var (
		subTokens []string
		runes     = []rune(word)
		start    = 0
	)
	for start < len(runes) {
		end := len(runes)
		var cur string
		for end > start {
			sub := string(runes[start:end])
			if start > 0 {
				sub = "##" + sub
			}
			if _, ok := wp.vocab[sub]; ok {
				cur = sub
				break
			}
			end--
		}
		if cur == "" {
			return []string{wp.unk}
		}
		subTokens = append(subTokens, cur)
		start = end
	}
	return subTokens
}

// basicTokens splits text on whitespace and isolates punctuation as separate tokens.
func basicTokens(text string, doLower bool) []string {
	runes := []rune(text)
	buf := make([]rune, 0, len(runes))
	emit := func(tokens *[]string) {
		if len(buf) == 0 {
			return
		}
		s := string(buf)
		if doLower {
			s = strings.ToLower(s)
		}
		*tokens = append(*tokens, s)
		buf = buf[:0]
	}
	var tokens []string
	for _, r := range runes {
		if unicode.IsSpace(r) {
			emit(&tokens)
			continue
		}
		if isPunct(r) {
			emit(&tokens)
			// punctuation as separate token
			p := string(r)
			if doLower { p = strings.ToLower(p) }
			tokens = append(tokens, p)
			continue
		}
		buf = append(buf, r)
	}
	emit(&tokens)
	return tokens
}

func isPunct(r rune) bool {
	// Treat ASCII punctuation and Unicode P categories as punctuation
	if r >= 33 && r <= 47 { // !"#$%&'()*+,-./
		return true
	}
	if r >= 58 && r <= 64 { // :;<=>?@
		return true
	}
	if r >= 91 && r <= 96 { // [\]^_`
		return true
	}
	if r >= 123 && r <= 126 { // {|}~
		return true
	}
	return unicode.Is(unicode.Punct, r)
}

// One-time loader for WordPiece from environment-configured paths.
var (
	wpOnce      sync.Once
	wpTok       *WordPiece
	wpLoadError error
)

func getWordPieceTokenizer() *WordPiece {
	wpOnce.Do(func() {
		// Prefer explicit env vars
		if path := strings.TrimSpace(os.Getenv("PS_DEBERTA_TOKENIZER_JSON")); path != "" {
			if t, err := loadWordPieceFromTokenizerJSON(path); err == nil {
				wpTok = t
				return
			} else {
				wpLoadError = err
			}
		}
		if path := strings.TrimSpace(os.Getenv("PS_DEBERTA_VOCAB_FILE")); path != "" {
			if t, err := loadWordPieceFromVocabFile(path, defaultLowercase()); err == nil {
				wpTok = t
				return
			} else {
				wpLoadError = err
			}
		}
		// Fallback to conventional location in repo if available
		fallback := filepath.Join("assets", "tokenizers", "deberta-vocab.txt")
		if _, err := os.Stat(fallback); err == nil {
			if t, err2 := loadWordPieceFromVocabFile(fallback, defaultLowercase()); err2 == nil {
				wpTok = t
				return
			} else {
				wpLoadError = err2
			}
		}
	})
	return wpTok
}

func defaultLowercase() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("PS_DEBERTA_LOWERCASE")))
	if v == "false" || v == "0" || v == "no" {
		return false
	}
	return true
}

// loadWordPieceFromVocabFile reads a vocab.txt where each line contains a token.
func loadWordPieceFromVocabFile(path string, lower bool) (*WordPiece, error) {
	f, err := os.Open(path)
	if err != nil { return nil, err }
	defer f.Close()
	scanner := bufio.NewScanner(f)
	vocab := make(map[string]struct{}, 30000)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" { continue }
		vocab[line] = struct{}{}
	}
	if err := scanner.Err(); err != nil { return nil, err }
	if len(vocab) == 0 { return nil, errors.New("empty vocab file") }
	return &WordPiece{vocab: vocab, unk: "[UNK]", doLowerCase: lower, maxCharsPerWord: 100}, nil
}

// loadWordPieceFromTokenizerJSON parses a HF tokenizer.json that contains a WordPiece model.
func loadWordPieceFromTokenizerJSON(path string) (*WordPiece, error) {
	b, err := os.ReadFile(path)
	if err != nil { return nil, err }
	// Minimal schema covering common shapes
	var root map[string]any
	if err := json.Unmarshal(b, &root); err != nil { return nil, err }
	lower := defaultLowercase()
	if norm, ok := root["normalizer"].(map[string]any); ok {
		if v, ok2 := norm["lowercase"].(bool); ok2 { lower = v }
	}
	vocab := make(map[string]struct{}, 30000)
	// Try model.vocab first
	if model, ok := root["model"].(map[string]any); ok {
		if v, ok2 := model["vocab"].(map[string]any); ok2 {
			for tok := range v { vocab[tok] = struct{}{} }
		}
		if t2i, ok2 := model["token_to_id"].(map[string]any); ok2 && len(vocab) == 0 {
			for tok := range t2i { vocab[tok] = struct{}{} }
		}
	}
	// Some files may contain top-level vocab
	if len(vocab) == 0 {
		if v, ok := root["vocab"].(map[string]any); ok {
			for tok := range v { vocab[tok] = struct{}{} }
		}
	}
	if len(vocab) == 0 { return nil, errors.New("no vocab found in tokenizer json") }
	return &WordPiece{vocab: vocab, unk: "[UNK]", doLowerCase: lower, maxCharsPerWord: 100}, nil
}

// estimateTokens prefers the WordPiece tokenizer when available, else falls back.
func estimateTokens(text string) int {
	if wp := getWordPieceTokenizer(); wp != nil {
		return wp.TokenCount(text)
	}
	// Fallback approximation: whitespace split with simple subword emulation
	s := strings.TrimSpace(text)
	if s == "" { return 0 }
	count := 0
	fields := strings.FieldsFunc(s, func(r rune) bool {
		switch r {
		case ' ', '\n', '\t', '\r':
			return true
		case '.', ',', ';', ':', '!', '?', '(', ')', '[', ']', '{', '}', '\'', '"', '/', '\\', '-', '_', '#', '@', '$', '%', '^', '&', '*', '+', '=':
			return true
		default:
			return false
		}
	})
	for _, w := range fields {
		if w == "" { continue }
		base := 1
		transitions := 0
		for i := 1; i < len(w); i++ {
			c0 := w[i-1]
			c1 := w[i]
			if (c0 >= 'a' && c0 <= 'z' && c1 >= 'A' && c1 <= 'Z') ||
			   (c0 >= '0' && c0 <= '9' && (c1 >= 'A' && c1 <= 'Z' || c1 >= 'a' && c1 <= 'z')) {
				transitions++
			}
		}
		base += transitions
		for i := 0; i < len(w); i++ { if w[i] == '_' { base++ } }
		if extra := len(w) - 6; extra > 0 { base += extra / 6 }
		count += base
	}
	return count
}

