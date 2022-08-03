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

	"github.com/reddec/layout/internal/gitclient"
	"github.com/reddec/layout/internal/ui"
	"github.com/reddec/layout/internal/ui/simple"
)

const (
	defaultRepoTemplate = "git@github.com:{0}.git" // template which will be used when no abbreviation used (ex: reddec/template)
)

// Config of layout deployment.
type Config struct {
	Source   string                 // git URL, shorthand, or path to directory
	Target   string                 // destination directory
	Aliases  map[string]string      // aliases (abbreviations) for cloning, values may contain {0} placeholder
	Default  string                 // default alias (for cloning without abbreviations, such as owner/repo), value may contain {0} placeholder, default is Github
	Display  ui.UI                  // how to interact with user, default is Simple TUI
	Debug    bool                   // enable debug messages and tracing
	Version  string                 // current version, used to filter manifests by constraints
	AskOnce  bool                   // do not try to ask for user input after wrong value and interrupt deployment
	Git      gitclient.Client       // Git client, default is gitclient.Auto
	Defaults map[string]interface{} // Global default values
}

func (cfg Config) withDefaults(ctx context.Context) Config {
	if cfg.Default == "" {
		cfg.Default = defaultRepoTemplate
	}
	if cfg.Display == nil {
		cfg.Display = simple.Default()
	}
	if cfg.Git == nil {
		cfg.Git = gitclient.Auto(ctx)
	}
	return cfg
}

// Deploy layout, which means clone repo, ask for question, and template content.
func Deploy(ctx context.Context, config Config) error {
	config = config.withDefaults(ctx)

	targetDir, err := filepath.Abs(config.Target)
	if err != nil {
		return fmt.Errorf("calculate abs path: %w", err)
	}

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
		tmpDir, err := cloneFromGit(ctx, config.Git, url)
		if err != nil {
			return fmt.Errorf("copy project from git %s: %w", url, err)
		}
		defer os.RemoveAll(tmpDir)
		projectDir = tmpDir
	}

	manifestFiles, err := findManifests(projectDir)
	if err != nil {
		return fmt.Errorf("find manifests: %w", err)
	}
	if len(manifestFiles) == 0 {
		return fmt.Errorf("no manifests files discovered")
	}

	var manifestFile string

	if len(manifestFiles) == 1 {
		// pick first as default
		manifestFile = manifestFiles[0]
		projectDir = filepath.Dir(manifestFile)
	} else {
		// ask which manifest to use
		selectedManifest, err := selectManifest(ctx, config.Display, manifestFiles)
		if err != nil {
			return fmt.Errorf("ask for manifest: %w", err)
		}
		manifestFile = selectedManifest
		projectDir = filepath.Dir(selectedManifest)
	}

	manifest, err := loadManifest(manifestFile)
	if err != nil {
		return fmt.Errorf("load manifest %s: %w", manifestFile, err)
	}

	if ok, err := manifest.isSupportedVersion(config.Version); err != nil {
		return fmt.Errorf("check manifest version: %w", err)
	} else if !ok {
		return fmt.Errorf("manifest version constraint (%s) requires another version of application (current %s)", manifest.Version, config.Version)
	}

	err = manifest.renderTo(ctx, config.Display, targetDir, projectDir, config.Debug, config.AskOnce, config.Defaults)
	if err != nil {
		return fmt.Errorf("render: %w", err)
	}

	return nil
}

func selectManifest(ctx context.Context, display ui.UI, manifests []string) (string, error) {
	var options []string
	for _, m := range manifests {
		manifest, err := loadManifest(m)
		if err != nil {
			return "", fmt.Errorf("read manifest %s: %w", manifest, err)
		}
		options = append(options, manifest.Title)
	}
	picked, err := display.Select(ctx, "Which to use", options[0], options)
	if err != nil {
		return "", fmt.Errorf("select manifest: %w", err)
	}
	for i, opt := range options {
		if opt == picked {
			return manifests[i], nil
		}
	}
	return "", fmt.Errorf("picked unknown manifest")
}

// find manifests in root directory recursive. It will not scan directory with manifest file deeper.
func findManifests(rootDir string) ([]string, error) {
	var files []string
	err := filepath.Walk(rootDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return nil
		}
		manifestFile := filepath.Join(path, ManifestFile)
		stat, err := os.Stat(manifestFile)
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if stat.IsDir() {
			return nil
		}
		files = append(files, manifestFile)
		return filepath.SkipDir // do not go to layout dir
	})
	return files, err
}

// clones from git repository into temporary directory.
// Returned directory should be removed by caller.
func cloneFromGit(ctx context.Context, client gitclient.Client, url string) (projectDir string, err error) {
	tmpDir, err := os.MkdirTemp("", "layout-*")
	if err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}
	err = client(ctx, url, tmpDir)
	if err != nil {
		_ = os.RemoveAll(tmpDir)
		return "", fmt.Errorf("clone repo: %w", err)
	}

	return tmpDir, nil
}

func CopyTree(src string, dest string) (*FSTree, error) {
	var root = &FSTree{
		Name: dest,
		Dir:  true,
	}
	err := filepath.Walk(src, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == src {
			return nil
		}
		relPath, err := filepath.Rel(src, path)
		destPath := filepath.Join(dest, relPath)
		root.Add(relPath, info.IsDir())
		if info.IsDir() {
			err := os.Mkdir(destPath, info.Mode())
			if os.IsExist(err) {
				return nil
			}
			return err
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
	return root, err
}

func splitAbbreviation(text string) (abbrev, repo string) {
	parts := strings.SplitN(text, ":", 2)
	if len(parts) == 1 {
		return "", text
	}
	return parts[0], parts[1]
}
