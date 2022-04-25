package internal_test

import (
	"bufio"
	"bytes"
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"layout/internal"
	"os"
	"testing"
)

func TestAsk(t *testing.T) {
	t.Run("base types", func(t *testing.T) {
		input := bytes.NewBufferString("123\ny\n456.78\nalice and bob\n1,3\n")
		expected := map[string]interface{}{
			"int":    int64(123),
			"bool":   true,
			"float":  456.78,
			"string": "alice and bob",
			"list":   []string{"alice", "charly"},
		}
		prompts := []internal.Prompt{
			{Var: "int", Type: internal.VarInt},
			{Var: "bool", Type: internal.VarBool},
			{Var: "float", Type: internal.VarFloat},
			{Var: "string", Type: internal.VarString},
			{Var: "list", Type: internal.VarList, Options: []string{"alice", "bob", "charly"}},
		}
		state := make(map[string]interface{})
		err := internal.AskState(context.Background(), os.Stdout, bufio.NewReader(input), prompts, "", state)
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
		err := internal.AskState(context.Background(), os.Stdout, bufio.NewReader(input), prompts, "", state)
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
		err := internal.AskState(context.Background(), os.Stdout, bufio.NewReader(input), prompts, "", state)
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
		err := internal.AskState(context.Background(), os.Stdout, bufio.NewReader(input), prompts, "", state)
		require.NoError(t, err)

		for k, v := range expected {
			assert.Equal(t, v, state[k])
		}

		for k := range state {
			assert.Contains(t, expected, k)
		}
	})
}
