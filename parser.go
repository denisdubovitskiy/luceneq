package luceneq

import (
	"fmt"
	"strings"
)

// tokenType определяет тип токена лексера.
type tokenType int

const (
	tokTerm tokenType = iota
	tokPhrase
	tokOperator
	tokPlus
	tokMinus
	tokLParen
	tokRParen
	tokBracketL
	tokBracketR
	tokCurlyL
	tokCurlyR
	tokTO
	tokCaret
	tokTilde
	tokEOF
)

// token представляет один токен в запросе.
type token struct {
	typ           tokenType
	value         string
	Wildcard      bool
	Fuzzy         bool
	FuzzyDistance int
}

// ParserState хранит состояние парсера.
type ParserState struct {
	tokens []token
	pos    int
}

// Parser определяет интерфейс для парсинга Lucene-запросов.
type Parser interface {
	// ParseQuery разбивает строку запроса на токены и строит
	// дерево выражений для последующей фильтрации текста.
	// Возвращает Matcher, который можно использовать для проверки текста.
	ParseQuery(query string) (Matcher, error)
}

// parser реализует парсер Lucene-запросов.
type parser struct{}

// NewParser создаёт новый парсер запросов.
func NewParser() Parser {
	return &parser{}
}

// ParseQuery парсит Lucene-запрос и возвращает Matcher для фильтрации текста.
func (p *parser) ParseQuery(query string) (Matcher, error) {
	tokens, err := tokenize(query)
	if err != nil {
		return nil, fmt.Errorf("токенизация: %w", err)
	}

	ps := &ParserState{
		tokens: tokens,
	}

	node, err := ps.parseOrExpression()
	if err != nil {
		return nil, err
	}

	if ps.pos < len(tokens) {
		return nil, fmt.Errorf("неожиданный токен: %v", tokens[ps.pos])
	}

	if node == nil {
		return &ConstantQuery{Value: true}, nil
	}

	return node.(Matcher), nil
}

// tokenize разбивает запрос на токены.
func tokenize(input string) ([]token, error) {
	lex := newLexer(input)
	tokens := make([]token, 0, 16)

	for {
		tok := lex.scan()
		if tok.typ == tokEOF {
			break
		}
		tokens = append(tokens, tok)
	}

	for i, tok := range tokens {
		if tok.typ == tokEOF && i < len(tokens)-1 {
			return nil, fmt.Errorf("незакрытая кавычка")
		}
	}

	return tokens, nil
}

// parseOrExpression разбирает выражение с учётом приоритета OR.
//
// Приоритеты операторов (от низшего к высшему):
//   1. OR / || (низший приоритет)
//   2. AND / &&
//   3. NOT / !
//   4. + / - (унарные)
//   5. Термы, фразы, скобки (высший приоритет)
//
// Алгоритм:
//   1. Рекурсивно вызывает parseAndExpression для левого операнда
//   2. Пока видит оператор OR, читает правый операнд
//   3. Строит AST: (left OR right)
//
// Это обеспечивает правильный приоритет: a OR b AND c = a OR (b AND c)
func (ps *ParserState) parseOrExpression() (QueryNode, error) {
	left, err := ps.parseAndExpression()
	if err != nil {
		return nil, err
	}
	for ps.pos < len(ps.tokens) && isOrOperator(ps.tokens[ps.pos]) {
		ps.pos++
		right, err := ps.parseAndExpression()
		if err != nil {
			return nil, err
		}
		left = &BooleanQuery{
			Operator: BooleanOR,
			Clauses:  []QueryNode{left, right},
		}
	}
	return left, nil
}

// isOrOperator проверяет, является ли токен оператором OR.
func isOrOperator(tok token) bool {
	return tok.typ == tokOperator && (tok.value == "OR" || tok.value == "||")
}

