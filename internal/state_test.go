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
	"bufio"
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/reddec/layout/internal/ui/simple"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAsk(t *testing.T) {
	t.Run("base types", func(t *testing.T) {
		input := bytes.NewBufferString("123\ny\n456.78\nalice and bob\n1,3\nalfa, beta\n")
		expected := map[string]interface{}{
			"int":       int64(123),
			"bool":      true,
			"float":     456.78,
			"string":    "alice and bob",
			"list":      []string{"alice", "charly"},
			"free-list": []string{"alfa", "beta"},
		}
		prompts := []Prompt{
			{Var: "int", Type: VarInt},
			{Var: "bool", Type: VarBool},
			{Var: "float", Type: VarFloat},
			{Var: "string", Type: VarString},
			{Var: "list", Type: VarList, Options: []string{"alice", "bob", "charly"}},
			{Var: "free-list", Type: VarList},
		}
		state := make(map[string]interface{})
		err := askState(context.Background(), simple.New(bufio.NewReader(input), io.Discard), prompts, "", "", state, true)
		require.NoError(t, err)

		for k, v := range expected {
			assert.Equal(t, v, state[k])
		}

		for k := range state {
			assert.Contains(t, expected, k)
		}
	})

	t.Run("default value", func(t *testing.T) {
		input := bytes.NewBufferString("\n")
		expected := map[string]interface{}{
			"int": int64(123),
		}
		prompts := []Prompt{
			{Var: "int", Type: VarInt, Default: "123"},
		}
		state := make(map[string]interface{})
		err := askState(context.Background(), simple.New(bufio.NewReader(input), io.Discard), prompts, "", "", state, true)
		require.NoError(t, err)

		for k, v := range expected {
			assert.Equal(t, v, state[k])
		}

		for k := range state {
			assert.Contains(t, expected, k)
		}
	})

	t.Run("restricted value", func(t *testing.T) {
		input := bytes.NewBufferString("woo\n")
		prompts := []Prompt{
			{Var: "string", Type: VarString, Options: []string{"abc", "def"}},
		}
		state := make(map[string]interface{})
		err := askState(context.Background(), simple.New(bufio.NewReader(input), io.Discard), prompts, "", "", state, true)
		require.Error(t, err)
	})

	t.Run("allowed value", func(t *testing.T) {
		input := bytes.NewBufferString("2\n")
		expected := map[string]interface{}{
			"string": "def",
		}
		prompts := []Prompt{
			{Var: "string", Type: VarString, Options: []string{"abc", "def"}},
		}
		state := make(map[string]interface{})
		err := askState(context.Background(), simple.New(bufio.NewReader(input), io.Discard), prompts, "", "", state, true)
		require.NoError(t, err)

		for k, v := range expected {
			assert.Equal(t, v, state[k])
		}

		for k := range state {
			assert.Contains(t, expected, k)
		}
	})

	t.Run("templated default value value", func(t *testing.T) {
		input := bytes.NewBufferString("def\n\n")
		expected := map[string]interface{}{
			"string":    "def",
			"templated": "abc def",
		}
		prompts := []Prompt{
			{Var: "string", Type: VarString},
			{Var: "templated", Type: VarString, Default: "abc {{.string}}"},
		}
		state := make(map[string]interface{})
		err := askState(context.Background(), simple.New(bufio.NewReader(input), io.Discard), prompts, "", "", state, true)
		require.NoError(t, err)

		for k, v := range expected {
			assert.Equal(t, v, state[k])
		}

		for k := range state {
			assert.Contains(t, expected, k)
		}
	})

	t.Run("condition - skip", func(t *testing.T) {
		input := bytes.NewBufferString("123\n\n")
		expected := map[string]interface{}{
			"foo": int64(123),
		}
		prompts := []Prompt{
			{Var: "foo", Type: VarInt},
			{Var: "skipped", Type: VarString, When: "foo < 100"},
		}
		state := make(map[string]interface{})
		err := askState(context.Background(), simple.New(bufio.NewReader(input), io.Discard), prompts, "", "", state, true)
		require.NoError(t, err)

		for k, v := range expected {
			assert.Equal(t, v, state[k])
		}

		for k := range state {
			assert.Contains(t, expected, k)
		}
	})

	t.Run("condition - pass", func(t *testing.T) {
		input := bytes.NewBufferString("99\n\n")
		expected := map[string]interface{}{
			"foo":     int64(99),
			"skipped": "",
		}
		prompts := []Prompt{
			{Var: "foo", Type: VarInt},
			{Var: "skipped", Type: VarString, When: "foo < 100"},
		}
		state := make(map[string]interface{})
		err := askState(context.Background(), simple.New(bufio.NewReader(input), io.Discard), prompts, "", "", state, true)
		require.NoError(t, err)

		for k, v := range expected {
			assert.Equal(t, v, state[k])
		}

		for k := range state {
			assert.Contains(t, expected, k)
		}
	})

	t.Run("include", func(t *testing.T) {
		input := bytes.NewBufferString("99\n\n\n")
		expected := map[string]interface{}{
			"foo": int64(99),
			"bar": "baz 99",
			"zoo": "zoo baz 99",
		}

		source := createDir(map[string]string{
			"dir/xxx.yaml": `
- var: bar
  default: "baz {{.foo}}"
- include: zoo.yaml # relative include
`,
			"dir/zoo.yaml": `
- var: zoo
  default: "zoo {{.bar}}"
`,
		})
		defer os.RemoveAll(source)

		prompts := []Prompt{
			{Var: "foo", Type: VarInt},
			{Include: "dir/xxx.yaml"},
		}
		state := make(map[string]interface{})
		err := askState(context.Background(), simple.New(bufio.NewReader(input), io.Discard), prompts, "", source, state, true)
		require.NoError(t, err)

		for k, v := range expected {
			assert.Equal(t, v, state[k])
		}

		for k := range state {
			assert.Contains(t, expected, k)
		}
	})

	t.Run("retry on error", func(t *testing.T) {
		input := bytes.NewBufferString("abc\n123\n")
		expected := map[string]interface{}{
			"foo": int64(123),
		}
		prompts := []Prompt{
			{Var: "foo", Type: VarInt},
		}
		state := make(map[string]interface{})
		err := askState(context.Background(), simple.New(bufio.NewReader(input), io.Discard), prompts, "", "", state, false)
		require.NoError(t, err)

		for k, v := range expected {
			assert.Equal(t, v, state[k])
		}

		for k := range state {
			assert.Contains(t, expected, k)
		}
	})
}

