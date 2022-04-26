package internal

import (
	"context"
	"fmt"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
	"strings"
)

func (h Hook) Execute(ctx context.Context, state map[string]interface{}, workDir string) error {
	ok, err := h.When.Ok(ctx, state)
	if err != nil {
		return fmt.Errorf("evalute condition: %w", err)
	}
	if !ok {
		return nil
	}

	cp, err := h.render(state)
	if err != nil {
		return fmt.Errorf("render hook: %w", err)
	}

	script, err := syntax.NewParser().Parse(strings.NewReader(cp.Run), "")
	if err != nil {
		return fmt.Errorf("parse script: %w", err)
	}

	runner, err := interp.New(interp.Dir(workDir))
	if err != nil {
		return fmt.Errorf("create script runner: %w", err)
	}

	return runner.Run(ctx, script)
}

func (h Hook) render(state map[string]interface{}) (Hook, error) {
	if v, err := render(h.Run, state); err != nil {
		return h, fmt.Errorf("render run: %w", err)
	} else {
		h.Run = v
	}

	return h, nil
}
