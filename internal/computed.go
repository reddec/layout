package internal

import (
	"context"
	"fmt"
)

func (c Computed) compute(ctx context.Context, state map[string]interface{}) error {
	ok, err := c.When.Ok(ctx, state)
	if err != nil {
		return fmt.Errorf("evalute condition: %w", err)
	}

	if !ok {
		return nil
	}

	stringValue, ok := c.Value.(string)
	if !ok {
		state[c.Var] = c.Value
		return nil
	}

	value, err := render(stringValue, state)
	if err != nil {
		return fmt.Errorf("render value: %w", err)
	}

	parsed, err := c.Type.Parse(value)
	if err != nil {
		return fmt.Errorf("parse value: %w", err)
	}
	state[c.Var] = parsed
	return nil
}