// parseAndExpression разбирает выражение с учётом приоритета AND.
//
// Алгоритм (низкоуровневый рекурсивный спуск):
//   1. Считывает левый операнд через parseNotExpression
//   2. В цикле проверяет текущий токен:
//      - AND / && — явно, объединяет через BooleanAND
//      - NOT / ! — преобразует в BooleanNOT, затем AND
//      - - — prohibited, преобразует в BooleanMUSTNOT
//      - Term/Phrase — неявный AND (a b = a AND b)
//      - + — required term, преобразует в BooleanMUST
//      - ( — неявный AND перед группой
//   3. При любом другом токене завершает цикл
//   4. Возвращает накопленное выражение
//
// Поддерживает левую рекурсию: a AND b AND c = ((a AND b) AND c)
func (ps *ParserState) parseAndExpression() (QueryNode, error) {
	left, err := ps.parseNotExpression()
	if err != nil {
		return nil, err
	}

	for ps.pos < len(ps.tokens) {
		tok := ps.tokens[ps.pos]
		if tok.typ == tokOperator && tok.value == "AND" {
			ps.pos++
			right, err := ps.parseNotExpression()
			if err != nil {
				return nil, err
			}
			left = &BooleanQuery{
				Operator: BooleanAND,
				Clauses:  []QueryNode{left, right},
			}
		} else if tok.typ == tokOperator && tok.value == "&&" {
			ps.pos++
			right, err := ps.parseNotExpression()
			if err != nil {
				return nil, err
			}
			left = &BooleanQuery{
				Operator: BooleanAND,
				Clauses:  []QueryNode{left, right},
			}
		} else if tok.typ == tokTerm || tok.typ == tokPhrase {
			right, err := ps.parseNotExpression()
			if err != nil {
				return nil, err
			}
			left = &BooleanQuery{
				Operator: BooleanAND,
				Clauses:  []QueryNode{left, right},
			}
		} else if tok.typ == tokOperator && tok.value == "NOT" {
			ps.pos++
			right, err := ps.parseNotExpression()
			if err != nil {
				return nil, err
			}
			notNode := &BooleanQuery{
				Operator: BooleanNOT,
				Clauses:  []QueryNode{right},
			}
			left = &BooleanQuery{
				Operator: BooleanAND,
				Clauses:  []QueryNode{left, notNode},
			}
		} else if tok.typ == tokOperator && tok.value == "!" {
			ps.pos++
			right, err := ps.parsePrimary()
			if err != nil {
				return nil, err
			}
			notNode := &BooleanQuery{
				Operator: BooleanNOT,
				Clauses:  []QueryNode{right},
			}
			left = &BooleanQuery{
				Operator: BooleanAND,
				Clauses:  []QueryNode{left, notNode},
			}
		} else if tok.typ == tokMinus {
			ps.pos++
			right, err := ps.parsePrimary()
			if err != nil {
				return nil, err
			}
			prohibited := &BooleanQuery{
				Operator: BooleanMUSTNOT,
				Clauses:  []QueryNode{right},
			}
			left = &BooleanQuery{
				Operator: BooleanAND,
				Clauses:  []QueryNode{left, prohibited},
			}
		} else if tok.typ == tokLParen {
			// Неявный AND перед группой
			right, err := ps.parsePrimary()
			if err != nil {
				return nil, err
			}
			left = &BooleanQuery{
				Operator: BooleanAND,
				Clauses:  []QueryNode{left, right},
			}
		} else if tok.typ == tokPlus {
			// Неявный AND + required
			ps.pos++
			right, err := ps.parsePrimary()
			if err != nil {
				return nil, err
			}
			required := &BooleanQuery{
				Operator: BooleanMUST,
				Clauses:  []QueryNode{right},
			}
			left = &BooleanQuery{
				Operator: BooleanAND,
				Clauses:  []QueryNode{left, required},
			}
		} else {
			break
		}
	}
	return left, nil
}

