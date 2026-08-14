package luceneq

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParser_ParseQuery(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		query     string
		wantErr   bool
		wantMatch bool
		testText  string
	}{
		// Одиночные термины
		{
			name:      "simple term match",
			query:     "hello",
			wantErr:   false,
			wantMatch: true,
			testText:  "hello world",
		},
		{
			name:      "simple term no match",
			query:     "world",
			wantErr:   false,
			wantMatch: false,
			testText:  "hello universe",
		},

		// Фразы
		{
			name:      "phrase match",
			query:     `"hello world"`,
			wantErr:   false,
			wantMatch: true,
			testText:  "hello world, how are you",
		},
		{
			name:      "phrase no match",
			query:     `"hello world"`,
			wantErr:   false,
			wantMatch: false,
			testText:  "world hello",
		},

		// AND
		{
			name:      "AND both match",
			query:     "hello AND world",
			wantErr:   false,
			wantMatch: true,
			testText:  "hello world example",
		},
		{
			name:      "AND first match",
			query:     "hello AND world",
			wantErr:   false,
			wantMatch: false,
			testText:  "hello universe",
		},
		{
			name:      "AND both match uppercase operator",
			query:     "hello && world",
			wantErr:   false,
			wantMatch: true,
			testText:  "hello world",
		},

		// OR
		{
			name:      "OR first match",
			query:     "hello OR world",
			wantErr:   false,
			wantMatch: true,
			testText:  "hello universe",
		},
		{
			name:      "OR second match",
			query:     "hello OR world",
			wantErr:   false,
			wantMatch: true,
			testText:  "universe world",
		},
		{
			name:      "OR no match",
			query:     "hello OR world",
			wantErr:   false,
			wantMatch: false,
			testText:  "goodbye universe",
		},
		{
			name:      "OR with pipe operator",
			query:     "hello || world",
			wantErr:   false,
			wantMatch: true,
			testText:  "hello test",
		},

		// NOT
		{
			name:      "NOT exclude match",
			query:     "hello NOT world",
			wantErr:   false,
			wantMatch: true,
			testText:  "hello universe",
		},
		{
			name:      "NOT exclude no match",
			query:     "hello NOT world",
			wantErr:   false,
			wantMatch: false,
			testText:  "hello world",
		},
		{
			name:      "NOT with exclamation mark",
			query:     "hello ! world",
			wantErr:   false,
			wantMatch: true,
			testText:  "hello foo bar",
		},

		// Wildcards
		{
			name:      "wildcard asterisk",
			query:     "test*",
			wantErr:   false,
			wantMatch: true,
			testText:  "testing is important",
		},
		{
			name:      "wildcard question mark",
			query:     "te?t",
			wantErr:   false,
			wantMatch: true,
			testText:  "test data",
		},
		{
			name:      "wildcard no match",
			query:     "test*",
			wantErr:   false,
			wantMatch: false,
			testText:  "best effort",
		},

		// Fuzzy search
		{
			name:      "fuzzy search match",
			query:     "roam~",
			wantErr:   false,
			wantMatch: true,
			testText:  "foam rubber",
		},
		{
			name:      "fuzzy search no match",
			query:     "roam~",
			wantErr:   false,
			wantMatch: false,
			testText:  "completely different",
		},

		// Группировка
		{
			name:      "grouped OR AND",
			query:     "(hello OR world) AND test",
			wantErr:   false,
			wantMatch: true,
			testText:  "hello test case",
		},
		{
			name:      "grouped OR AND no match",
			query:     "(hello OR world) AND test",
			wantErr:   false,
			wantMatch: false,
			testText:  "hello world",
		},
		{
			name:      "nested groups",
			query:     "(hello OR (world AND test)) AND foo",
			wantErr:   false,
			wantMatch: true,
			testText:  "hello foo bar",
		},

		// Range
		{
			name:      "range inclusive",
			query:     "[a TO m]",
			wantErr:   false,
			wantMatch: true,
			testText:  "apple",
		},
		{
			name:      "range exclusive lower",
			query:     "{a TO m}",
			wantErr:   false,
			wantMatch: true, // apple > a и apple < m
			testText:  "apple",
		},

		// Required term
		{
			name:      "required term must exist",
			query:     "+hello world",
			wantErr:   false,
			wantMatch: true,
			testText:  "hello there world",
		},
		{
			name:      "required term missing",
			query:     "+hello world",
			wantErr:   false,
			wantMatch: false,
			testText:  "there world",
		},

		// Prohibited term
		{
			name:      "prohibited term excluded",
			query:     "hello -world",
			wantErr:   false,
			wantMatch: true,
			testText:  "hello foo bar",
		},
		{
			name:      "prohibited term present",
			query:     "hello -world",
			wantErr:   false,
			wantMatch: false,
			testText:  "hello world",
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
			require.NoError(t, err)

			// assert
			got := matcher.Match(tt.testText)
			assert.Equal(t, tt.wantMatch, got, "query: %q, text: %q", tt.query, tt.testText)
		})
	}
}
