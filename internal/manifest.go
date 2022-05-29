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
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/reddec/layout/internal/ui"

	"github.com/Masterminds/semver"
	"github.com/davecgh/go-spew/spew"
	"gopkg.in/yaml.v3"
)

const (
	MagicVarDir = "dirname" // contains base name of destination directory (aka: project name)
)

// Loads YAML manifest from file, does not support multi-document format.
func loadManifest(file string) (*Manifest, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var m Manifest
	return &m, yaml.NewDecoder(f).Decode(&m)
}

// Communicates with user and renders all templates and executes hooks. Debug flag enables state dump to stdout
// after user input. Once flags disables retry on wrong user input.
func (m *Manifest) renderTo(ctx context.Context, display ui.UI, destinationDir, layoutDir string, debug, once bool, initialState map[string]interface{}) error {
	welcomeMessage := strings.TrimSpace(strings.Join([]string{m.Title, m.Description}, "\n\n"))
	if welcomeMessage != "" {
		if err := display.Title(ctx, welcomeMessage); err != nil {
			return fmt.Errorf("show welcome message: %w", err)
		}
	}
	var state = make(map[string]interface{})
	for k, v := range initialState {
		state[k] = v
	}
	// set required magic variables
	state[MagicVarDir] = filepath.Base(destinationDir)
	renderer := newRenderContext(state).Delimiters(m.Delimiters.Open, m.Delimiters.Close)

	for i, c := range m.Default {
		if err := c.compute(renderer); err != nil {
			return fmt.Errorf("set default value #%d (%s): %w", i, c.Var, err)
		}
	}

	if err := askState(ctx, display, m.Prompts, "", layoutDir, renderer, once); err != nil {
		return fmt.Errorf("get values for prompts: %w", err)
	}

	for i, c := range m.Computed {
		if err := c.compute(ctx, renderer); err != nil {
			return fmt.Errorf("compute value #%d (%s): %w", i, c.Var, err)
		}
	}

	if debug {
		spew.Dump(state)
	}

	// here there is sense to copy content, not before state computation
	if err := os.MkdirAll(destinationDir, 0755); err != nil {
		return fmt.Errorf("create destination: %w", err)
	}

	if _, err := CopyTree(filepath.Join(layoutDir, ContentDir), destinationDir); err != nil {
		return fmt.Errorf("copy content: %w", err)
	}

	// execute pre-generate
	for i, h := range m.Before {
		if ok, err := h.When.Ok(ctx, state); err != nil {
			return fmt.Errorf("evaluate condition of pre-generate hook #%d (%s): %w", i, h.what(), err)
		} else if !ok {
			continue
		}
		if err := h.display(ctx, display.Info); err != nil {
			return fmt.Errorf("display pre-generate hook #%d (%s): %w", i, h.what(), err)
		}
		if err := h.execute(ctx, renderer, destinationDir, layoutDir); err != nil {
			return fmt.Errorf("execute pre-generate hook #%d (%s): %w", i, h.what(), err)
		}
	}

	// render template
	// rename files and dirs, empty entries removed
	err := walk(destinationDir, func(dir string, d fs.DirEntry) error {
		renderedName, err := renderer.Render(d.Name())
		if err != nil {
			return err
		}
		renderedName = strings.TrimSpace(renderedName)
		oldPath := filepath.Join(dir, d.Name())
		newPath := filepath.Join(dir, renderedName)
		if len(renderedName) == 0 {
			return os.RemoveAll(oldPath)
		}
		if oldPath == newPath {
			return nil
		}
		return os.Rename(oldPath, newPath)
	})
	if err != nil {
		return fmt.Errorf("render files names: %w", err)
	}
	// render file contents as template, except ignored
	ignoredFiles, err := m.filesToIgnore(destinationDir)
	if err != nil {
		return fmt.Errorf("calculate which files to ignore: %w", err)
	}
	err = filepath.Walk(destinationDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || ignoredFiles[path] {
			return nil
		}
		templateData, err := ioutil.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read content of %s: %w", path, err)
		}
		data, err := renderer.Render(string(templateData))
		if err != nil {
			return fmt.Errorf("render %s: %w", path, err)
		}
		return ioutil.WriteFile(path, []byte(data), info.Mode())
	})
	if err != nil {
		return fmt.Errorf("render: %w", err)
	}

	// exec post-generate
	for i, h := range m.After {
		if ok, err := h.When.Ok(ctx, state); err != nil {
			return fmt.Errorf("evaluate condition of post-generate hook #%d (%s): %w", i, h.what(), err)
		} else if !ok {
			continue
		}
		if err := h.display(ctx, display.Info); err != nil {
			return fmt.Errorf("display post-generate hook #%d (%s): %w", i, h.what(), err)
		}
		if err := h.execute(ctx, renderer, destinationDir, layoutDir); err != nil {
			return fmt.Errorf("execute post-generate hook #%d (%s): %w", i, h.what(), err)
		}
	}

	return nil
}

// generates list of files which should be not rendered as template.
// Executes AFTER rendering file names.
func (m *Manifest) filesToIgnore(contentDir string) (map[string]bool, error) {
	var set = make(map[string]bool)
	for i, pattern := range m.Ignore {
		list, err := filepath.Glob(filepath.Join(contentDir, pattern))
		if err != nil {
			return nil, fmt.Errorf("match files in ignore pattern #%d (%s): %w", i, pattern, err)
		}
		for _, file := range list {
			set[file] = true
		}
	}
	return set, nil
}

