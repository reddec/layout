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

package nice

import (
	"context"
	"fmt"
	"strings"

	"github.com/AlecAivazis/survey/v2"
)

func New() *UI {
	return &UI{}
}

type UI struct {
}

func (ui *UI) One(_ context.Context, question string, defaultValue string) (string, error) {
	var res string
	err := survey.AskOne(&survey.Input{
		Message: question,
		Default: defaultValue,
	}, &res)
	return res, err
}

func (ui *UI) Many(_ context.Context, question string, defaultValue string) ([]string, error) {
	var res string
	err := survey.AskOne(&survey.Input{
		Message: question,
		Help:    "comma-separated list",
		Default: defaultValue,
	}, &res)
	return toList(res), err
}

func (ui *UI) Select(_ context.Context, question string, defaultValue string, options []string) (string, error) {
	var res string
	err := survey.AskOne(&survey.Select{
		Message: question,
		Options: options,
		Default: defaultValue,
	}, &res)
	return res, err
}

func (ui *UI) Choose(_ context.Context, question string, defaultValue string, options []string) ([]string, error) {
	var res = make([]string, 0)
	err := survey.AskOne(&survey.MultiSelect{
		Message: question,
		Options: options,
		Default: defaultValue,
	}, &res)
	return res, err
}

func (ui *UI) Error(_ context.Context, message string) error {
	_, err := fmt.Println("[error] ", message)
	return err
}

func (ui *UI) Title(_ context.Context, message string) error {
	_, err := fmt.Println("\n\n", message)
	_, _ = fmt.Println()
	return err
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