// parseNotExpression разбирает NOT и унарные операторы (+, -).
//
// Алгоритм:
//   1. Проверяет текущий токен:
//      - NOT — рекурсивно вызывает parseNotExpression для правого операнда
//      - ! — вызывает parsePrimary для правого операнда
//      - + — required, вызывает parsePrimary
//      - - — prohibited, вызывает parsePrimary
//   2. Если унарных операторов нет — вызывает parsePrimary
//
// Поддерживает цепочки: a NOT NOT b = NOT(NOT(a, b))
func (ps *ParserState) parseNotExpression() (QueryNode, error) {
	tok := ps.currentToken()

	switch tok.typ {
	case tokOperator:
		if tok.value == "NOT" {
			ps.pos++
			right, err := ps.parseNotExpression()
			if err != nil {
				return nil, err
			}
			return &BooleanQuery{
				Operator: BooleanNOT,
				Clauses:  []QueryNode{right},
			}, nil
		}
		if tok.value == "!" {
			ps.pos++
			expr, err := ps.parsePrimary()
			if err != nil {
				return nil, err
			}
			return &BooleanQuery{
				Operator: BooleanMUSTNOT,
				Clauses:  []QueryNode{expr},
			}, nil
		}
	case tokPlus:
		ps.pos++
		expr, err := ps.parsePrimary()
		if err != nil {
			return nil, err
		}
		return &BooleanQuery{
			Operator: BooleanMUST,
			Clauses:  []QueryNode{expr},
		}, nil
	case tokMinus:
		ps.pos++
		expr, err := ps.parsePrimary()
		if err != nil {
			return nil, err
		}
		return &BooleanQuery{
			Operator: BooleanMUSTNOT,
			Clauses:  []QueryNode{expr},
		}, nil
	}

	return ps.parsePrimary()
}

// parsePrimary разбирает первичные элементы запроса.
//
// Это высший уровень приоритета — термы, фразы и группировки.
//
// Алгоритм:
//   1. Term → buildTermQuery (обработка wildcard, fuzzy)
//   2. Phrase → PhraseQuery (разбивка по словам)
//   3. (expr) — рекурсивный вызов parseOrExpression, проверка закрывающей скобки
//   4. [a TO b] — RangeQuery с включёнными границами
//   5. {a TO b} — RangeQuery с исключёнными границами
//   6. Неизвестный токен → ConstantQuery{true} (fallback)
//
// Обработка ошибок:
//   - Незакрытая скобка → ошибка
//   - Отсутствие TO в range query → ошибка
//   - Незакрытый range → ошибка
func (ps *ParserState) parsePrimary() (QueryNode, error) {
	tok := ps.currentToken()

	switch tok.typ {
	case tokTerm:
		ps.pos++
		return buildTermQuery(tok.value, tok.Wildcard, tok.Fuzzy, tok.FuzzyDistance), nil

	case tokPhrase:
		ps.pos++
		words := strings.Fields(tok.value)
		return &PhraseQuery{Words: words}, nil

	case tokLParen:
		ps.pos++
		expr, err := ps.parseOrExpression()
		if err != nil {
			return nil, err
		}
		if ps.pos >= len(ps.tokens) || ps.tokens[ps.pos].typ != tokRParen {
			return nil, fmt.Errorf("незакрытая скобка")
		}
		ps.pos++
		return expr, nil

	case tokBracketL:
		ps.pos++
		lower := ps.parseRangeTerm()
		if ps.pos >= len(ps.tokens) || ps.tokens[ps.pos].typ != tokTO {
			return nil, fmt.Errorf("ошибка в range query: ожидается TO")
		}
		ps.pos++
		upper := ps.parseRangeTerm()
		if ps.pos >= len(ps.tokens) || ps.tokens[ps.pos].typ != tokBracketR {
			return nil, fmt.Errorf("ошибка в range query: ожидается ]")
		}
		ps.pos++
		return &RangeQuery{
			Lower:        lower,
			Upper:        upper,
			IncludeLower: true,
			IncludeUpper: true,
		}, nil

	case tokCurlyL:
		ps.pos++
		lower := ps.parseRangeTerm()
		if ps.pos >= len(ps.tokens) || ps.tokens[ps.pos].typ != tokTO {
			return nil, fmt.Errorf("ошибка в range query: ожидается TO")
		}
		ps.pos++
		upper := ps.parseRangeTerm()
		if ps.pos >= len(ps.tokens) || ps.tokens[ps.pos].typ != tokCurlyR {
			return nil, fmt.Errorf("ошибка в range query: ожидается }")
		}
		ps.pos++
		return &RangeQuery{
			Lower:        lower,
			Upper:        upper,
			IncludeLower: false,
			IncludeUpper: false,
		}, nil
	}

	return &ConstantQuery{Value: true}, nil
}

// parseRangeTerm извлекает термин для range queries.
func (ps *ParserState) parseRangeTerm() string {
	if ps.pos >= len(ps.tokens) {
		return ""
	}
	tok := ps.tokens[ps.pos]
	if tok.typ == tokTerm {
		ps.pos++
		return tok.value
	}
	return ""
}