// walk is customized implementation of filepath.WalkDir which supports FS modifications in handler.
func walk(path string, handler func(dir string, stat os.DirEntry) error) error {
	list, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	for _, item := range list {
		err = handler(path, item)
		if err != nil {
			return err
		}
	}
	// re-read dir in case something was changed
	list, err = os.ReadDir(path)
	if err != nil {
		return err
	}
	for _, item := range list {
		if item.IsDir() {
			err = walk(filepath.Join(path, item.Name()), handler)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// validates manifest version against current layout binary version.
// Always returns true if no version provided or no constraints in manifest. Otherwise, uses semver semantic to check.
func (m *Manifest) isSupportedVersion(currentVersion string) (bool, error) {
	if currentVersion == "" || m.Version == "" {
		return true, nil
	}
	version, err := semver.NewVersion(strings.Trim(currentVersion, "v"))
	if err != nil {
		return false, fmt.Errorf("parse current version: %w", err)
	}
	constraint, err := semver.NewConstraint(m.Version)
	if err != nil {
		return false, fmt.Errorf("parse version constraint in manifest: %w", err)
	}
	return constraint.Check(version), nil
}

func newRenderContext(initialState map[string]interface{}) *renderContext {
	return &renderContext{
		state: initialState,
		open:  "{{",
		close: "}}",
	}
}

// renderContext aggregates required information for rendering templates.
type renderContext struct {
	state map[string]interface{}
	open  string
	close string
}

// Delimiters which will be used in template. Default is {{ and }}.
func (r *renderContext) Delimiters(open, close string) *renderContext {
	if open != "" {
		r.open = open
	}
	if close != "" {
		r.close = close
	}
	return r
}

// State of context with all known variables.
func (r *renderContext) State() map[string]interface{} {
	return r.state
}

// Save value in the state
func (r *renderContext) Save(key string, value interface{}) {
	if r.state == nil {
		r.state = make(map[string]interface{})
	}
	r.state[key] = value
}

// Render go-template value with state as context in memory.
func (r *renderContext) Render(value string) (string, error) {
	funcMap := sprig.TxtFuncMap()
	funcMap["getRootFile"] = getRootFile
	funcMap["findRootFile"] = findRootFile
	funcMap["findRootDir"] = findRootDir
	p, err := template.New("").Delims(r.open, r.close).Funcs(funcMap).Parse(value)
	if err != nil {
		return "", err
	}
	var out bytes.Buffer
	err = p.Execute(&out, r.state)
	return out.String(), err
}

// get content of file with specific name (can be only base name) in any of root folders:
//
//     WD: /foo/bar/xyz
//     Name: .gitignore
//     Will check:
//        /foo/bar/xyz/.gitignore
//        /foo/bar/.gitignore
//        /foo/.gitignore
//        /.gitignore
//
// If nothing found - ErrNotExists returned
func getRootFile(name string) (string, error) {
	root, err := os.Getwd()
	if err != nil {
		return "", err
	}
	name = filepath.Base(name)
	for {
		file := filepath.Join(root, name)
		content, err := ioutil.ReadFile(file)
		if err == nil {
			return string(content), nil
		}
		if !os.IsNotExist(err) {
			return "", fmt.Errorf("read %s: %w", file, err)
		}
		next := filepath.Dir(root)
		if next == root {
			return "", os.ErrNotExist
		}
		root = next
	}
}

// find path to file with specific name (can be only base name) in any of root folders:
//
//     WD: /foo/bar/xyz
//     Name: .gitignore
//     Will check:
//        /foo/bar/xyz/.gitignore
//        /foo/bar/.gitignore
//        /foo/.gitignore
//        /.gitignore
//
// If nothing found - ErrNotExists returned
func findRootFile(name string) (string, error) {
	root, err := os.Getwd()
	if err != nil {
		return "", err
	}
	name = filepath.Base(name)
	for {
		file := filepath.Join(root, name)
		if stat, err := os.Stat(file); err == nil && !stat.IsDir() {
			return file, nil
		} else if err != nil && !os.IsNotExist(err) {
			return "", fmt.Errorf("stat %s: %w", file, err)
		}
		next := filepath.Dir(root)
		if next == root {
			return "", os.ErrNotExist
		}
		root = next
	}
}

// find path to directory with specific name (can be only base name) in any of root folders:
//
//     WD: /foo/bar/xyz
//     Name: .git
//     Will check:
//        /foo/bar/xyz/.git
//        /foo/bar/.git
//        /foo/.git
//        /.git
//
// If nothing found - ErrNotExists returned
func findRootDir(name string) (string, error) {
	root, err := os.Getwd()
	if err != nil {
		return "", err
	}
	name = filepath.Base(name)
	for {
		dirPath := filepath.Join(root, name)
		if stat, err := os.Stat(dirPath); err == nil && stat.IsDir() {
			return dirPath, nil
		} else if err != nil && !os.IsNotExist(err) {
			return "", fmt.Errorf("stat %s: %w", dirPath, err)
		}
		next := filepath.Dir(root)
		if next == root {
			return "", os.ErrNotExist
		}
		root = next
	}
}
