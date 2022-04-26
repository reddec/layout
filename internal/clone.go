package internal

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

func Deploy(ctx context.Context, src string, targetDir string) error {
	return DeployFrom(ctx, src, targetDir, os.Stdout, bufio.NewReader(os.Stdin))
}

func DeployFrom(ctx context.Context, src string, targetDir string, out io.Writer, in *bufio.Reader) error {
	var projectDir string

	info, err := os.Stat(src)
	switch {
	case err == nil && info.IsDir():
		if p, err := cloneFromDir(src, targetDir); err != nil {
			return fmt.Errorf("copy project from %s: %w", src, err)
		} else {
			projectDir = p
		}
	default:
		return fmt.Errorf("unknown source %s", src)
	}

	manifestFile := filepath.Join(projectDir, ManifestFile)
	manifest, err := LoadManifestFromFile(manifestFile)
	if err != nil {
		return fmt.Errorf("load manifest %s: %w", manifestFile, err)
	}

	err = manifest.RenderTo(ctx, out, in, manifestFile, targetDir)
	if err != nil {
		return fmt.Errorf("render: %w", err)
	}

	return nil
}

func cloneFromDir(srcDir string, targetDir string) (projectDir string, err error) {
	if err := copyTree(filepath.Join(srcDir, ContentDir), targetDir); err != nil {
		return "", fmt.Errorf("copy content from %s: %w", srcDir, err)
	}
	return srcDir, nil
}

func copyTree(src string, dest string) error {
	return filepath.Walk(src, func(path string, info fs.FileInfo, err error) error {
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
}
