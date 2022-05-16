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
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunnable(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	ctx := context.Background()
	state := map[string]interface{}{
		"foo": 123,
		"bar": "baz",
	}

	t.Run("inline run should work", func(t *testing.T) {
		run := Runnable{
			Run: "echo -n {{.foo}} > inline.txt",
		}
		err := run.execute(ctx, newRenderContext(state), tmpDir, "")
		require.NoError(t, err)
		require.FileExists(t, filepath.Join(tmpDir, "inline.txt"))
		requireContent(t, "123", filepath.Join(tmpDir, "inline.txt"))
	})

	t.Run("simple hook should work", func(t *testing.T) {
		hooksDir, err := os.MkdirTemp("", "")
		require.NoError(t, err)
		defer os.RemoveAll(hooksDir)

		err = ioutil.WriteFile(filepath.Join(hooksDir, "h o o k.sh"), []byte(strings.TrimSpace(`
#!/bin/sh
echo -n {{.foo}} > hook.txt
`)), 0755)
		require.NoError(t, err)

		run := Runnable{
			Script: "'h o o k.sh'",
		}
		err = run.execute(ctx, newRenderContext(state), tmpDir, hooksDir)
		require.NoError(t, err)
		require.FileExists(t, filepath.Join(tmpDir, "hook.txt"))
		requireContent(t, "123", filepath.Join(tmpDir, "hook.txt"))
	})

	t.Run("hook with args should work", func(t *testing.T) {
		hooksDir, err := os.MkdirTemp("", "")
		require.NoError(t, err)
		defer os.RemoveAll(hooksDir)

		err = ioutil.WriteFile(filepath.Join(hooksDir, "hook.sh"), []byte(strings.TrimSpace(`
#!/bin/sh
echo -n "$1" > hook2.txt
`)), 0755)
		require.NoError(t, err)

		run := Runnable{
			Script: "hook.sh '{{.foo}}'",
		}
		err = run.execute(ctx, newRenderContext(state), tmpDir, hooksDir)
		require.NoError(t, err)
		require.FileExists(t, filepath.Join(tmpDir, "hook2.txt"))
		requireContent(t, "123", filepath.Join(tmpDir, "hook2.txt"))
	})
}

func requireContent(t *testing.T, expected string, fileName string) {
	d, err := ioutil.ReadFile(fileName)
	require.NoError(t, err)
	require.Equal(t, expected, string(d))
}
