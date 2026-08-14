package luceneq

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFuzzySearch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		query       string
		wantMatch   bool
		text        string
		fuzzyDist   int
		description string
	}{
		// ============================================
		// Fuzzy с расстоянием 0 (точное совпадение)
		// ============================================

		{
			name:        "fuzzy exact match distance 0",
			query:       "hello",
			wantMatch:   true,
			text:        "hello world",
			fuzzyDist:   0,
			description: "точное совпадение",
		},
		{
			name:        "fuzzy exact no match distance 0",
			query:       "hello",
			wantMatch:   false,
			text:        "halo world",
			fuzzyDist:   0,
			description: "различия есть — не совпадает",
		},

		// ============================================
		// Fuzzy с расстоянием 1 (одно исправление)
		// ============================================

		{
			name:        "fuzzy distance 1: substitution",
			query:       "hello",
			wantMatch:   true,
			text:        "hallo world",
			fuzzyDist:   1,
			description: "замена одной буквы: e -> a",
		},
		{
			name:        "fuzzy distance 1: insertion",
			query:       "test",
			wantMatch:   true,
			text:        "tests found",
			fuzzyDist:   1,
			description: "вставка буквы: test -> tests",
		},
		{
			name:        "fuzzy distance 1: no match",
			query:       "testing",
			wantMatch:   false,
			text:        "test data found",
			fuzzyDist:   1,
			description: "testing -> test (расстояние 3, больше 1)",
		},
		{
			name:        "fuzzy distance 2: no match",
			query:       "hello",
			wantMatch:   false,
			text:        "xyz world",
			fuzzyDist:   2,
			description: "слишком большая разница",
		},

		// ============================================
		// Fuzzy с расстоянием 3
		// ============================================

		{
			name:        "fuzzy distance 3: kitten sitting",
			query:       "kitten",
			wantMatch:   true,
			text:        "sitting test",
			fuzzyDist:   3,
			description: "kitten -> sitting (расстояние 3)",
		},

		// ============================================
		// Fuzzy с расстоянием 3 (три исправления)
		// ============================================

		{
			name:        "fuzzy distance 3: kitten sitting",
			query:       "kitten",
			wantMatch:   true,
			text:        "sitting test",
			fuzzyDist:   3,
			description: "kitten -> sitting (расстояние 3)",
		},
		{
			name:        "fuzzy distance 3: long transformation",
			query:       "testing",
			wantMatch:   false,
			text:        "tentative approach",
			fuzzyDist:   3,
			description: "testing -> tentative (расстояние 5, больше 3)",
		},

		// ============================================
		// Fuzzy с очень большим расстоянием
		// ============================================

		{
			name:        "fuzzy large distance: very similar",
			query:       "algorithm",
			wantMatch:   true,
			text:        "algorithms implemented",
			fuzzyDist:   5,
			description: "algorithm -> algorithms (вставка s и s)",
		},
		{
			name:        "fuzzy large distance: very different",
			query:       "algorithm",
			wantMatch:   false,
			text:        "completely random words here",
			fuzzyDist:   5,
			description: "слишком разные слова",
		},

		// ============================================
		// Fuzzy с короткими словами
		// ============================================

		{
			name:        "fuzzy short word: cat",
			query:       "cat",
			wantMatch:   true,
			text:        "cats and dogs",
			fuzzyDist:   1,
			description: "cat -> cats (вставка s)",
		},
		{
			name:        "fuzzy short word: log",
			query:       "log",
			wantMatch:   true,
			text:        "logs found",
			fuzzyDist:   1,
			description: "log -> logs (вставка s)",
		},
		{
			name:        "fuzzy short word: api",
			query:       "api",
			wantMatch:   true,
			text:        "apis defined",
			fuzzyDist:   1,
			description: "api -> apis (вставка s)",
		},

		// ============================================
		// Fuzzy с длинными словами
		// ============================================

		{
			name:        "fuzzy long word: authentication",
			query:       "authentication",
			wantMatch:   true,
			text:        "authentication token",
			fuzzyDist:   0,
			description: "точное совпадение длинного слова",
		},
		{
			name:        "fuzzy long word: authorization",
			query:       "authentication",
			wantMatch:   true,
			text:        "authorization headers",
			fuzzyDist:   4,
			description: "authentication -> authorization (несколько замен)",
		},
		{
			name:        "fuzzy long word: implementation",
			query:       "implementation",
			wantMatch:   true,
			text:        "implementations tested",
			fuzzyDist:   3,
			description: "implementation -> implementations (вставки)",
		},

		// ============================================
		// Fuzzy на разных позициях текста
		// ============================================

		{
			name:        "fuzzy at start",
			query:       "hello",
			wantMatch:   true,
			text:        "hallo there world",
			fuzzyDist:   1,
			description: "слово с ошибкой в начале",
		},
		{
			name:        "fuzzy in middle",
			query:       "world",
			wantMatch:   true,
			text:        "hello wrld is great",
			fuzzyDist:   1,
			description: "слово с ошибкой в середине",
		},
		{
			name:        "fuzzy at end",
			query:       "test",
			wantMatch:   true,
			text:        "this is a tes",
			fuzzyDist:   1,
			description: "слово с ошибкой в конце: test -> tes (удаление t)",
		},

		// ============================================
		// Fuzzy не должен находить подстроки
		// ============================================

		{
			name:        "fuzzy no substring match",
			query:       "cat",
			wantMatch:   false,
			text:        "category theory",
			fuzzyDist:   1,
			description: "cat не должен находить category (подстрока)",
		},
		{
			name:        "fuzzy no partial match",
			query:       "go",
			wantMatch:   false,
			text:        "gopher programming",
			fuzzyDist:   1,
			description: "go не должен находить gopher",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// arrange
			q := &TermQuery{
				Term:          tt.query,
				Fuzzy:         true,
				FuzzyDistance: tt.fuzzyDist,
			}

			// act
			got := q.Match(tt.text)

			// assert
			assert.Equal(t, tt.wantMatch, got, "%s: query=%q, text=%q, dist=%d", tt.description, tt.query, tt.text, tt.fuzzyDist)
		})
	}
}

