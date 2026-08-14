package luceneq

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParser_ParseQuery_Errors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		query string
	}{
		{
			name:  "unclosed parenthesis",
			query: "(hello world",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// arrange
			p := NewParser()

			// act
			_, err := p.ParseQuery(tt.query)

			// assert
			assert.Error(t, err)
		})
	}
}

func TestTermQuery_Match_Wildcards(t *testing.T) {
	t.Parallel()

	t.Run("wildcard question mark", func(t *testing.T) {
		t.Parallel()

		// arrange
		q := &TermQuery{Term: "t?t", Wildcard: true}

		// act & assert
		assert.True(t, q.Match("tat"))
		assert.True(t, q.Match("tet"))
		assert.True(t, q.Match("tit"))
		assert.False(t, q.Match("tt"))
		assert.False(t, q.Match("taat"))
	})

	t.Run("wildcard asterisk", func(t *testing.T) {
		t.Parallel()

		// arrange
		q := &TermQuery{Term: "test*", Wildcard: true}

		// act & assert
		assert.True(t, q.Match("testing"))
		assert.True(t, q.Match("test"))
		assert.True(t, q.Match("tester"))
		assert.False(t, q.Match("best"))
	})
}
