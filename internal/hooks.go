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
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
)

func (h Hook) Execute(ctx context.Context, state map[string]interface{}, workDir string, layoutFS string) error {
	ok, err := h.When.Ok(ctx, state)
	if err != nil {
		return fmt.Errorf("evalute condition: %w", err)
	}
	if !ok {
		return nil
	}

	if h.Script != "" {
		return h.executeScript(ctx, state, workDir, layoutFS)
	}
	return h.executeInline(ctx, state, workDir)
}

func (h Hook) executeInline(ctx context.Context, state map[string]interface{}, workDir string) error {
	cp, err := h.render(state)
	if err != nil {
		return fmt.Errorf("render hook: %w", err)
	}

	script, err := syntax.NewParser().Parse(strings.NewReader(cp.Run), "")
	if err != nil {
		return fmt.Errorf("parse script: %w", err)
	}

	runner, err := interp.New(interp.Dir(workDir), interp.StdIO(nil, os.Stdout, os.Stderr))
	if err != nil {
		return fmt.Errorf("create script runner: %w", err)
	}

	return runner.Run(ctx, script)
}

func (h Hook) executeScript(ctx context.Context, state map[string]interface{}, workDir string, layoutFS string) error {
	scriptContent, err := ioutil.ReadFile(filepath.Join(layoutFS, path.Clean(h.Script)))
	if err != nil {
		return fmt.Errorf("read hook script content: %w", err)
	}

	newScriptContent, err := render(string(scriptContent), state)
	if err != nil {
		return fmt.Errorf("render hook script content: %w", err)
	}

	f, err := os.CreateTemp("", "")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	defer os.RemoveAll(f.Name())
	defer f.Close()

	if _, err := f.WriteString(newScriptContent); err != nil {
		return fmt.Errorf("write rendered hook content: %w", err)
	}

	if err := f.Close(); err != nil {
		return fmt.Errorf("close script: %w", err)
	}

	if err := os.Chmod(f.Name(), 0700); err != nil {
		return fmt.Errorf("mark script as executable: %w", err)
	}

	cmd := exec.CommandContext(ctx, f.Name())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = workDir

	return cmd.Run()
}

func (h Hook) render(state map[string]interface{}) (Hook, error) {
	if v, err := render(h.Run, state); err != nil {
		return h, fmt.Errorf("render run: %w", err)
	} else {
		h.Run = v
	}

	return h, nil
}

func (h Hook) what() string {
	if h.Script != "" {
		return h.Script
	}
	return h.Run
}
