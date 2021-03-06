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
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/reddec/layout/internal/ui"

	"github.com/d5/tengo/v2"
	"gopkg.in/yaml.v3"
)

// Ask questions to user and generate state. Base file initially equal to manifest file and used to resolve relative includes.
func askState(ctx context.Context, display ui.UI, prompts []Prompt, baseFile string, layoutDir string, renderContext *renderContext, once bool) error {
	for i, prompt := range prompts {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// we may skip the prompt in case condition is not empty and returns non-true
		if prompt.When != "" {
			execute, err := prompt.When.Eval(ctx, renderContext.State())
			if err != nil {
				return fmt.Errorf("condition in step %d in %s: %w", i, baseFile, err)
			}
			if !execute {
				continue
			}
		}

		// before processing prompt further we have to render all templated fields
		prompt, err := prompt.render(renderContext)
		if err != nil {
			return fmt.Errorf("render step %d in %s: %w", i, baseFile, err)
		}

		// in case prompt is include we will recursive process file and NOT process the prompt as general variable
		if prompt.Include != "" {
			children, childFile, err := include(prompt.Include, baseFile, layoutDir)
			if err != nil {
				return fmt.Errorf("step %d, file %s, include %s: %w", i, baseFile, prompt.Include, err)
			}
			if err := askState(ctx, display, children, childFile, layoutDir, renderContext, once); err != nil {
				return fmt.Errorf("step %d, file %s, process include %s: %w", i, baseFile, prompt.Include, err)
			}
			continue
		}

		// in case of failed user input we will retry again and again till Stdin or context closed
		for {
			value, err := prompt.ask(ctx, display)
			if errors.Is(err, io.EOF) || errors.Is(err, os.ErrClosed) || errors.Is(err, ui.ErrInterrupted) {
				return fmt.Errorf("ask value for %s (step %d) in %s: %w", prompt.Var, i, baseFile, err)
			}
			if err != nil {
				if once {
					return fmt.Errorf("get value for %s (step %d) in %s: %w", prompt.Var, i, baseFile, err)
				}
				if err := display.Error(ctx, err.Error()); err != nil {
					return fmt.Errorf("show error for value for step %d in %s: %w", i, baseFile, err)
				}
				continue
			}
			renderContext.Save(prompt.Var, value)
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
	expr := strings.TrimSpace(string(p))
	script := tengo.NewScript([]byte(fmt.Sprintf("__res__ := (%s)", expr)))
	for pk, pv := range sanitizeState(state) {
		err := script.Add(pk, pv)
		if err != nil {
			return false, fmt.Errorf("script add: %w", err)
		}
	}
	// helpers
	if err := script.Add("has", hasHelper); err != nil {
		return false, fmt.Errorf("add 'has' helper: %w", err)
	}

	compiled, err := script.RunContext(ctx)
	if err != nil {
		return false, fmt.Errorf("script run: %w", err)
	}
	res := compiled.Get("__res__").Value()
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
func (p Prompt) render(renderer *renderContext) (Prompt, error) {
	if v, err := renderer.Render(p.Label); err != nil {
		return p, fmt.Errorf("render label: %w", err)
	} else {
		p.Label = v
	}

	if v, err := renderer.Render(p.Include); err != nil {
		return p, fmt.Errorf("render include: %w", err)
	} else {
		p.Include = v
	}

	if s, ok := p.Default.(string); ok {
		if v, err := renderer.Render(s); err != nil {
			return p, fmt.Errorf("render default: %w", err)
		} else {
			p.Default = v
		}
	}

	options := make([]string, 0, len(p.Options))
	for i, opt := range p.Options {
		if v, err := renderer.Render(opt); err != nil {
			return p, fmt.Errorf("render option %d: %w", i, err)
		} else {
			options = append(options, v)
		}
	}
	p.Options = options

	return p, nil
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

// returns true if first argument is list and contains second argument
func hasHelper(args ...tengo.Object) (ret tengo.Object, err error) {
	if len(args) != 2 {
		return tengo.UndefinedValue, fmt.Errorf("2 arguments required")
	}
	seq := args[0]
	if !seq.CanIterate() {
		return tengo.UndefinedValue, fmt.Errorf("first argument should be iterable")
	}
	item := args[1]

	for it := seq.Iterate(); it.Next(); {
		if it.Value().Equals(item) {
			return tengo.TrueValue, nil
		}
	}
	return tengo.FalseValue, nil
}
