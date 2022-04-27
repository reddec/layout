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

package internal_test

import (
	"bufio"
	"bytes"
	"context"
	"os"
	"testing"
	"testing/fstest"

	"layout/internal"

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
		prompts := []internal.Prompt{
			{Var: "int", Type: internal.VarInt},
			{Var: "bool", Type: internal.VarBool},
			{Var: "float", Type: internal.VarFloat},
			{Var: "string", Type: internal.VarString},
			{Var: "list", Type: internal.VarList, Options: []string{"alice", "bob", "charly"}},
			{Var: "free-list", Type: internal.VarList},
		}
		state := make(map[string]interface{})
		err := internal.AskState(context.Background(), os.Stdout, bufio.NewReader(input), prompts, "", nil, state)
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
		prompts := []internal.Prompt{
			{Var: "int", Type: internal.VarInt, Default: "123"},
		}
		state := make(map[string]interface{})
		err := internal.AskState(context.Background(), os.Stdout, bufio.NewReader(input), prompts, "", nil, state)
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
		prompts := []internal.Prompt{
			{Var: "string", Type: internal.VarString, Options: []string{"abc", "def"}},
		}
		state := make(map[string]interface{})
		err := internal.AskState(context.Background(), os.Stdout, bufio.NewReader(input), prompts, "", nil, state)
		require.Error(t, err)
	})

	t.Run("allowed value", func(t *testing.T) {
		input := bytes.NewBufferString("2\n")
		expected := map[string]interface{}{
			"string": "def",
		}
		prompts := []internal.Prompt{
			{Var: "string", Type: internal.VarString, Options: []string{"abc", "def"}},
		}
		state := make(map[string]interface{})
		err := internal.AskState(context.Background(), os.Stdout, bufio.NewReader(input), prompts, "", nil, state)
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
		prompts := []internal.Prompt{
			{Var: "string", Type: internal.VarString},
			{Var: "templated", Type: internal.VarString, Default: "abc {{.string}}"},
		}
		state := make(map[string]interface{})
		err := internal.AskState(context.Background(), os.Stdout, bufio.NewReader(input), prompts, "", nil, state)
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
		prompts := []internal.Prompt{
			{Var: "foo", Type: internal.VarInt},
			{Var: "skipped", Type: internal.VarString, When: "foo < 100"},
		}
		state := make(map[string]interface{})
		err := internal.AskState(context.Background(), os.Stdout, bufio.NewReader(input), prompts, "", nil, state)
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
		prompts := []internal.Prompt{
			{Var: "foo", Type: internal.VarInt},
			{Var: "skipped", Type: internal.VarString, When: "foo < 100"},
		}
		state := make(map[string]interface{})
		err := internal.AskState(context.Background(), os.Stdout, bufio.NewReader(input), prompts, "", nil, state)
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

		source := fstest.MapFS{
			"dir/xxx.yaml": &fstest.MapFile{
				Data: []byte(`
- var: bar
  default: "baz {{.foo}}"
- include: zoo.yaml # relative include
`),
			},
			"dir/zoo.yaml": &fstest.MapFile{Data: []byte(`
- var: zoo
  default: "zoo {{.bar}}"
`)},
		}

		prompts := []internal.Prompt{
			{Var: "foo", Type: internal.VarInt},
			{Include: "dir/xxx.yaml"},
		}
		state := make(map[string]interface{})
		err := internal.AskState(context.Background(), os.Stdout, bufio.NewReader(input), prompts, "", source, state)
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
		prompts := []internal.Prompt{
			{Var: "foo", Type: internal.VarInt},
		}
		state := make(map[string]interface{})
		err := internal.AskState(context.Background(), os.Stdout, bufio.NewReader(input), prompts, "", nil, state)
		require.NoError(t, err)

		for k, v := range expected {
			assert.Equal(t, v, state[k])
		}

		for k := range state {
			assert.Contains(t, expected, k)
		}
	})
}
