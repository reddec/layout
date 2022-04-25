package internal

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"

	"github.com/d5/tengo/v2"
	"gopkg.in/yaml.v2"
)

func Ask(ctx context.Context, prompts []Prompt, baseFile string) (map[string]interface{}, error) {
	var state = make(map[string]interface{})
	return state, AskState(ctx, prompts, baseFile, state)
}

func AskState(ctx context.Context, prompts []Prompt, baseFile string, state map[string]interface{}) error {
	for i, prompt := range prompts {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		skip, err := prompt.Skip(ctx, state)
		if err != nil {
			return fmt.Errorf("step %d in %s: %w", i, baseFile, err)
		}
		if skip {
			continue
		}

		if prompt.Include != "" {
			children, childFile, err := load(prompt.Include, baseFile)
			if err != nil {
				return fmt.Errorf("step %d, file %s, include %s: %w", i, baseFile, prompt.Include, err)
			}
			if err := AskState(ctx, children, childFile, state); err != nil {
				return fmt.Errorf("step %d, file %s, process include %s: %w", i, baseFile, prompt.Include, err)
			}
			continue
		}
		// TODO: UI ask
		// TODO: template
		prompt.Var
	}

	return ctx.Err()
}

func (p *Prompt) Skip(ctx context.Context, state map[string]interface{}) (bool, error) {
	if p.When == "" {
		return false, nil
	}
	res, err := tengo.Eval(ctx, p.When, state)
	if err != nil {
		return false, err
	}
	if v, ok := res.(bool); ok {
		return v, nil
	}
	return false, fmt.Errorf("when condition returned not boolean")
}

func load(includeFile string, baseFile string) ([]Prompt, string, error) {
	file := filepath.Join(path.Dir(baseFile), includeFile)
	var prompts []Prompt
	f, err := os.Open(file)
	if err != nil {
		return nil, file, err
	}
	defer f.Close()

	decoder := yaml.NewDecoder(f)
	// multi-document support
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
