# luceneq

<div align="center">

Go-библиотека для парсинга Lucene-запросов и фильтрации текста

[![Go Report Card](https://goreportcard.com/badge/github.com/denisdubovitskiy/luceneq)](https://goreportcard.com/report/github.com/denisdubovitskiy/luceneq)
[![Go Reference](https://pkg.go.dev/badge/github.com/denisdubovitskiy/luceneq.svg)](https://pkg.go.dev/github.com/denisdubovitskiy/luceneq)

</div>

## Возможности

- ✨ **Парсинг** Lucene-подобного синтаксиса запросов
- 🔍 **Фильтрация** текста по условиям из запроса
- 🧩 **Логические операторы**: AND, OR, NOT, `&&`, `||`, `!`
- 📝 **Фразы** в кавычках: `"hello world"`
- 🃏 **Wildcard-поиск**: `test*`, `te?t`
- 🎯 **Нечёткий поиск**: `roam~` (Levenshtein distance)
- ➕ **Required/Prohibited**: `+term`, `-term`
- 🏗 **Группировка**: `(a OR b) AND c`
- 📏 **Range-запросы**: `[a TO z]`, `{a TO z}`

## Установка

```bash
go get github.com/denisdubovitskiy/luceneq
```

## Быстрый старт

```go
package main

import (
    "fmt"
    "github.com/denisdubovitskiy/luceneq"
)

func main() {
    parser := luceneq.NewParser()
    
    // Парсим запрос
    matcher, err := parser.ParseQuery("hello AND world")
    if err != nil {
        panic(err)
    }
    
    // Проверяем текст
    fmt.Println(matcher.Match("hello world"))  // true
    fmt.Println(matcher.Match("hello only"))   // false
}
```

## API

### Интерфейсы

```go
// Parser разбивает строку запроса на токены и строит
// дерево выражений для последующей фильтрации текста.
type Parser interface {
    ParseQuery(query string) (Matcher, error)
}

// Matcher проверяет, соответствует ли текст условию запроса.
type Matcher interface {
    Match(text string) bool
}
```

### Примеры запросов

#### Одиночные термины

```go
parser.ParseQuery("hello")
// ✅ "hello world", "say hello"
// ❌ "goodbye"
```

#### Фразы

```go
parser.ParseQuery(`"hello world"`)
// ✅ "hello world, how are you"
// ❌ "world hello", "hello there world"
```

#### Логические операторы

```go
parser.ParseQuery("hello AND world")     // оба термина должны быть
parser.ParseQuery("hello OR world")      // хотя бы один
parser.ParseQuery("hello NOT world")     // hello без world
parser.ParseQuery("hello && world")      // альтернатива AND
parser.ParseQuery("hello || world")      // альтернатива OR
```

#### Required / Prohibited

```go
parser.ParseQuery("+hello world")    // hello обязательно
parser.ParseQuery("hello -world")    // world исключён
```

#### Wildcard-поиск

```go
parser.ParseQuery("test*")   // test, testing, tester
parser.ParseQuery("te?t")    // test, tent, tot
```

#### Нечёткий поиск

```go
parser.ParseQuery("roam~")     // roam, foam, roams (distance ≤ 2)
parser.ParseQuery("foam~1")    // только distance ≤ 1
```

#### Группировка

```go
parser.ParseQuery("(hello OR world) AND test")
```

#### Range-запросы

```go
parser.ParseQuery("[a TO m]")    // включительно: [a, m]
parser.ParseQuery("{a TO m}")    // исключая границы: (a, m)
```

## Структура проекта

```
.
├── query.go            # AST узлы (TermQuery, PhraseQuery, RangeQuery, BooleanQuery)
├── parser.go           # Лексер и парсер (рекурсивный спуск)
├── matcher.go          # Matcher интерфейсы и Match()
├── luceneq.go          # Публичный API: NewParser()
├── complex_test.go     # 60+ тестов на сложные запросы
├── fuzzy_test.go       # 60+ тестов на fuzzy search
├── parser_test.go      # Тесты парсинга
├── matcher_test.go     # Тесты matcher
├── Makefile            # Команды сборки, тестов, линтинга
├── .golangci.yml       # Конфигурация golangci-lint
└── README.md           # Документация
```

## Разработка

### Makefile

| Команда | Описание |
|---------|----------|
| `make` | Полная проверка (lint + vet + test) |
| `make build` | Собрать библиотеку |
| `make test` | Запустить все тесты |
| `make test-race` | Тесты с детектором гонок |
| `make test-cover` | Тесты с отчётом о покрытии |
| `make test-bench` | Запустить бенчмарки |
| `make lint` | Запустить golangci-lint |
| `make lint-fix` | Линтинг с автоисправлениями |
| `make fmt` | Отформатировать код |
| `make vet` | Запустить go vet |
| `make check` | Полная проверка |
| `make pre-commit` | Подготовка к коммиту |
| `make help` | Показать справку |

### golangci-lint

Устанавливается автоматически при первом запуске `make lint`:

```bash
make lint-install  # установить вручную
```

Или через go install:

```bash
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

## Лицензия

MIT

## Об авторе

Проект целиком и полностью написан локально с помощью модели
**Qwen3.6-35B-A3B-MTP-GGUF-UD-Q4_K_XL**.
