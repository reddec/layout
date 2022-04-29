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
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"layout/internal/ui"
	"layout/internal/ui/simple"

	"github.com/go-git/go-git/v5"
)

const (
	defaultRepoTemplate = "git@github.com:{0}.git" // template which will be used when no abbreviation used (ex: reddec/template)
)

// Config of layout deployment.
type Config struct {
	Source  string            // git URL, shorthand, or path to directory
	Target  string            // destination directory
	Aliases map[string]string // aliases (abbreviations) for cloning, values may contain {0} placeholder
	Default string            // default alias (for cloning without abbreviations, such as owner/repo), value may contain {0} placeholder, default is Github
	Display ui.UI             // how to interact with user, default is Simple TUI
	Debug   bool              // enable debug messages and tracing
	Version string            // current version, used to filter manifests by constraints
	AskOnce bool              // do not try to ask for user input after wrong value and interrupt deployment
}

func (cfg Config) withDefaults() Config {
	if cfg.Default == "" {
		cfg.Default = defaultRepoTemplate
	}
	if cfg.Display == nil {
		cfg.Display = simple.Default()
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
	case err == nil && info.IsDir(): // first try as directory
		projectDir = config.Source
	case !strings.Contains(config.Source, ":"): // ok, let's try as remote. If we don't have delimiter it's shorthand for default template
		// this is default case since url should contain either abbreviation or protocol delimited by :
		repoTemplate = config.Default
		fallthrough
	case aliasExist: // we found abbreviation template
		url = strings.ReplaceAll(repoTemplate, "{0}", repo)
		fallthrough
	default: // finally all we need is to pull remote repository by URL
		tmpDir, err := cloneFromGit(ctx, url)
		if err != nil {
			return fmt.Errorf("copy project from git %s: %w", url, err)
		}
		defer os.RemoveAll(tmpDir)
		projectDir = tmpDir
	}
	manifestFile := filepath.Join(projectDir, ManifestFile)
	manifest, err := loadManifest(manifestFile)
	if err != nil {
		return fmt.Errorf("load manifest %s: %w", manifestFile, err)
	}

	if ok, err := manifest.isSupportedVersion(config.Version); err != nil {
		return fmt.Errorf("check manifest version: %w", err)
	} else if !ok {
		return fmt.Errorf("manifest version constraint (%s) requires another version of application (current %s)", manifest.Version, config.Version)
	}

	err = manifest.renderTo(ctx, config.Display, config.Target, projectDir, config.Debug, config.AskOnce)
	if err != nil {
		return fmt.Errorf("render: %w", err)
	}

	return nil
}

// clones from git repository from default branch with minimal depth (1).
// Reports progress to STDERR. Supports submodules.
// Returned directory should be removed by caller.
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