func TestComputed(t *testing.T) {
	t.Run("computed works", func(t *testing.T) {
		c := Computed{
			Var:   "foo",
			Value: "foo {{.bar}}",
		}
		state := make(map[string]interface{})
		state["bar"] = 123
		err := c.compute(context.Background(), state)
		require.NoError(t, err)
		require.Contains(t, state, "foo")
		assert.Equal(t, "foo 123", state["foo"])
	})

	t.Run("skipped computed not rendered at all", func(t *testing.T) {
		c := Computed{
			Var:   "foo",
			Value: "foo {{.unknown}}",
			When:  "false",
		}
		state := make(map[string]interface{})
		err := c.compute(context.Background(), state)
		require.NoError(t, err)
		require.NotContains(t, state, "foo")
	})

	t.Run("non-string values used as-is", func(t *testing.T) {
		c := Computed{
			Var:   "foo",
			Value: 123456,
		}
		state := make(map[string]interface{})
		err := c.compute(context.Background(), state)
		require.NoError(t, err)
		require.Equal(t, 123456, state["foo"])
	})
}

func TestCondition_Eval(t *testing.T) {
	t.Run("helper 'has' works", func(t *testing.T) {
		ok, err := Condition(`has(options, "alice")`).Eval(context.Background(), map[string]interface{}{
			"options": []string{"bob", "alice", "charly"},
		})
		require.NoError(t, err)
		require.True(t, ok)

		// negative
		ok, err = Condition(`has(options, "alice")`).Eval(context.Background(), map[string]interface{}{
			"options": []string{"bob", "viktor", "charly"},
		})
		require.NoError(t, err)
		require.False(t, ok)

		// wrong call
		_, err = Condition(`has(options)`).Eval(context.Background(), map[string]interface{}{
			"options": []string{"bob", "alice", "charly"},
		})
		require.Error(t, err)

		_, err = Condition(`has(123, "alice")`).Eval(context.Background(), map[string]interface{}{})
		require.Error(t, err)
	})
}

func createDir(content map[string]string) string {
	d, err := os.MkdirTemp("", "")
	if err != nil {
		panic(err)
	}

	for path, content := range content {
		realPath := filepath.Join(d, path)
		if err := os.MkdirAll(filepath.Dir(realPath), 0755); err != nil {
			panic(err)
		}
		if err := ioutil.WriteFile(realPath, []byte(content), 0755); err != nil {
			panic(err)
		}
	}
	return d
}
