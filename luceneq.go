// Package luceneq — Go-библиотека для парсинга Lucene-запросов и применения их
// к произвольному тексту для фильтрации.
//
// Пример использования:
//
//	parser := luceneq.NewParser()
//	matcher, err := parser.ParseQuery("hello AND world")
//	if err != nil {
//	    panic(err)
//	}
//	result := matcher.Match("hello world") // true
package luceneq
