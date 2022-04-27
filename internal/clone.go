package internal

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/go-git/go-git/v5"
)

func Deploy(ctx context.Context, src string, targetDir string) error {
	return DeployFrom(ctx, src, targetDir, os.Stdout, bufio.NewReader(os.Stdin))
}

func DeployFrom(ctx context.Context, src string, targetDir string, out io.Writer, in *bufio.Reader) error {
	var projectDir string

	info, err := os.Stat(src)
	switch {
	case err == nil && info.IsDir():
		projectDir = src
		//TODO: shorthand
		// user/repo - github
		// <alias>:repo - .layoutrc
	default:
		tmpDir, err := cloneFromGit(ctx, src)
		if err != nil {
			return fmt.Errorf("copy project from git: %w", err)
		}
		defer os.RemoveAll(tmpDir)
		projectDir = tmpDir
	}
	sourceDir := filepath.Join(projectDir, ContentDir)
	manifestFile := filepath.Join(projectDir, ManifestFile)
	manifest, err := LoadManifestFromFile(manifestFile)
	if err != nil {
		return fmt.Errorf("load manifest %s: %w", manifestFile, err)
	}

	err = manifest.RenderTo(ctx, out, in, manifestFile, targetDir, sourceDir)
	if err != nil {
		return fmt.Errorf("render: %w", err)
	}

	return nil
}

func cloneFromGit(ctx context.Context, url string) (projectDir string, err error) {
	tmpDir, err := os.MkdirTemp("", "layout-*")
	if err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}
	_, err = git.PlainCloneContext(ctx, tmpDir, false, &git.CloneOptions{
		URL:               url,
		Depth:             1,
		RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
		Progress:          os.Stderr,
	})
	if err != nil {
		_ = os.RemoveAll(tmpDir)
		return "", fmt.Errorf("clone repo: %w", err)
	}

	return tmpDir, nil
}

func CopyTree(src string, dest string) ([]string, error) {
	var files []string
	err := filepath.Walk(src, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == src {
			return nil
		}
		relPath, err := filepath.Rel(src, path)
		destPath := filepath.Join(dest, relPath)
		if info.IsDir() {
			return os.Mkdir(destPath, info.Mode())
		}
		files = append(files, relPath)
		srcFile, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("open source (%s): %w", path, err)
		}
		defer srcFile.Close()

		destFile, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY, info.Mode())
		if err != nil {
			return fmt.Errorf("open destination (%s): %w", path, err)
		}
		defer destFile.Close()

		_, err = io.Copy(destFile, srcFile)
		if err != nil {
			return fmt.Errorf("copy content (%s): %w", relPath, err)
		}

		return destFile.Close()
	})
	return files, err
}
