package internal

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"text/template"

	"github.com/Masterminds/sprig/v3"

	"github.com/d5/tengo/v2"
	"gopkg.in/yaml.v2"
)

func Ask(ctx context.Context, prompts []Prompt, baseFile string) (map[string]interface{}, error) {
	var state = make(map[string]interface{})
	rootDir := "."
	if baseFile != "" {
		rootDir = filepath.Dir(baseFile)
	}
	return state, AskState(ctx, os.Stdout, bufio.NewReader(os.Stdin), prompts, baseFile, os.DirFS(rootDir), state)
}

func AskState(ctx context.Context, out io.Writer, in *bufio.Reader, prompts []Prompt, baseFile string, source fs.FS, state map[string]interface{}) error {
	for i, prompt := range prompts {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		if prompt.When != "" {
			execute, err := prompt.When.Eval(ctx, state)
			if err != nil {
				return fmt.Errorf("condition in step %d in %s: %w", i, baseFile, err)
			}
			if !execute {
				continue
			}
		}

		prompt, err := prompt.Render(state)
		if err != nil {
			return fmt.Errorf("render step %d in %s: %w", i, baseFile, err)
		}

		if prompt.Include != "" {
			children, childFile, err := include(prompt.Include, baseFile, source)
			if err != nil {
				return fmt.Errorf("step %d, file %s, include %s: %w", i, baseFile, prompt.Include, err)
			}
			if err := AskState(ctx, out, in, children, childFile, source, state); err != nil {
				return fmt.Errorf("step %d, file %s, process include %s: %w", i, baseFile, prompt.Include, err)
			}
			continue
		}

		// retry loop
		for {
			value, err := prompt.ask(out, in)
			if errors.Is(err, io.EOF) || errors.Is(err, os.ErrClosed) {
				return err
			}
			if err != nil {
				if _, err := fmt.Fprintln(out, "Something went wrong:", err); err != nil {
					return fmt.Errorf("ask value for step %d in %s: %w", i, baseFile, err)
				}
				continue
			}
			state[prompt.Var] = value
			break
		}
	}

	return ctx.Err()
}

func (p Condition) Eval(ctx context.Context, state map[string]interface{}) (bool, error) {
	if p == "" {
		return false, nil
	}
	res, err := tengo.Eval(ctx, string(p), state)
	if err != nil {
		return false, err
	}
	if v, ok := res.(bool); ok {
		return v, nil
	}
	return false, fmt.Errorf("condition returned not boolean")
}

func (p Condition) Ok(ctx context.Context, state map[string]interface{}) (bool, error) {
	if p == "" {
		return true, nil
	}
	return p.Eval(ctx, state)
}

func include(includeFile string, baseFile string, source fs.FS) ([]Prompt, string, error) {
	file := filepath.Join(path.Dir(baseFile), includeFile)
	var prompts []Prompt

	f, err := source.Open(file)
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
