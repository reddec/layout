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
)

// compute variable: render value (if string) and convert it to desired type.
// If value is not string, it will be returned as-is
func (c Computed) compute(ctx context.Context, renderer *renderContext) error {
	ok, err := c.When.Ok(ctx, renderer.State())
	if err != nil {
		return fmt.Errorf("evalute condition: %w", err)
	}

	if !ok {
		return nil
	}

	stringValue, ok := c.Value.(string)
	if !ok {
		renderer.Save(c.Var, c.Value)
		return nil
	}

	value, err := renderer.Render(stringValue)
	if err != nil {
		return fmt.Errorf("render value: %w", err)
	}

	parsed, err := c.Type.Parse(value)
	if err != nil {
		return fmt.Errorf("parse value: %w", err)
	}
	renderer.Save(c.Var, parsed)
	return nil
}

// condition-less default variable: render value (if string) and convert it to desired type.
// If value is not string, it will be returned as-is
func (d Default) compute(renderer *renderContext) error {
	stringValue, ok := d.Value.(string)
	if !ok {
		renderer.Save(d.Var, d.Value)
		return nil
	}

	value, err := renderer.Render(stringValue)
	if err != nil {
		return fmt.Errorf("render value: %w", err)
	}

	parsed, err := d.Type.Parse(value)
	if err != nil {
		return fmt.Errorf("parse value: %w", err)
	}
	renderer.Save(d.Var, parsed)
	return nil
}
