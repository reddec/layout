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
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"text/template"

	"layout/internal/ui"

	"github.com/Masterminds/sprig/v3"

	"github.com/d5/tengo/v2"
	"gopkg.in/yaml.v2"
)

// Ask questions to user and generate state. Base file initially equal to manifest file and used to resolve relative includes.
func askState(ctx context.Context, display ui.UI, prompts []Prompt, baseFile string, layoutDir string, state map[string]interface{}) error {
	for i, prompt := range prompts {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// we may skip the prompt in case condition is not empty and returns non-true
		if prompt.When != "" {
			execute, err := prompt.When.Eval(ctx, state)
			if err != nil {
				return fmt.Errorf("condition in step %d in %s: %w", i, baseFile, err)
			}
			if !execute {
				continue
			}
		}

		// before processing prompt further we have to render all templated fields
		prompt, err := prompt.render(state)
		if err != nil {
			return fmt.Errorf("render step %d in %s: %w", i, baseFile, err)
		}

		// in case prompt is include we will recursive process file and NOT process the prompt as general variable
		if prompt.Include != "" {
			children, childFile, err := include(prompt.Include, baseFile, layoutDir)
			if err != nil {
				return fmt.Errorf("step %d, file %s, include %s: %w", i, baseFile, prompt.Include, err)
			}
			if err := askState(ctx, display, children, childFile, layoutDir, state); err != nil {
				return fmt.Errorf("step %d, file %s, process include %s: %w", i, baseFile, prompt.Include, err)
			}
			continue
		}

		// in case of failed user input we will retry again and again till Stdin or context closed
		for {
			value, err := prompt.ask(ctx, display)
			if errors.Is(err, io.EOF) || errors.Is(err, os.ErrClosed) {
				return fmt.Errorf("ask value for %s (step %d) in %s: %w", prompt.Var, i, baseFile, err)
			}
			if err != nil {
				if err := display.Error(ctx, err.Error()); err != nil {
					return fmt.Errorf("show error for value for step %d in %s: %w", i, baseFile, err)
				}
				continue
			}
			state[prompt.Var] = value
			break
		}
	}

	return ctx.Err()
}

// Eval returns true only in case evaluated expression (in Tengo language) returns true.
func (p Condition) Eval(ctx context.Context, state map[string]interface{}) (bool, error) {
	if p == "" {
		return false, nil
	}
	res, err := tengo.Eval(ctx, string(p), sanitizeState(state))
	if err != nil {
		return false, err
	}
	if v, ok := res.(bool); ok {
		return v, nil
	}
	return false, fmt.Errorf("condition returned not boolean")
}

// Ok is corner case of Eval and returns true in case expression is not set, otherwise it returns result of Eval.
func (p Condition) Ok(ctx context.Context, state map[string]interface{}) (bool, error) {
	if p == "" {
		return true, nil
	}
	return p.Eval(ctx, state)
}

// include YAML file with relative to baseFile (which is also relative to layoutFS) path with list of prompts. File could be multi-document.
func include(includeFile string, baseFile string, layoutFS string) ([]Prompt, string, error) {
	file := filepath.Join(path.Dir(baseFile), path.Clean(includeFile))
	var prompts []Prompt

	f, err := os.Open(filepath.Join(layoutFS, file))
	if err != nil {
		return nil, file, err
	}
	defer f.Close()

	decoder := yaml.NewDecoder(f)
	// this is multi-document support
	for {
		var batch []Prompt
		if err := decoder.Decode(&batch); err == nil {
			prompts = append(prompts, batch...)
		} else if errors.Is(err, io.EOF) {
			break
		} else {
			return nil, file, err
		}
	}

	return prompts, file, nil
}

// render all templated values in render: label, include, default, options.
func (p Prompt) render(state map[string]interface{}) (Prompt, error) {
	if v, err := render(p.Label, state); err != nil {
		return p, fmt.Errorf("render label: %w", err)
	} else {
		p.Label = v
	}

	if v, err := render(p.Include, state); err != nil {
		return p, fmt.Errorf("render include: %w", err)
	} else {
		p.Include = v
	}

	if v, err := render(p.Default, state); err != nil {
		return p, fmt.Errorf("render default: %w", err)
	} else {
		p.Default = v
	}

	options := make([]string, 0, len(p.Options))
	for i, opt := range p.Options {
		if v, err := render(opt, state); err != nil {
			return p, fmt.Errorf("render option %d: %w", i, err)
		} else {
			options = append(options, v)
		}
	}
	p.Options = options

	return p, nil
}

// render is helper for rendering go-template value with state in memory.
func render(value string, state map[string]interface{}) (string, error) {
	p, err := template.New("").Funcs(sprig.TxtFuncMap()).Parse(value)
	if err != nil {
		return "", err
	}
	var out bytes.Buffer
	err = p.Execute(&out, state)
	return out.String(), err
}

// prepares state for Tengo
func sanitizeState(state map[string]interface{}) map[string]interface{} {
	ng := make(map[string]interface{}, len(state))
	for k, v := range state {

		if s, ok := v.([]string); ok { // tengo can not understand other arrays except []byte or []interface{}
			var ans = make([]interface{}, 0, len(s))
			for _, item := range s {
				ans = append(ans, item)
			}
			v = ans
		}
		ng[k] = v
	}
	return ng
}
