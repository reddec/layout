/*
Copyright 2022 Aleksandr Baryshnikov

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package internal

import (
	"fmt"
	"os"
	"path/filepath"
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

func TestGetRootFile(t *testing.T) {
	workDir, err := filepath.Abs(".")
	assert.NoError(t, err)
	t.Run("simple go mod", func(t *testing.T) {
		content, err := getRootFile(workDir)("go.mod")
		require.NoError(t, err)
		require.Contains(t, content, "module github.com/reddec/layout")
	})

	t.Run("proofed go mod", func(t *testing.T) {
		_, err := getRootFile(workDir)("../../../../../../etc/hosts")
		require.Error(t, err)
		require.ErrorIs(t, err, os.ErrNotExist)
	})

	t.Run("malformed go mod", func(t *testing.T) {
		content, err := getRootFile(workDir)("/foo/bar/go.mod")
		require.NoError(t, err)
		require.Contains(t, content, "module github.com/reddec/layout")
	})
}

func TestSubmatch(t *testing.T) {
	content := `
foo bar

foo barbaz
bar=123,456
`
	t.Run("match groups", func(t *testing.T) {
		matches, err := findSubmatchAll(`foo[ ]+(.+)`, content)
		require.NoError(t, err)
		require.Equal(t, []string{"bar", "barbaz"}, matches)
	})
}

func ExampleFindSubmatch() {
	pattern := `foo[ ]+([^ ]+)`
	text := `foo bar foo baz`
	out, _ := findSubmatchAll(pattern, text)
	fmt.Println(out)
	// output: [bar baz]
}
