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

package gitclient

import (
	"context"
	"os"
	"os/exec"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/go-git/go-git/v5"
)

// Client for GIT.
type Client func(ctx context.Context, repo string, directory string) error

// Embedded go-native git client which clones from git repository from default branch with minimal depth (1).
// Reports progress to STDERR. Supports submodules.
func Embedded(ctx context.Context, repo string, directory string) error {
	_, err := git.PlainCloneContext(ctx, directory, false, &git.CloneOptions{
		URL:               repo,
		Depth:             1,
		RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
		Progress:          os.Stderr,
	})
	return err
}

// Native git client (uses git binary) which clones from git repository from default branch with minimal depth (1).
// Send both outputs to STDERR. Supports submodules.
func Native(ctx context.Context, repo string, directory string) error {
	cmd := exec.CommandContext(ctx, "git", "clone", "--depth", "1", "--recurse-submodules", repo, directory)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Auto select git client. In case Git binary exists and version is at least 2.13+ than use native, otherwise - embedded.
func Auto(ctx context.Context) Client {
	minVersion := semver.MustParse("2.13")
	version, err := getGitVersion(ctx)
	if err != nil || version.LessThan(minVersion) {
		return Embedded
	}
	return Native
}

func getGitVersion(ctx context.Context) (*semver.Version, error) {
	cmd := exec.CommandContext(ctx, "git", "--version")
	cmd.Stderr = os.Stderr
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	// git version x.y.z
	parts := strings.Split(strings.TrimSpace(string(output)), " ")
	return semver.NewVersion(parts[len(parts)-1])
}
