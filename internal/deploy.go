package internal

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
)

// Config of layout deployment.
type Config struct {
	Source  string            // git URL, shorthand, or path to directory
	Target  string            // destination directory
	Input   *bufio.Reader     // input stream for parameters, default is based on os.Stdin
	Output  io.Writer         // output stream for questions, default is os.Stdout
	Aliases map[string]string // aliases (abbreviations) for cloning, values may contain {0} placeholder
	Default string            // default alias (for cloning without abbreviations, such as owner/repo), value may contain {0} placeholder, default is Github
}

func (cfg Config) withDefaults() Config {
	if cfg.Input == nil {
		cfg.Input = bufio.NewReader(os.Stdin)
	}
	if cfg.Output == nil {
		cfg.Output = os.Stdout
	}
	if cfg.Default == "" {
		cfg.Default = "git@github.com:{0}.git"
	}
	return cfg
}

// Deploy layout, which means clone repo, ask for question, and template content.
func Deploy(ctx context.Context, config Config) error {
	config = config.withDefaults()

	var projectDir string

	// strategy
	// - try as directory
	// - try as default
	// - try as aliased
	// - try as git URL

	info, err := os.Stat(config.Source)
	alias, repo := splitAbbreviation(config.Source)
	repoTemplate, aliasExist := config.Aliases[alias]
	url := config.Source

	switch {
	case err == nil && info.IsDir():
		projectDir = config.Source
	case !strings.Contains(config.Source, ":"):
		// this is default case since url should contain either abbreviation or protocol delimited by :
		repoTemplate = config.Default
		fallthrough
	case aliasExist:
		url = strings.ReplaceAll(repoTemplate, "{0}", repo)
		fallthrough
	default:
		tmpDir, err := cloneFromGit(ctx, url)
		if err != nil {
			return fmt.Errorf("copy project from git %s: %w", url, err)
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

	err = manifest.RenderTo(ctx, config.Output, config.Input, manifestFile, config.Target, sourceDir)
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

func splitAbbreviation(text string) (abbrev, repo string) {
	parts := strings.SplitN(text, ":", 2)
	if len(parts) == 1 {
		return "", text
	}
	return parts[0], parts[1]
}