func TestFuzzyDistanceCalculation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		a, b string
		dist int
	}{
		{"kitten", "sitting", 3},
		{"roam", "foam", 1},
		{"roam", "roams", 1},
		{"roam", "rooms", 2},
		{"hello", "hello", 0},
		{"hello", "hallo", 1},
		{"hello", "xello", 1},
		{"test", "tests", 1},
		{"test", "testing", 3},
		{"api", "apis", 1},
		{"cat", "cats", 1},
		{"dog", "dogs", 1},
		{"log", "logs", 1},
		{"log", "logging", 4},
		{"authentication", "authorization", 4},
		{"implementation", "implementations", 1},
		{"", "test", 4},
		{"test", "", 4},
		{"", "", 0},
		{"a", "b", 1},
		{"ab", "ac", 1},
		{"abc", "axc", 1},
		{"abc", "axy", 2},
		{"abc", "axyz", 3},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.a+"-"+tt.b, func(t *testing.T) {
			t.Parallel()

			// act
			got := levenshteinDistance(tt.a, tt.b)

			// assert
			assert.Equal(t, tt.dist, got, "levenshteinDistance(%q, %q)", tt.a, tt.b)
		})
	}
}

func TestFuzzyInComplexQueries(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		query     string
		wantMatch bool
		text      string
	}{
		// ============================================
		// Fuzzy в AND compound
		// ============================================

		{
			name:      "fuzzy AND exact",
			query:     "goroutin* AND channels",
			wantMatch: true,
			text:      "goroutines and channels work",
		},
		{
			name:      "fuzzy AND fuzzy",
			query:     "goroutin* OR channels",
			wantMatch: true,
			text:      "goroutines work",
		},

		// ============================================
		// Fuzzy в OR compound
		// ============================================

		{
			name:      "fuzzy OR fuzzy: first match",
			query:     "channel* OR goroutine",
			wantMatch: true,
			text:      "channels used",
		},
		{
			name:      "fuzzy OR fuzzy: second match",
			query:     "channel* OR goroutine",
			wantMatch: true,
			text:      "goroutines created",
		},
		{
			name:      "fuzzy OR fuzzy: no match",
			query:     "channel* OR goroutine",
			wantMatch: false,
			text:      "completely different words",
		},

		// ============================================
		// Fuzzy в NOT compound
		// ============================================

		{
			name:      "fuzzy NOT",
			query:     "deployment AND NOT rollback",
			wantMatch: true,
			text:      "deployment version upgrade",
		},
		{
			name:      "fuzzy NOT: fail",
			query:     "deployment AND NOT rollback",
			wantMatch: false,
			text:      "deployment rollback version",
		},

		// ============================================
		// Fuzzy в группах
		// ============================================

		{
			name:      "fuzzy in group OR",
			query:     "(channl OR goroutin*) AND concurrency",
			wantMatch: true,
			text:      "goroutines concurrency model",
		},
		{
			name:      "fuzzy in group AND",
			query:     "(goroutin* OR channels) AND sync",
			wantMatch: true,
			text:      "goroutines sync package",
		},

		// ============================================
		// Fuzzy в длинных запросах
		// ============================================

		{
			name:      "fuzzy long chain",
			query:     "goroutin* AND (channels OR sync) AND (go OR select)",
			wantMatch: true,
			text:      "goroutines channels go runtime",
		},
		{
			name:      "fuzzy with multiple groups",
			query:     "((goroutin* OR channels) AND (sync OR runtime)) AND concurrency",
			wantMatch: true,
			text:      "goroutines sync concurrency model",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// arrange
			p := NewParser()

			// act
			matcher, err := p.ParseQuery(tt.query)
			assert.NoError(t, err, "query should parse: %q", tt.query)

			// assert
			got := matcher.Match(tt.text)
			assert.Equal(t, tt.wantMatch, got, "query: %q, text: %q", tt.query, tt.text)
		})
	}
}
