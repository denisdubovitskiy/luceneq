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
	lx := newLexer(input)
	tokens := make([]token, 0, 16)

	for {
		tok := lx.scan()
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

// parseNotExpression разбирает NOT и унарные операторы.
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

// parsePrimary разбирает первичные элементы.
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
	input   string
	pos     int
	length  int
	current rune
}

func newLexer(input string) *lexer {
	lx := &lexer{
		input:  input,
		length: len(input),
	}
	if lx.length > 0 {
		lx.current = rune(lx.input[0])
	}
	return lx
}

func (l *lexer) next() {
	l.pos++
	if l.pos < l.length {
		l.current = rune(l.input[l.pos])
	} else {
		l.current = 0
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

func (l *lexer) skipWhitespace() {
	for l.pos < l.length && isSpace(l.current) {
		l.next()
	}
}

func (l *lexer) scan() token {
	l.skipWhitespace()

	if l.pos >= l.length {
		return token{typ: tokEOF}
	}

	switch l.current {
	case '(':
		l.next()
		return token{typ: tokLParen}
	case ')':
		l.next()
		return token{typ: tokRParen}
	case '[':
		l.next()
		return token{typ: tokBracketL}
	case ']':
		l.next()
		return token{typ: tokBracketR}
	case '{':
		l.next()
		return token{typ: tokCurlyL}
	case '}':
		l.next()
		return token{typ: tokCurlyR}
	case '^':
		l.next()
		return token{typ: tokCaret}
	case '~':
		l.next()
		return token{typ: tokTilde}
	case '+':
		l.next()
		return token{typ: tokPlus}
	case '-':
		l.next()
		return token{typ: tokMinus}
	case '|':
		l.next()
		if l.pos < l.length && l.current == '|' {
			l.next()
			return token{typ: tokOperator, value: "OR"}
		}
		return token{typ: tokOperator, value: "|"}
	case '&':
		l.next()
		if l.pos < l.length && l.current == '&' {
			l.next()
			return token{typ: tokOperator, value: "AND"}
		}
		return token{typ: tokOperator, value: "&"}
	case '!':
		l.next()
		return token{typ: tokOperator, value: "!"}
	case '"':
		return l.scanPhrase()
	case 'T', 't':
		return l.scanTO()
	case 'A', 'a':
		return l.scanKeyword("AND")
	case 'O', 'o':
		return l.scanKeyword("OR")
	case 'N', 'n':
		return l.scanKeyword("NOT")
	case '\\':
		l.next()
		if l.pos >= l.length {
			return token{typ: tokTerm, value: "\\"}
		}
		esc := l.current
		l.next()
		return token{typ: tokTerm, value: string(esc)}
	default:
		return l.scanTerm()
	}
}

func (l *lexer) scanPhrase() token {
	l.next()
	var sb strings.Builder

	for l.pos < l.length && l.current != '"' {
		if l.current == '\\' {
			l.next()
			if l.pos < l.length && l.current != 0 {
				sb.WriteString(string(l.current))
				l.next()
			}
			continue
		}
		sb.WriteRune(l.current)
		l.next()
	}

	if l.pos >= l.length {
		return token{typ: tokEOF}
	}

	l.next()
	return token{typ: tokPhrase, value: sb.String()}
}

func (l *lexer) scanTerm() token {
	var sb strings.Builder

	for l.pos < l.length && !isSpace(l.current) && !isDelimiter(l.current) {
		if l.current == '\\' {
			l.next()
			if l.pos < l.length && l.current != 0 {
				sb.WriteString(string(l.current))
				l.next()
			}
			continue
		}
		sb.WriteRune(l.current)
		l.next()
	}

	value := sb.String()
	if len(value) == 0 {
		return token{typ: tokEOF}
	}

	return l.parseTermModifiers(value)
}

func (l *lexer) parseTermModifiers(raw string) token {
	runes := []rune(raw)
	n := len(runes)
	t := token{typ: tokTerm, value: raw}

	if n > 0 && runes[n-1] == '~' {
		t.Fuzzy = true
		t.value = string(runes[:n-1])
		if n > 1 && runes[n-2] >= '0' && runes[n-2] <= '9' {
			_, err := fmt.Sscanf(string(runes[n-2:n]), "%d", &t.FuzzyDistance)
			if err != nil {
				t.FuzzyDistance = 2
			} else if t.FuzzyDistance > 2 {
				t.FuzzyDistance = 2
			}
		}
		return t
	}

	for i := 0; i < n; i++ {
		if runes[i] == '*' || runes[i] == '?' {
			t.Wildcard = true
			// Не убираем wildcards — они нужны для matchWildcard
			return t
		}
	}

	return t
}

func (l *lexer) scanKeyword(expected string) token {
	var sb strings.Builder
	startPos := l.pos

	for l.pos < l.length {
		sb.WriteRune(l.current)
		l.next()
		if isSpace(l.current) || isDelimiter(l.current) {
			break
		}
	}

	value := strings.ToUpper(sb.String())
	if value == expected {
		return token{typ: tokOperator, value: value}
	}

	l.pos = startPos
	if l.pos < l.length {
		l.current = rune(l.input[l.pos])
	}
	return l.scanTerm()
}

func (l *lexer) scanTO() token {
	var sb strings.Builder
	startPos := l.pos

	for l.pos < l.length {
		sb.WriteRune(l.current)
		l.next()
		if isSpace(l.current) || isDelimiter(l.current) {
			break
		}
	}

	value := strings.ToUpper(sb.String())
	if value == "TO" {
		return token{typ: tokTO}
	}

	l.pos = startPos
	if l.pos < l.length {
		l.current = rune(l.input[l.pos])
	}
	return l.scanTerm()
}