// currentToken возвращает текущий токен.
func (ps *ParserState) currentToken() token {
	if ps.pos < len(ps.tokens) {
		return ps.tokens[ps.pos]
	}
	return token{typ: tokEOF}
}

// buildTermQuery создаёт TermQuery из параметров.
//
// Алгоритм:
//   1. Создаёт базовый TermQuery с термом
//   2. Если wildcard=true — устанавливает флаг Wildcard
//   3. Если fuzzy=true — устанавливает Fuzzy и FuzzyDistance
//   4. Если fuzzyDistance=0 (авто) — ставит значение по умолчанию 2
//
// Ограничение: fuzzyDistance никогда не превышает 2
func buildTermQuery(term string, wildcard, fuzzy bool, fuzzyDistance int) QueryNode {
	tq := &TermQuery{Term: term}
	if wildcard {
		tq.Wildcard = true
	}
	if fuzzy {
		tq.Fuzzy = true
		tq.FuzzyDistance = fuzzyDistance
		if fuzzyDistance == 0 {
			tq.FuzzyDistance = 2
		}
	}
	return tq
}

// lexer разбивает запрос на токены.
type lexer struct {
	runes   []rune
	pos     int
	length  int
	current rune
}

func newLexer(input string) *lexer {
	runes := []rune(input)
	lex := &lexer{
		runes:  runes,
		length: len(runes),
	}
	if lex.length > 0 {
		lex.current = lex.runes[0]
	}
	return lex
}

func (lex *lexer) next() {
	lex.pos++
	if lex.pos < lex.length {
		lex.current = lex.runes[lex.pos]
	} else {
		lex.current = 0
	}
}

func isSpace(r rune) bool {
	return r == ' ' || r == '\t' || r == '\n' || r == '\r'
}

func isDelimiter(r rune) bool {
	switch r {
	case '(', ')', '[', ']', '{', '}', '^', '+', '-', '"', ':', '\\', '|', '&':
		return true
	}
	return isSpace(r)
}

func (lex *lexer) skipWhitespace() {
	for lex.pos < lex.length && isSpace(lex.current) {
		lex.next()
	}
}

// scan сканирует входную строку и возвращает следующий токен.
//
// Алгоритм лексера:
//   1. Пропускает пробельные символы
//   2. Проверяет специальные символы: ( ) [ ] { } ^ ~ + -
//   3. Распознаёт составные операторы: || → OR, && → AND
//   4. Вызывает scanPhrase для строк в кавычках
//   5. Вызывает scanTO/scanKeyword для ключевых слов TO, AND, OR, NOT
//   6. Вызывает scanTerm для термов (с обработкой экранирования)
//   7. Возвращает EOF при конце ввода
//
// Экранирование: \x превращается в терм x
func (lex *lexer) scan() token {
	lex.skipWhitespace()

	if lex.pos >= lex.length {
		return token{typ: tokEOF}
	}

	switch lex.current {
	case '(':
		lex.next()
		return token{typ: tokLParen}
	case ')':
		lex.next()
		return token{typ: tokRParen}
	case '[':
		lex.next()
		return token{typ: tokBracketL}
	case ']':
		lex.next()
		return token{typ: tokBracketR}
	case '{':
		lex.next()
		return token{typ: tokCurlyL}
	case '}':
		lex.next()
		return token{typ: tokCurlyR}
	case '^':
		lex.next()
		return token{typ: tokCaret}
	case '~':
		lex.next()
		return token{typ: tokTilde}
	case '+':
		lex.next()
		return token{typ: tokPlus}
	case '-':
		lex.next()
		return token{typ: tokMinus}
	case '|':
		lex.next()
		if lex.pos < lex.length && lex.current == '|' {
			lex.next()
			return token{typ: tokOperator, value: "OR"}
		}
		return token{typ: tokOperator, value: "|"}
	case '&':
		lex.next()
		if lex.pos < lex.length && lex.current == '&' {
			lex.next()
			return token{typ: tokOperator, value: "AND"}
		}
		return token{typ: tokOperator, value: "&"}
	case '!':
		lex.next()
		return token{typ: tokOperator, value: "!"}
	case '"':
		return lex.scanPhrase()
	case 'T', 't':
		return lex.scanTO()
	case 'A', 'a':
		return lex.scanKeyword("AND")
	case 'O', 'o':
		return lex.scanKeyword("OR")
	case 'N', 'n':
		return lex.scanKeyword("NOT")
	case '\\':
		lex.next()
		if lex.pos >= lex.length {
			return token{typ: tokTerm, value: "\\"}
		}
		esc := lex.current
		lex.next()
		return token{typ: tokTerm, value: string(esc)}
	default:
		return lex.scanTerm()
	}
}

