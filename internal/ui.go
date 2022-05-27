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
	"strings"

	"github.com/reddec/layout/internal/ui"
)

func (p Prompt) question() string {
	v := p.Label
	if p.Label == "" {
		v = p.Var
	}
	return strings.TrimRight(v, "?")
}

func (p Prompt) ask(ctx context.Context, display ui.Dialog) (interface{}, error) {
	switch p.Type {
	case VarList:
		if len(p.Options) == 0 {
			return display.Many(ctx, p.question(), p.defaultOptions())
		}
		return display.Choose(ctx, p.question(), p.defaultOptions(), p.Options)
	case "":
		p.Type = VarString
		fallthrough
	default:
		var v string
		var err error
		if len(p.Options) > 0 {
			v, err = display.Select(ctx, p.question(), p.defaultOption(), p.Options)
		} else {
			v, err = display.One(ctx, p.question(), p.defaultOption())
		}
		if err != nil {
			return nil, err
		}
		return p.Type.Parse(v)
	}
}

func (p Prompt) defaultOption() string {
	if s, ok := p.Default.(string); ok {
		return s
	}
	if p.Default == nil {
		return ""
	}
	return fmt.Sprint(p.Default)
}

func (p Prompt) defaultOptions() []string {
	if l, ok := p.Default.([]string); ok {
		return l
	}
	if arr, ok := p.Default.([]interface{}); ok {
		ans := make([]string, 0, len(arr))
		for _, v := range arr {
			ans = append(ans, fmt.Sprint(v))
		}
		return ans
	}
	opt := p.defaultOption()
	if opt == "" {
		return nil
	}
	return []string{opt}
}

func toBool(line string) bool {
	line = strings.ToLower(line)
	return line == "t" || line == "y" || line == "true" || line == "yes" || line == "ok"
}

func toList(line string) []string {
	var values []string
	line = strings.TrimSpace(line)
	for _, value := range strings.Split(line, ",") {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		values = append(values, value)
	}
	return values
}
