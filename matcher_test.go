package luceneq

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTermQuery_Match(t *testing.T) {
	t.Parallel()

	t.Run("exact match", func(t *testing.T) {
		t.Parallel()

		// arrange
		q := &TermQuery{Term: "hello"}

		// act & assert
		assert.True(t, q.Match("hello world"))
		assert.True(t, q.Match("say hello"))
		assert.False(t, q.Match("goodbye"))
	})

	t.Run("case insensitive", func(t *testing.T) {
		t.Parallel()

		// arrange
		q := &TermQuery{Term: "Hello"}

		// act & assert
		assert.True(t, q.Match("hello WORLD"))
		assert.True(t, q.Match("HELLO world"))
	})

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
}

func TestPhraseQuery_Match(t *testing.T) {
	t.Parallel()

	t.Run("exact phrase match", func(t *testing.T) {
		t.Parallel()

		// arrange
		q := &PhraseQuery{Words: []string{"hello", "world"}}

		// act & assert
		assert.True(t, q.Match("hello world, how are you"))
		assert.True(t, q.Match("say hello world"))
		assert.False(t, q.Match("world hello"))
		assert.False(t, q.Match("hello there world"))
	})
}

func TestBooleanQuery_Match(t *testing.T) {
	t.Parallel()

	t.Run("AND operator", func(t *testing.T) {
		t.Parallel()

		// arrange
		q := &BooleanQuery{
			Operator: BooleanAND,
			Clauses: []QueryNode{
				&TermQuery{Term: "hello"},
				&TermQuery{Term: "world"},
			},
		}

		// act & assert
		assert.True(t, q.Match("hello world test"))
		assert.False(t, q.Match("hello only"))
		assert.False(t, q.Match("only world"))
		assert.False(t, q.Match("nothing here"))
	})

	t.Run("OR operator", func(t *testing.T) {
		t.Parallel()

		// arrange
		q := &BooleanQuery{
			Operator: BooleanOR,
			Clauses: []QueryNode{
				&TermQuery{Term: "hello"},
				&TermQuery{Term: "world"},
			},
		}

		// act & assert
		assert.True(t, q.Match("hello world"))
		assert.True(t, q.Match("hello only"))
		assert.True(t, q.Match("only world"))
		assert.False(t, q.Match("nothing here"))
	})

	t.Run("NOT operator", func(t *testing.T) {
		t.Parallel()

		// arrange
		q := &BooleanQuery{
			Operator: BooleanNOT,
			Clauses: []QueryNode{
				&TermQuery{Term: "bad"},
			},
		}

		// act & assert
		assert.True(t, q.Match("good text"))
		assert.False(t, q.Match("bad text"))
	})

	t.Run("MUST operator", func(t *testing.T) {
		t.Parallel()

		// arrange
		q := &BooleanQuery{
			Operator: BooleanMUST,
			Clauses: []QueryNode{
				&TermQuery{Term: "required"},
			},
		}

		// act & assert
		assert.True(t, q.Match("required field"))
		assert.False(t, q.Match("optional field"))
	})

	t.Run("MUST_NOT operator", func(t *testing.T) {
		t.Parallel()

		// arrange
		q := &BooleanQuery{
			Operator: BooleanMUSTNOT,
			Clauses: []QueryNode{
				&TermQuery{Term: "excluded"},
			},
		}

		// act & assert
		assert.True(t, q.Match("allowed text"))
		assert.False(t, q.Match("excluded text"))
	})
}

func TestConstantQuery_Match(t *testing.T) {
	t.Parallel()

	t.Run("always true", func(t *testing.T) {
		t.Parallel()

		// arrange
		q := &ConstantQuery{Value: true}

		// act & assert
		assert.True(t, q.Match("anything"))
		assert.True(t, q.Match(""))
	})

	t.Run("always false", func(t *testing.T) {
		t.Parallel()

		// arrange
		q := &ConstantQuery{Value: false}

		// act & assert
		assert.False(t, q.Match("anything"))
		assert.False(t, q.Match(""))
	})
}

func TestRangeQuery_Match(t *testing.T) {
	t.Parallel()

	t.Run("inclusive range", func(t *testing.T) {
		t.Parallel()

		// arrange
		q := &RangeQuery{
			Lower:        "banana",
			Upper:        "mango",
			IncludeLower: true,
			IncludeUpper: true,
		}

		// act & assert
		assert.True(t, q.Match("banana"))
		assert.True(t, q.Match("cantaloupe"))
		assert.True(t, q.Match("mango"))
		assert.False(t, q.Match("avocado"))
		assert.False(t, q.Match("zucchini"))
	})

	t.Run("exclusive lower bound", func(t *testing.T) {
		t.Parallel()

		// arrange
		q := &RangeQuery{
			Lower:        "apple",
			Upper:        "mango",
			IncludeLower: false,
			IncludeUpper: true,
		}

		// act & assert
		assert.False(t, q.Match("apple"))
		assert.True(t, q.Match("banana"))
		assert.True(t, q.Match("mango"))
	})
}