func (lex *lexer) scanPhrase() token {
	lex.next()
	var builder strings.Builder

	for lex.pos < lex.length && lex.current != '"' {
		if lex.current == '\\' {
			lex.next()
			if lex.pos < lex.length && lex.current != 0 {
				builder.WriteString(string(lex.current))
				lex.next()
			}
			continue
		}
		builder.WriteRune(lex.current)
		lex.next()
	}

	if lex.pos >= lex.length {
		return token{typ: tokEOF}
	}

	lex.next()
	return token{typ: tokPhrase, value: builder.String()}
}

func (lex *lexer) scanTerm() token {
	var builder strings.Builder

	for lex.pos < lex.length && !isSpace(lex.current) && !isDelimiter(lex.current) {
		if lex.current == '\\' {
			lex.next()
			if lex.pos < lex.length && lex.current != 0 {
				builder.WriteString(string(lex.current))
				lex.next()
			}
			continue
		}
		builder.WriteRune(lex.current)
		lex.next()
	}

	value := builder.String()
	if len(value) == 0 {
		return token{typ: tokEOF}
	}

	return lex.parseTermModifiers(value)
}

// parseTermModifiers обрабатывает модификаторы термов: fuzzy (~) и wildcards (?, *).
//
// Алгоритм:
//   1. Проверяет окончание ~ для fuzzy поиска:
//      - fuzzyDistance из цифры после ~ (0-2, по умолчанию 2)
//      - Удаляет ~ из значения терма
//   2. Проверяет наличие ? или * в терме для wildcards
//   3. Возвращает токен с установленными флагами
//
// Примеры:
//   - roam~ → fuzzy, distance=2
//   - roam~1 → fuzzy, distance=1
//   - test* → wildcard
//   - te?t → wildcard
func (lex *lexer) parseTermModifiers(raw string) token {
	runes := []rune(raw)
	runesCount := len(runes)
	t := token{typ: tokTerm, value: raw}

	if runesCount > 0 && runes[runesCount-1] == '~' {
		t.Fuzzy = true
		t.value = string(runes[:runesCount-1])
		if runesCount > 1 && runes[runesCount-2] >= '0' && runes[runesCount-2] <= '9' {
			_, err := fmt.Sscanf(string(runes[runesCount-2:runesCount]), "%d", &t.FuzzyDistance)
			if err != nil {
				t.FuzzyDistance = 2
			} else if t.FuzzyDistance > 2 {
				t.FuzzyDistance = 2
			}
		}
		return t
	}

	for i := 0; i < runesCount; i++ {
		if runes[i] == '*' || runes[i] == '?' {
			t.Wildcard = true
			// Не убираем wildcards — они нужны для matchWildcard
			return t
		}
	}

	return t
}

func (lex *lexer) scanKeyword(expected string) token {
	var builder strings.Builder
	startIndex := lex.pos

	for lex.pos < lex.length {
		builder.WriteRune(lex.current)
		lex.next()
		if isSpace(lex.current) || isDelimiter(lex.current) {
			break
		}
	}

	value := strings.ToUpper(builder.String())
	if value == expected {
		return token{typ: tokOperator, value: value}
	}

	lex.pos = startIndex
	if lex.pos < lex.length {
		lex.current = lex.runes[lex.pos]
	}
	return lex.scanTerm()
}

func (lex *lexer) scanTO() token {
	var builder strings.Builder
	startIndex := lex.pos

	for lex.pos < lex.length {
		builder.WriteRune(lex.current)
		lex.next()
		if isSpace(lex.current) || isDelimiter(lex.current) {
			break
		}
	}

	value := strings.ToUpper(builder.String())
	if value == "TO" {
		return token{typ: tokTO}
	}

	lex.pos = startIndex
	if lex.pos < lex.length {
		lex.current = lex.runes[lex.pos]
	}
	return lex.scanTerm()
}
