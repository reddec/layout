package internal

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/Masterminds/sprig/v3"
	"io"
	"os"
	"path"
	"path/filepath"
	"text/template"

	"github.com/d5/tengo/v2"
	"gopkg.in/yaml.v2"
)

func Ask(ctx context.Context, prompts []Prompt, baseFile string) (map[string]interface{}, error) {
	var state = make(map[string]interface{})
	return state, AskState(ctx, os.Stdout, bufio.NewReader(os.Stdin), prompts, baseFile, state)
}

func AskState(ctx context.Context, out io.Writer, in *bufio.Reader, prompts []Prompt, baseFile string, state map[string]interface{}) error {
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

		prompt, err = prompt.Render(state)
		if err != nil {
			return fmt.Errorf("render step %d in %s: %w", i, baseFile, err)
		}

		if prompt.Include != "" {
			children, childFile, err := load(prompt.Include, baseFile)
			if err != nil {
				return fmt.Errorf("step %d, file %s, include %s: %w", i, baseFile, prompt.Include, err)
			}
			if err := AskState(ctx, out, in, children, childFile, state); err != nil {
				return fmt.Errorf("step %d, file %s, process include %s: %w", i, baseFile, prompt.Include, err)
			}
			continue
		}

		value, err := prompt.ask(out, in)
		if err != nil {
			return fmt.Errorf("ask value for step %d in %s: %w", i, baseFile, err)
		}

		state[prompt.Var] = value
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
		return !v, nil
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

func render(value string, state map[string]interface{}) (string, error) {
	p, err := template.New("").Funcs(sprig.TxtFuncMap()).Parse(value)
	if err != nil {
		return "", err
	}
	var out bytes.Buffer
	err = p.Execute(&out, state)
	return out.String(), err
}

func (p Prompt) Render(state map[string]interface{}) (Prompt, error) {
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
