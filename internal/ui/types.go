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

package ui

import (
	"context"
)

type Dialog interface {
	// One result for one question
	One(ctx context.Context, question string, defaultValue string) (string, error)
	// Many results for one question
	Many(ctx context.Context, question string, defaultValue string) ([]string, error)
	// Select one option from list
	Select(ctx context.Context, question string, defaultValue string, options []string) (string, error)
	// Choose several options from list
	Choose(ctx context.Context, question string, defaultValue string, options []string) ([]string, error)
}

type UI interface {
	Dialog
	// Error shows error message
	Error(ctx context.Context, message string) error
	// Title shows UI title
	Title(ctx context.Context, message string) error
	// Info shows information message
	Info(ctx context.Context, message string) error
}
