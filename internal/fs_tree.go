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
	"os"
	"path/filepath"
	"strings"
)

// FSTree is general tree which reflects hierarchy of modified files.
// It's used to track modified files even after rendering and copying.
type FSTree struct {
	Name     string    // path or base name
	Dir      bool      // is it directory (used to skip rendering content)
	Children []*FSTree // child nodes (files or dirs)
	parent   *FSTree   // ref to parent (could be null for root)
}

// Paths returns list of all paths from this node and children.
// This is very slow operation.
func (fs *FSTree) Paths() []string {
	var ans = []string{fs.Path()}
	for _, c := range fs.Children {
		ans = append(ans, c.Paths()...)
	}
	return ans
}

// Render node name.
// Rules for returned string:
// 1. same name -> do nothing
// 2. empty name -> remove node and children
// 3. rename
func (fs *FSTree) Render(renderName func(node *FSTree) (string, error)) error {
	newName, err := renderName(fs)
	if err != nil {
		return err
	}
	if newName == fs.Name {
		// nothing changed, continue walk
		cp := fs.Children
		for _, child := range cp {
			if err := child.Render(renderName); err != nil {
				return err
			}
		}
		return nil
	}

	if newName == "" {
		//remove
		if err := os.RemoveAll(fs.Path()); err != nil {
			return err
		}
		var filtered []*FSTree
		for _, child := range fs.parent.Children {
			if child != fs {
				filtered = append(filtered, child)
			}
		}
		fs.parent.Children = filtered
		return nil
	}

	// name changed
	// we are walking top-down, so it's enough just to update corresponding node
	oldPath := fs.Path()
	fs.Name = newName
	newPath := fs.Path()
	if err := os.Rename(oldPath, newPath); err != nil {
		return err
	}
	// continue walk
	cp := fs.Children
	for _, child := range cp {
		if err := child.Render(renderName); err != nil {
			return err
		}
	}
	return nil
}

// Add node to tree. It's naive implementation and requires O(N*M) complexity, where N is number of sections in path,
// and M is number of elements per section.
func (fs *FSTree) Add(path string, dir bool) *FSTree {
	parts := strings.Split(path, string(filepath.Separator))
	current := fs
	for i, p := range parts {
		isDir := i != len(parts)-1 || dir
		current = current.child(p, isDir)
	}
	return current
}

// Path from the root node.
func (fs *FSTree) Path() string {
	if fs.parent == nil {
		return fs.Name
	}
	return filepath.Join(fs.parent.Path(), fs.Name)
}

func (fs *FSTree) child(name string, dir bool) *FSTree {
	for _, c := range fs.Children {
		if c.Name == name {
			return c
		}
	}
	child := &FSTree{
		Name:   name,
		Dir:    dir,
		parent: fs,
	}
	fs.Children = append(fs.Children, child)
	return child
}
