// The package used for end-to-end testing
package layout_test

import (
	"bufio"
	"bytes"
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"layout/internal"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
