// The package used for end-to-end testing
package layout_test

import (
	"bufio"
	"bytes"
	"context"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"layout/internal"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRender_basic(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	err = internal.DeployFrom(context.Background(), "test-data", tempDir, os.Stderr, bufio.NewReader(strings.NewReader(
		"alice\nthe foo\nn\n",
	)))
	require.NoError(t, err)

	assert.FileExists(t, filepath.Join(tempDir, "created.txt"))
	assert.FileExists(t, filepath.Join(tempDir, "README.md"))
	assert.DirExists(t, filepath.Join(tempDir, "alice"))
	require.FileExists(t, filepath.Join(tempDir, "alice", "the foo.txt"))
	content, err := ioutil.ReadFile(filepath.Join(tempDir, "alice", "the foo.txt"))
	require.NoError(t, err)
	require.Equal(t, "Hello world the foo as bar", string(bytes.TrimSpace(content)))
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

	files, err := internal.CopyTree("test-data", repoDir)
	require.NoError(t, err)

	for _, f := range files {
		t.Log("+", f)
		_, err = w.Add(f)
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
	err = internal.DeployFrom(context.Background(), "file://"+tempDir, resultDir, os.Stderr, bufio.NewReader(strings.NewReader(
		"alice\nthe foo\nn\n",
	)))
	require.NoError(t, err)

	assert.FileExists(t, filepath.Join(resultDir, "created.txt"))
	assert.FileExists(t, filepath.Join(resultDir, "README.md"))
	assert.DirExists(t, filepath.Join(resultDir, "alice"))
	require.FileExists(t, filepath.Join(resultDir, "alice", "the foo.txt"))
	content, err := ioutil.ReadFile(filepath.Join(resultDir, "alice", "the foo.txt"))
	require.NoError(t, err)
	require.Equal(t, "Hello world the foo as bar", string(bytes.TrimSpace(content)))
}
