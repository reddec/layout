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

package simple

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

func New(in *bufio.Reader, out io.Writer) *UI {
	return &UI{
		in:  in,
		out: out,
	}
}

func Default() *UI {
	return New(bufio.NewReader(os.Stdin), os.Stdout)
}

type UI struct {
	in  *bufio.Reader
	out io.Writer
}

func (ui *UI) One(_ context.Context, question string, defaultValue string) (string, error) {
	if err := ui.print(question, "? "); err != nil {
		return "", err
	}
	if defaultValue != "" {
		if err := ui.print("[default: ", defaultValue, "] "); err != nil {
			return "", err
		}
	}
	return ui.readLine(defaultValue)
}

func (ui *UI) Many(_ context.Context, question string, defaultValue string) ([]string, error) {
	if err := ui.print(question, "? (comma-separated) "); err != nil {
		return nil, err
	}

	if defaultValue != "" {
		if err := ui.print("[default: ", defaultValue, "] "); err != nil {
			return nil, err
		}
	}

	line, err := ui.readLine("")
	return toList(line), err
}

func (ui *UI) Select(_ context.Context, question string, defaultValue string, options []string) (string, error) {
	if err := ui.print(question, "\n"); err != nil {
		return "", err
	}

	for i, opt := range options {
		if err := ui.print(i+1, " - ", opt, "\n"); err != nil {
			return "", err
		}
	}

	if err := ui.print("Pick the option "); err != nil {
		return "", err
	}

	if defaultValue != "" {
		if err := ui.print("[default: ", defaultValue, "] "); err != nil {
			return "", err
		}
	}

	if err := ui.print(": "); err != nil {
		return "", err
	}

	opts, err := ui.readOptions(options, defaultValue)
	if err != nil {
		return "", err
	}
	if len(opts) == 0 {
		return "", fmt.Errorf("no option selected")
	}
	return opts[0], nil
}

func (ui *UI) Choose(_ context.Context, question string, defaultValue string, options []string) ([]string, error) {
	if err := ui.print(question, "\n"); err != nil {
		return nil, err
	}

	for i, opt := range options {
		if err := ui.print(i+1, " - ", opt, "\n"); err != nil {
			return nil, err
		}
	}

	if err := ui.print("Choose options (comma-separated) "); err != nil {
		return nil, err
	}

	if defaultValue != "" {
		if err := ui.print("[default: ", defaultValue, "] "); err != nil {
			return nil, err
		}
	}

	if err := ui.print(": "); err != nil {
		return nil, err
	}

	return ui.readOptions(options, defaultValue)
}

func (ui *UI) Error(ctx context.Context, message string) error {
	return ui.print("[error] ", message, "\n")
}

func (ui *UI) Title(ctx context.Context, message string) error {
	return ui.print("\n\n", message, "\n\n")
}

func (ui *UI) print(data ...interface{}) error {
	_, err := fmt.Fprint(ui.out, data...)
	return err
}

func (ui *UI) readLine(defaultValue string) (string, error) {
	line, _, err := ui.in.ReadLine()
	if err != nil {
		return "", err
	}

	v := strings.TrimSpace(string(line))
	if v == "" {
		v = defaultValue
	}

	return v, nil
}

func (ui *UI) readOptions(options []string, defaultLine string) ([]string, error) {
	line, err := ui.readLine(defaultLine)
	if err != nil {
		return nil, err
	}

	var result []string
	var alreadyPicked = make(map[int]bool)

	for _, numValue := range toList(line) {
		num, err := strconv.Atoi(numValue)
		if err != nil {
			return nil, err
		}

		if num < 1 || num > len(options) {
			return nil, fmt.Errorf("unknown option %d", num)
		}

		if !alreadyPicked[num] {
			alreadyPicked[num] = true
			result = append(result, options[num-1])
		}
	}

	return result, nil
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
