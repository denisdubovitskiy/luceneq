package luceneq

import (
	"strings"
	"unicode"
)

// Matcher определяет интерфейс для проверки совпадения текста.
type Matcher interface {
	// Match проверяет, соответствует ли текст условию запроса.
	Match(text string) bool
}

// Проверяем, что все QueryNode реализуют Matcher.
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
// Словом считается последовательность Unicode-букв/цифр или '_'.
func splitWords(text string) []string {
	var words []string
	var current strings.Builder

	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
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

// matchWildcard проверяет термин с wildcards (? и *)
// с помощью алгоритма динамического программирования.
//
// Алгоритм строит DP-таблицу dp[patternLen+1][textLen+1], где:
//   - dp[i][j] = true, если prefix паттерна длины i совпадает с prefix текста длины j
//   - Символ ? совпадает с любым одиночным символом
//   - Символ * совпадает с последовательностью любых символов (включая пустую)
//
// Переходы:
//   - pattern[i-1] == '?' или текст[j-1] == pattern[i-1]: берём dp[i-1][j-1]
//   - pattern[i-1] == '*': dp[i-1][j] (пропустить *) || dp[i][j-1] (* совпадает с символом)
//
// Временная сложность: O(patternLen × textLen)
func matchWildcard(pattern, text string) bool {
	patternLen := len(pattern)
	textLen := len(text)

	if patternLen == 0 {
		return textLen == 0
	}

	// DP таблица
	dp := make([][]bool, patternLen+1)
	for i := range dp {
		dp[i] = make([]bool, textLen+1)
	}

	dp[0][0] = true

	// Обработка * в начале
	for i := 1; i <= patternLen; i++ {
		if pattern[i-1] == '*' {
			dp[i][0] = dp[i-1][0]
		} else {
			break
		}
	}

	for i := 1; i <= patternLen; i++ {
		for j := 1; j <= textLen; j++ {
			switch pattern[i-1] {
			case '?', text[j-1]:
				dp[i][j] = dp[i-1][j-1]
			case '*':
				dp[i][j] = dp[i-1][j] || dp[i][j-1]
			}
		}
	}

	return dp[patternLen][textLen]
}

// levenshteinDistance вычисляет расстояние Левенштейна между двумя строками
// с оптимизацией по памяти — используются только две строки prev и curr.
//
// Расстояние Левенштейна — минимальное количество односимвольных операций
// (вставка, удаление, замена), необходимых для превращения строки a в строку b.
//
// Алгоритм:
//   - prev[j] — расстояние между a[0..i-1] и b[0..j-1]
//   - curr[j] — расстояние между a[0..i] и b[0..j-1]
//   - cost = 0, если символы совпадают, иначе 1
//   - curr[j] = min(curr[j-1]+1, prev[j]+1, prev[j-1]+cost)
//     - curr[j-1]+1 — вставка
//     - prev[j]+1 — удаление
//     - prev[j-1]+cost — замена (или совпадение)
//
// Временная сложность: O(len(a) × len(b))
// Пространственная сложность: O(len(b)) вместо O(len(a) × len(b))
func levenshteinDistance(a, b string) int {
	lenA := len(a)
	lenB := len(b)

	if lenA == 0 {
		return lenB
	}
	if lenB == 0 {
		return lenA
	}

	prev := make([]int, lenB+1)
	curr := make([]int, lenB+1)

	for j := 0; j <= lenB; j++ {
		prev[j] = j
	}

	for i := 1; i <= lenA; i++ {
		curr[0] = i
		for j := 1; j <= lenB; j++ {
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

	return prev[lenB]
}

// containsConsecutiveWords проверяет, идут ли слова из words подряд в тексте.
//
// Алгоритм:
//   1. Разбивает текст на слова через splitWords
//   2. Перебирает все подпоследовательности длины wordsLen
//   3. Для каждой подпоследовательности проверяет посимвольное совпадение
//   4. При полном совпадении возвращает true
//
// Временная сложность: O((textWordsLen - wordsLen + 1) × wordsLen)
func containsConsecutiveWords(text string, words []string) bool {
	textWords := splitWords(text)
	textWordsLen := len(textWords)
	wordsLen := len(words)

	if textWordsLen < wordsLen {
		return false
	}

	for i := 0; i <= textWordsLen-wordsLen; i++ {
		match := true
		for j := 0; j < wordsLen; j++ {
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

// containsPhraseWithSlop проверяет, содержит ли текст все слова из words
// в правильном порядке с допустимым смещением (slop) между словами.
//
// Алгоритм:
//   1. Разбивает текст на слова
//   2. Проверяет, что в тексте достаточно слов для фразы
//   3. Делегирует рекурсивный поиск в findPhraseRecursively
//   4. Перебирает все возможные позиции для каждого слова
//   5. Проверяет, чтобы gap между соседними словами не превышал slop
//   6. При нахождении полного совпадения возвращает true
//
// Временная сложность: в худшем случае экспоненциальная, но на практике
// ограничена размером текста и slop.
func containsPhraseWithSlop(text string, words []string, slop int) bool {
	textWords := splitWords(text)
	textWordsLen := len(textWords)
	wordsLen := len(words)

	if textWordsLen < wordsLen {
		return false
	}

	// Перебираем все комбинации позиций
	return findPhraseRecursively(textWords, words, 0, 0, slop, 0)
}

// findPhraseRecursively рекурсивно ищет фразу с учётом slop.
//
// Алгоритм:
//   1. Базовый случай: если все слова найдены (wordIdx == len(words)) — возвращает true
//   2. Тупик: если кончился текст или слов недостаточно — возвращает false
//   3. Оптимизация: проверяет, осталось ли достаточно слов в тексте
//   4. Для каждого слова ищет все совпадающие позиции в тексте
//   5. Проверяет gap между позициями — если > slop, пропускает
//   6. Рекурсивно продолжает поиск для следующего слова
//
// Параметры:
//   - textWords: слова текста
//   - words: слова фразы для поиска
//   - textIdx: текущая позиция в словах текста
//   - wordIdx: текущее слово фразы, которое ищем
//   - slop: максимальный допустимый зазор между словами
//   - prevTextIdx: позиция предыдущего найденного слова в тексте
func findPhraseRecursively(textWords, words []string, textIdx, wordIdx, slop, prevTextIdx int) bool {
	if wordIdx == len(words) {
		return true
	}
	if textIdx >= len(textWords) {
		return false
	}

	// Проверяем, осталось ли достаточно слов
	if len(textWords)-textIdx < len(words)-wordIdx {
		return false
	}

	for i := textIdx; i <= len(textWords)-(len(words)-wordIdx); i++ {
		if textWords[i] == words[wordIdx] {
			// Проверяем допустимый slop
			if wordIdx > 0 {
				gap := i - prevTextIdx - 1
				if gap > slop {
					continue
				}
			}
			if findPhraseRecursively(textWords, words, i+1, wordIdx+1, slop, i) {
				return true
			}
		}
	}

	return false
}
