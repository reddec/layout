package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCustomDelimiters(t *testing.T) {
	state := map[string]interface{}{
		"foo": "bar",
	}

	r := newRenderContext(state).Delimiters("[[", "]]")
	t.Run("should render with custom delimiters", func(t *testing.T) {
		v, err := r.Render("hello [[.foo]]")
		require.NoError(t, err)
		assert.Equal(t, "hello bar", v)
	})
	t.Run("should ignore original delimiters", func(t *testing.T) {
		v, err := r.Render("hello {{.foo}}")
		require.NoError(t, err)
		assert.Equal(t, "hello {{.foo}}", v)
	})
}
