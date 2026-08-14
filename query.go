package luceneq

// QueryNode представляет узел абстрактного синтаксического дерева (AST)
// запроса Lucene.
type QueryNode interface {
	isQueryNode()
}

// TermQuery — одиночный термин для поиска.
// Поддерживает wildcards (? и *) и нечеткий поиск (~).
type TermQuery struct {
	// Term искомый термин без модификаторов.
	Term string
	// Wildcard указывает на наличие wildcard символов.
	Wildcard bool
	// Fuzzy указывает на наличие fuzzy поиска.
	Fuzzy bool
	// FuzzyDistance нечеткость поиска (0 = авто, > 0 = фиксированное расстояние).
	FuzzyDistance int
}

func (*TermQuery) isQueryNode() {}

// PhraseQuery — точная фраза для поиска.
// Поддерживает proximity search (tilde с числом).
type PhraseQuery struct {
	// Words слова фразы в порядке appearances.
	Words []string
	// Slop максимальное расстояние между словами (для proximity search).
	Slop int
}

func (*PhraseQuery) isQueryNode() {}

// RangeQuery — поиск в диапазоне значений.
type RangeQuery struct {
	// Lower нижняя граница.
	Lower string
	// Upper верхняя граница.
	Upper string
	// IncludeLower включает ли нижнюю границу.
	IncludeLower bool
	// IncludeUpper включает ли верхнюю границу.
	IncludeUpper bool
}

func (*RangeQuery) isQueryNode() {}

// BooleanQuery — логический оператор.
type BooleanQuery struct {
	// Operator тип логической операции.
	Operator BooleanOperator
	// Clauses дочерние узлы.
	Clauses []QueryNode
}

// BooleanOperator определяет тип логической операции.
type BooleanOperator int

const (
	// BooleanAND операция AND (все условия должны совпасть).
	BooleanAND BooleanOperator = iota
	// BooleanOR операция OR (хотя бы одно условие).
	BooleanOR
	// BooleanNOT операция NOT (исключение).
	BooleanNOT
	// BooleanMUST обязательное наличие (required term).
	BooleanMUST
	// BooleanMUSTNOT запрет (prohibited term).
	BooleanMUSTNOT
)

func (*BooleanQuery) isQueryNode() {}

// ConstantQuery всегда возвращает заданное значение.
type ConstantQuery struct {
	Value bool
}

func (*ConstantQuery) isQueryNode() {}
