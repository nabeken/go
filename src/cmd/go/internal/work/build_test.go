// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package work

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"cmd/go/internal/base"
	"cmd/go/internal/cfg"
	"cmd/go/internal/load"
)

func TestRemoveDevNull(t *testing.T) {
	fi, err := os.Lstat(os.DevNull)
	if err != nil {
		t.Skip(err)
	}
	if fi.Mode().IsRegular() {
		t.Errorf("Lstat(%s).Mode().IsRegular() = true; expected false", os.DevNull)
	}
	mayberemovefile(os.DevNull)
	_, err = os.Lstat(os.DevNull)
	if err != nil {
		t.Errorf("mayberemovefile(%s) did remove it; oops", os.DevNull)
	}
}

func TestSplitPkgConfigOutput(t *testing.T) {
	for _, test := range []struct {
		in   []byte
		want []string
	}{
		{[]byte(`-r:foo -L/usr/white\ space/lib -lfoo\ bar -lbar\ baz`), []string{"-r:foo", "-L/usr/white space/lib", "-lfoo bar", "-lbar baz"}},
		{[]byte(`-lextra\ fun\ arg\\`), []string{`-lextra fun arg\`}},
		{[]byte(`broken flag\`), []string{"broken", "flag"}},
		{[]byte("\textra     whitespace\r\n"), []string{"extra", "whitespace"}},
		{[]byte("     \r\n      "), nil},
	} {
		got := splitPkgConfigOutput(test.in)
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("splitPkgConfigOutput(%v) = %v; want %v", test.in, got, test.want)
		}
	}
}

func TestSharedLibName(t *testing.T) {
	// TODO(avdva) - make these values platform-specific
	prefix := "lib"
	suffix := ".so"
	testData := []struct {
		args      []string
		pkgs      []*load.Package
		expected  string
		expectErr bool
		rootedAt  string
	}{
		{
			args:     []string{"std"},
			pkgs:     []*load.Package{},
			expected: "std",
		},
		{
			args:     []string{"std", "cmd"},
			pkgs:     []*load.Package{},
			expected: "std,cmd",
		},
		{
			args:     []string{},
			pkgs:     []*load.Package{pkgImportPath("gopkg.in/somelib")},
			expected: "gopkg.in-somelib",
		},
		{
			args:     []string{"./..."},
			pkgs:     []*load.Package{pkgImportPath("somelib")},
			expected: "somelib",
			rootedAt: "somelib",
		},
		{
			args:     []string{"../somelib", "../somelib"},
			pkgs:     []*load.Package{pkgImportPath("somelib")},
			expected: "somelib",
		},
		{
			args:     []string{"../lib1", "../lib2"},
			pkgs:     []*load.Package{pkgImportPath("gopkg.in/lib1"), pkgImportPath("gopkg.in/lib2")},
			expected: "gopkg.in-lib1,gopkg.in-lib2",
		},
		{
			args: []string{"./..."},
			pkgs: []*load.Package{
				pkgImportPath("gopkg.in/dir/lib1"),
				pkgImportPath("gopkg.in/lib2"),
				pkgImportPath("gopkg.in/lib3"),
			},
			expected: "gopkg.in",
			rootedAt: "gopkg.in",
		},
		{
			args:      []string{"std", "../lib2"},
			pkgs:      []*load.Package{},
			expectErr: true,
		},
		{
			args:      []string{"all", "./"},
			pkgs:      []*load.Package{},
			expectErr: true,
		},
		{
			args:      []string{"cmd", "fmt"},
			pkgs:      []*load.Package{},
			expectErr: true,
		},
	}
	for _, data := range testData {
		func() {
			if data.rootedAt != "" {
				tmpGopath, err := ioutil.TempDir("", "gopath")
				if err != nil {
					t.Fatal(err)
				}
				oldGopath := cfg.BuildContext.GOPATH
				defer func() {
					cfg.BuildContext.GOPATH = oldGopath
					os.Chdir(base.Cwd)
					err := os.RemoveAll(tmpGopath)
					if err != nil {
						t.Error(err)
					}
				}()
				root := filepath.Join(tmpGopath, "src", data.rootedAt)
				err = os.MkdirAll(root, 0755)
				if err != nil {
					t.Fatal(err)
				}
				cfg.BuildContext.GOPATH = tmpGopath
				os.Chdir(root)
			}
			computed, err := libname(data.args, data.pkgs)
			if err != nil {
				if !data.expectErr {
					t.Errorf("libname returned an error %q, expected a name", err.Error())
				}
			} else if data.expectErr {
				t.Errorf("libname returned %q, expected an error", computed)
			} else {
				expected := prefix + data.expected + suffix
				if expected != computed {
					t.Errorf("libname returned %q, expected %q", computed, expected)
				}
			}
		}()
	}
}

func pkgImportPath(pkgpath string) *load.Package {
	return &load.Package{
		PackagePublic: load.PackagePublic{
			ImportPath: pkgpath,
		},
	}
}
