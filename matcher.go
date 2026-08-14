package luceneq

import (
	"strings"
)

// Matcher определяет интерфейс для проверки совпадения текста.
type Matcher interface {
	// Match проверяет, соответствует ли текст условию запроса.
	Match(text string) bool
}

// Ensure that all QueryNodes implement Matcher.
var (
	_ Matcher = (*TermQuery)(nil)
	_ Matcher = (*PhraseQuery)(nil)
	_ Matcher = (*RangeQuery)(nil)
	_ Matcher = (*BooleanQuery)(nil)
	_ Matcher = (*ConstantQuery)(nil)
)

// Match проверяет, содержит ли текст данный термин.
// Учитывает wildcards и fuzzy search.
func (q *TermQuery) Match(text string) bool {
	lowerText := strings.ToLower(text)
	lowerTerm := strings.ToLower(q.Term)

	if q.Wildcard {
		// Ищем термин с wildcard в тексте
		words := splitWords(lowerText)
		for _, word := range words {
			if matchWildcard(lowerTerm, word) {
				return true
			}
		}
		return false
	}

	if q.Fuzzy {
		// Fuzzy search — ищем слово, близкое к искомому
		words := splitWords(lowerText)
		for _, word := range words {
			if levenshteinDistance(lowerTerm, word) <= q.FuzzyDistance {
				return true
			}
		}
		return false
	}

	// Точный поиск подстроки
	return strings.Contains(lowerText, lowerTerm)
}

// Match проверяет, содержит ли текст указанную фразу.
func (q *PhraseQuery) Match(text string) bool {
	if len(q.Words) == 0 {
		return true
	}

	lowerText := strings.ToLower(text)
	lowerWords := make([]string, len(q.Words))
	for i, w := range q.Words {
		lowerWords[i] = strings.ToLower(w)
	}

	// Slop = 0 — слова должны идти подряд в том же порядке
	if q.Slop == 0 {
		return containsConsecutiveWords(lowerText, lowerWords)
	}

	// Slop > 0 — слова могут быть разнесены
	return containsPhraseWithSlop(lowerText, lowerWords, q.Slop)
}

// Match проверяет, попадает ли текст в указанный диапазон.
func (q *RangeQuery) Match(text string) bool {
	cmp := strings.ToLower(text)

	// Проверяем границы
	if cmp < q.Lower || cmp > q.Upper {
		return false
	}
	if !q.IncludeLower && cmp == q.Lower {
		return false
	}
	if !q.IncludeUpper && cmp == q.Upper {
		return false
	}
	return true
}

// Match проверяет текст с учётом логической операции.
func (q *BooleanQuery) Match(text string) bool {
	switch q.Operator {
	case BooleanAND:
		for _, clause := range q.Clauses {
			if !clause.(Matcher).Match(text) {
				return false
			}
		}
		return true
	case BooleanOR:
		for _, clause := range q.Clauses {
			if clause.(Matcher).Match(text) {
				return true
			}
		}
		return false
	case BooleanNOT:
		if len(q.Clauses) < 1 {
			return true
		}
		return !q.Clauses[len(q.Clauses)-1].(Matcher).Match(text)
	case BooleanMUST:
		if len(q.Clauses) < 1 {
			return true
		}
		return q.Clauses[len(q.Clauses)-1].(Matcher).Match(text)
	case BooleanMUSTNOT:
		if len(q.Clauses) < 1 {
			return true
		}
		return !q.Clauses[len(q.Clauses)-1].(Matcher).Match(text)
	default:
		return false
	}
}

// Match возвращает константное значение.
func (q *ConstantQuery) Match(_ string) bool {
	return q.Value
}

// splitWords разбивает текст на слова (убирает пунктуацию).
func splitWords(text string) []string {
	var words []string
	var current strings.Builder

	for _, r := range text {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			current.WriteRune(r)
		} else {
			if current.Len() > 0 {
				words = append(words, current.String())
				current.Reset()
			}
		}
	}
	if current.Len() > 0 {
		words = append(words, current.String())
	}
	return words
}

// matchWildcard проверяет термин с wildcards (? и *).
func matchWildcard(pattern, text string) bool {
	pLen := len(pattern)
	tLen := len(text)

	if pLen == 0 {
		return tLen == 0
	}

	// DP таблица
	dp := make([][]bool, pLen+1)
	for i := range dp {
		dp[i] = make([]bool, tLen+1)
	}

	dp[0][0] = true

	// Обработка * в начале
	for i := 1; i <= pLen; i++ {
		if pattern[i-1] == '*' {
			dp[i][0] = dp[i-1][0]
		} else {
			break
		}
	}

	for i := 1; i <= pLen; i++ {
		for j := 1; j <= tLen; j++ {
			switch pattern[i-1] {
			case '?', text[j-1]:
				dp[i][j] = dp[i-1][j-1]
			case '*':
				dp[i][j] = dp[i-1][j] || dp[i][j-1]
			}
		}
	}

	return dp[pLen][tLen]
}

// levenshteinDistance вычисляет расстояние Левенштейна между двумя строками.
func levenshteinDistance(a, b string) int {
	la := len(a)
	lb := len(b)

	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}

	prev := make([]int, lb+1)
	curr := make([]int, lb+1)

	for j := 0; j <= lb; j++ {
		prev[j] = j
	}

	for i := 1; i <= la; i++ {
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			curr[j] = min(
				curr[j-1]+1,
				prev[j]+1,
				prev[j-1]+cost,
			)
		}
		prev, curr = curr, prev
	}

	return prev[lb]
}

// containsConsecutiveWords проверяет, идут ли слова подряд в тексте.
func containsConsecutiveWords(text string, words []string) bool {
	textWords := splitWords(text)
	tLen := len(textWords)
	wLen := len(words)

	if tLen < wLen {
		return false
	}

	for i := 0; i <= tLen-wLen; i++ {
		match := true
		for j := 0; j < wLen; j++ {
			if textWords[i+j] != words[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// containsPhraseWithSlop проверяет фразу с допустимым смещением слов.
func containsPhraseWithSlop(text string, words []string, slop int) bool {
	textWords := splitWords(text)
	tLen := len(textWords)
	wLen := len(words)

	if tLen < wLen {
		return false
	}

	// Перебираем все комбинации позиций
	return findPhraseRecursively(textWords, words, 0, 0, slop, 0)
}

// findPhraseRecursively рекурсивно ищет фразу с учётом slop.
func findPhraseRecursively(textWords, words []string, tIdx, wIdx, slop, prevTIdx int) bool {
	if wIdx == len(words) {
		return true
	}
	if tIdx >= len(textWords) {
		return false
	}

	// Проверяем, осталось ли достаточно слов
	if len(textWords)-tIdx < len(words)-wIdx {
		return false
	}

	for i := tIdx; i <= len(textWords)-(len(words)-wIdx); i++ {
		if textWords[i] == words[wIdx] {
			// Проверяем допустимый slop
			if wIdx > 0 {
				gap := i - prevTIdx - 1
				if gap > slop {
					continue
				}
			}
			if findPhraseRecursively(textWords, words, i+1, wIdx+1, slop, i) {
				return true
			}
		}
	}

	return false
}
