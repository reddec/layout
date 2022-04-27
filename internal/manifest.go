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
	"bufio"
	"context"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"
)

const (
	MagicVarDir = "dirname" // contains base name of destination directory (aka: project name)
)

func LoadManifestFromFile(file string) (*Manifest, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var m Manifest
	return &m, yaml.NewDecoder(f).Decode(&m)
}

func (m *Manifest) Render(ctx context.Context, manifestFile, contentDir, sourceDir string) error {
	return m.RenderTo(ctx, os.Stdout, bufio.NewReader(os.Stdin), manifestFile, contentDir, sourceDir)
}

func (m *Manifest) RenderTo(ctx context.Context, out io.Writer, in *bufio.Reader, manifestFile, contentDir string, sourceDir string) error {
	if m.Title != "" {
		if _, err := fmt.Fprintln(out, m.Title); err != nil {
			return fmt.Errorf("print title: %w", err)
		}
	}
	source := os.DirFS(filepath.Dir(manifestFile))
	var state = make(map[string]interface{})
	// set magic variables
	state[MagicVarDir] = filepath.Base(contentDir)

	if err := AskState(ctx, out, in, m.Prompts, manifestFile, source, state); err != nil {
		return fmt.Errorf("get values for prompts: %w", err)
	}

	for i, c := range m.Computed {
		if err := c.compute(ctx, state); err != nil {
			return fmt.Errorf("compute value #%d (%s): %w", i, c.Var, err)
		}
	}

	// here there is sense to copy content, not before state computation
	if err := os.MkdirAll(contentDir, 0755); err != nil {
		return fmt.Errorf("create destination: %w", err)
	}

	if _, err := CopyTree(sourceDir, contentDir); err != nil {
		return fmt.Errorf("copy content: %w", err)
	}

	// execute pre-generate
	for i, h := range m.Before {
		if err := h.Execute(ctx, state, manifestFile, contentDir); err != nil {
			return fmt.Errorf("execute pre-generate hook #%d: %w", i, err)
		}
	}

	// render template
	// rename files and dirs, empty entries removed
	err := walk(contentDir, func(dir string, d fs.DirEntry) error {
		renderedName, err := render(d.Name(), state)
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
	ignoredFiles, err := m.filesToIgnore(contentDir)
	if err != nil {
		return fmt.Errorf("calculate which files to ignore: %w", err)
	}
	err = filepath.Walk(contentDir, func(path string, info fs.FileInfo, err error) error {
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
		data, err := render(string(templateData), state)
		if err != nil {
			return fmt.Errorf("render %s: %w", path, err)
		}
		return ioutil.WriteFile(path, []byte(data), info.Mode())
	})

	// exec post-generate
	for i, h := range m.After {
		if err := h.Execute(ctx, state, manifestFile, contentDir); err != nil {
			return fmt.Errorf("execute post-generate hook #%d: %w", i, err)
		}
	}

	return nil
}

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

// walk is customized implementation of filepath.WalkDir which supports modifications in handler.
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
