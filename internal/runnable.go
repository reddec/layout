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
	"path"
	"path/filepath"
	"strings"

	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
)

// execute hook as script (priority) or inline shell. Shell is platform-independent, thanks to mvdan.cc/sh.
func (h Runnable) execute(ctx context.Context, renderContext *renderContext, workDir string, layoutFS string) error {
	cp, err := h.render(renderContext)
	if err != nil {
		return fmt.Errorf("render hook: %w", err)
	}
	if cp.Script != "" {
		return cp.executeScript(ctx, renderContext, workDir, layoutFS)
	}
	return cp.executeInline(ctx, workDir)
}

// execute inline (run) shell script.
func (h Runnable) executeInline(ctx context.Context, workDir string) error {
	script, err := syntax.NewParser().Parse(strings.NewReader(h.Run), "")
	if err != nil {
		return fmt.Errorf("parse script: %w", err)
	}

	runner, err := interp.New(interp.Dir(workDir), interp.StdIO(nil, os.Stdout, os.Stderr))
	if err != nil {
		return fmt.Errorf("create script runner: %w", err)
	}

	return runner.Run(ctx, script)
}

// render script to temporary file and execute it. Automatically sets +x (executable) flag to file.
// It CAN support more or less complex shell execution, however, it designed for direct script invocation: <script> [args...]
func (h Runnable) executeScript(ctx context.Context, renderContext *renderContext, workDir string, layoutFS string) error {
	parsedCommand, err := syntax.NewParser().Parse(strings.NewReader(h.Script), "")
	if err != nil {
		return fmt.Errorf("parse script invokation: %w", err)
	}
	var callExpr *syntax.CallExpr
	for _, stmt := range parsedCommand.Stmts {
		if call, ok := stmt.Cmd.(*syntax.CallExpr); ok && len(call.Args) > 0 {
			callExpr = call
			break
		}
	}

	if callExpr != nil {
		// render script content and copy it to temp dir

		scriptContent, err := ioutil.ReadFile(filepath.Join(layoutFS, path.Clean(assemblePathToCommand(callExpr))))
		if err != nil {
			return fmt.Errorf("read hook script content: %w", err)
		}

		newScriptContent, err := renderContext.Render(string(scriptContent))
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

		// mock call in expression
		callExpr.Args[0].Parts[0] = &syntax.Lit{Value: f.Name()}
	}

	runner, err := interp.New(interp.Dir(workDir), interp.StdIO(nil, os.Stdout, os.Stderr))
	if err != nil {
		return fmt.Errorf("create script runner: %w", err)
	}

	return runner.Run(ctx, parsedCommand)
}

// render templated variables: run, script
func (h Runnable) render(renderer *renderContext) (Runnable, error) {
	if v, err := renderer.Render(h.Run); err != nil {
		return h, fmt.Errorf("render run: %w", err)
	} else {
		h.Run = v
	}
	if v, err := renderer.Render(h.Script); err != nil {
		return h, fmt.Errorf("render script: %w", err)
	} else {
		h.Script = v
	}
	return h, nil
}

// describe what will be executed: path to script or shell command
func (h Runnable) what() string {
	if h.Script != "" {
		return h.Script
	}
	return h.Run
}

func assemblePathToCommand(stmt *syntax.CallExpr) string {
	var ans []string = make([]string, 0, len(stmt.Args[0].Parts))
	for _, p := range stmt.Args[0].Parts {
		switch v := p.(type) {
		case *syntax.SglQuoted:
			ans = append(ans, v.Value)
		case *syntax.Lit:
			ans = append(ans, v.Value)
		}
	}
	return strings.Join(ans, "")
}
