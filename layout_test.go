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

// The package used for end-to-end testing
package layout_test

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/reddec/layout/internal"
	"github.com/reddec/layout/internal/ui/simple"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRender_basic(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	err = internal.Deploy(context.Background(), internal.Config{
		Source: "test-data/projectA",
		Target: tempDir,
		Display: simple.New(bufio.NewReader(strings.NewReader(
			"alice\n1234\n3\nn\n1\n",
		)), io.Discard),
	})
	require.NoError(t, err)

	filepath.Walk(tempDir, func(path string, info fs.FileInfo, err error) error {
		t.Log(path)
		return err
	})

	assert.FileExists(t, filepath.Join(tempDir, "created.txt"))
	assert.FileExists(t, filepath.Join(tempDir, "README.md"))
	assert.DirExists(t, filepath.Join(tempDir, "alice"))
	require.FileExists(t, filepath.Join(tempDir, "alice", "the foo.txt"))
	content, err := ioutil.ReadFile(filepath.Join(tempDir, "alice", "the foo.txt"))
	require.NoError(t, err)
	require.Equal(t, "Hello world the foo as bar", string(bytes.TrimSpace(content)))

	rootContent, err := ioutil.ReadFile(filepath.Join(tempDir, "root.text"))
	require.NoError(t, err)
	require.Equal(t, "the foo", string(bytes.TrimSpace(rootContent)))

	ingored, err := ioutil.ReadFile(filepath.Join(tempDir, "alice", "ignore.txt"))
	require.NoError(t, err)
	require.Equal(t, "This file should not be templated {{.foo}}", string(bytes.TrimSpace(ingored)))
}

func TestRender_gitClone(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	repoDir, err := os.MkdirTemp("", "")
	require.NoError(t, err)
	defer os.RemoveAll(repoDir)

	_, err = git.PlainInit(tempDir, true)
	require.NoError(t, err)

	repo, err := git.PlainInit(repoDir, false)
	require.NoError(t, err)
	err = repo.CreateBranch(&config.Branch{
		Name:   "master",
		Remote: "origin",
		Merge:  "refs/heads/master",
	})

	w, err := repo.Worktree()
	require.NoError(t, err)

	files, err := internal.CopyTree("test-data/projectA", repoDir)
	require.NoError(t, err)

	for _, f := range files.Paths() {
		rel, _ := filepath.Rel(repoDir, f)
		t.Log("+", rel)
		_, err = w.Add(rel)
		require.NoError(t, err)
	}

	_, err = w.Commit("initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Demo",
			Email: "demo@example.com",
			When:  time.Now(),
		},
		All: true,
	})
	require.NoError(t, err)

	_, err = repo.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{tempDir},
	})
	require.NoError(t, err)

	err = repo.Push(&git.PushOptions{
		RemoteName: "origin",
	})
	require.NoError(t, err)

	resultDir, err := os.MkdirTemp("", "")
	require.NoError(t, err)
	defer os.RemoveAll(resultDir)

	// wooh - finally we initialized bare repo which we can clone
	err = internal.Deploy(context.Background(), internal.Config{
		Source: "file://" + tempDir,
		Target: resultDir,
		Display: simple.New(bufio.NewReader(strings.NewReader(
			"alice\n1234\n3\nn\n1\n",
		)), io.Discard),
	})
	require.NoError(t, err)

	assert.FileExists(t, filepath.Join(resultDir, "created.txt"))
	assert.FileExists(t, filepath.Join(resultDir, "README.md"))
	assert.DirExists(t, filepath.Join(resultDir, "alice"))
	require.FileExists(t, filepath.Join(resultDir, "alice", "the foo.txt"))
	content, err := ioutil.ReadFile(filepath.Join(resultDir, "alice", "the foo.txt"))
	require.NoError(t, err)
	require.Equal(t, "Hello world the foo as bar", string(bytes.TrimSpace(content)))
}

func TestRender_multiProject(t *testing.T) {
	t.Run("select project A", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		err = internal.Deploy(context.Background(), internal.Config{
			Source: "test-data",
			Target: tempDir,
			Display: simple.New(bufio.NewReader(strings.NewReader(
				"1\nalice\n1234\n3\nn\n1\n",
			)), io.Discard),
		})
		require.NoError(t, err)
		assert.FileExists(t, filepath.Join(tempDir, "created.txt"))

	})
	t.Run("select project B", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		err = internal.Deploy(context.Background(), internal.Config{
			Source: "test-data",
			Target: tempDir,
			Display: simple.New(bufio.NewReader(strings.NewReader(
				"2\n",
			)), io.Discard),
		})
		require.NoError(t, err)
		assert.FileExists(t, filepath.Join(tempDir, "layout2"))

	})
}
