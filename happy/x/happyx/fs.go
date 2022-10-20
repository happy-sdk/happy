// Copyright 2022 The Happy Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package happyx

import (
	"embed"
	"github.com/mkungla/happy"
	"io/fs"
	"os"
	"path"
	"path/filepath"
)

type localFS string

func LocalFS(dir string) happy.FS {
	return localFS(dir)
}

func (lfs localFS) Open(name string) (fs.File, error) {
	return os.Open(filepath.Join(string(lfs), name))
}

func (lfs localFS) ReadDir(name string) ([]fs.DirEntry, error) {
	return os.ReadDir(filepath.Join(string(lfs), name))
}

func (lfs localFS) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(filepath.Join(string(lfs), name))
}

type embedFS struct {
	fs  embed.FS
	dir string
}

func EmbedSubFS(efsys embed.FS, dir string) happy.FS {
	return &embedFS{
		fs:  efsys,
		dir: dir,
	}
}

func (efs *embedFS) Open(name string) (fs.File, error) {
	return efs.fs.Open(path.Join(efs.dir, name))
}

func (efs *embedFS) ReadDir(name string) ([]fs.DirEntry, error) {
	return efs.fs.ReadDir(path.Join(efs.dir, name))
}

func (efs *embedFS) ReadFile(name string) ([]byte, error) {
	return efs.fs.ReadFile(path.Join(efs.dir, name))
}
