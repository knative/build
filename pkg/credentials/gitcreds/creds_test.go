/*
Copyright 2018 Google, Inc. All rights reserved.
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

package gitcreds

import (
	"flag"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/build-crd/pkg/credentials"
)

func TestFlagHandling(t *testing.T) {
	credentials.VolumePath = os.Getenv("TEST_TMPDIR")
	dir := credentials.VolumeName("foo")
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		t.Fatalf("os.MkdirAll(%s) = %v", dir, err)
	}
	if err := ioutil.WriteFile(filepath.Join(dir, "username"), []byte("bar"), 0777); err != nil {
		t.Fatalf("ioutil.WriteFile(username) = %v", err)
	}
	if err := ioutil.WriteFile(filepath.Join(dir, "password"), []byte("baz"), 0777); err != nil {
		t.Fatalf("ioutil.WriteFile(username) = %v", err)
	}

	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	flags(fs)
	err := fs.Parse([]string{
		"-basic-git=foo=https://github.com",
	})
	if err != nil {
		t.Fatalf("flag.CommandLine.Parse() = %v", err)
	}

	os.Setenv("HOME", credentials.VolumePath)
	if err := NewBuilder().Write(); err != nil {
		t.Fatalf("Write() = %v", err)
	}

	b, err := ioutil.ReadFile(filepath.Join(credentials.VolumePath, ".gitconfig"))
	if err != nil {
		t.Fatalf("ioutil.ReadFile(.gitconfig) = %v", err)
	}

	expectedGitConfig := `[credential]
	helper = store
[credential "https://github.com"]
	username = bar
`
	if string(b) != expectedGitConfig {
		t.Errorf("got: %v, wanted: %v", string(b), expectedGitConfig)
	}

	b, err = ioutil.ReadFile(filepath.Join(credentials.VolumePath, ".git-credentials"))
	if err != nil {
		t.Fatalf("ioutil.ReadFile(.git-credentials) = %v", err)
	}

	expectedGitCredentials := `https://bar:baz@github.com
`
	if string(b) != expectedGitCredentials {
		t.Errorf("got: %v, wanted: %v", string(b), expectedGitCredentials)
	}
}

func TestFlagHandlingTwice(t *testing.T) {
	credentials.VolumePath = os.Getenv("TEST_TMPDIR")
	fooDir := credentials.VolumeName("foo")
	if err := os.MkdirAll(fooDir, os.ModePerm); err != nil {
		t.Fatalf("os.MkdirAll(%s) = %v", fooDir, err)
	}
	if err := ioutil.WriteFile(filepath.Join(fooDir, "username"), []byte("asdf"), 0777); err != nil {
		t.Fatalf("ioutil.WriteFile(username) = %v", err)
	}
	if err := ioutil.WriteFile(filepath.Join(fooDir, "password"), []byte("blah"), 0777); err != nil {
		t.Fatalf("ioutil.WriteFile(username) = %v", err)
	}
	barDir := credentials.VolumeName("bar")
	if err := os.MkdirAll(barDir, os.ModePerm); err != nil {
		t.Fatalf("os.MkdirAll(%s) = %v", barDir, err)
	}
	if err := ioutil.WriteFile(filepath.Join(barDir, "username"), []byte("bleh"), 0777); err != nil {
		t.Fatalf("ioutil.WriteFile(username) = %v", err)
	}
	if err := ioutil.WriteFile(filepath.Join(barDir, "password"), []byte("belch"), 0777); err != nil {
		t.Fatalf("ioutil.WriteFile(username) = %v", err)
	}

	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	flags(fs)
	err := fs.Parse([]string{
		"-basic-git=foo=https://github.com",
		"-basic-git=bar=https://gitlab.com",
	})
	if err != nil {
		t.Fatalf("flag.CommandLine.Parse() = %v", err)
	}

	os.Setenv("HOME", credentials.VolumePath)
	if err := NewBuilder().Write(); err != nil {
		t.Fatalf("Write() = %v", err)
	}

	b, err := ioutil.ReadFile(filepath.Join(credentials.VolumePath, ".gitconfig"))
	if err != nil {
		t.Fatalf("ioutil.ReadFile(.gitconfig) = %v", err)
	}

	expectedGitConfig := `[credential]
	helper = store
[credential "https://github.com"]
	username = asdf
[credential "https://gitlab.com"]
	username = bleh
`
	if string(b) != expectedGitConfig {
		t.Errorf("got: %v, wanted: %v", string(b), expectedGitConfig)
	}

	b, err = ioutil.ReadFile(filepath.Join(credentials.VolumePath, ".git-credentials"))
	if err != nil {
		t.Fatalf("ioutil.ReadFile(.git-credentials) = %v", err)
	}

	expectedGitCredentials := `https://asdf:blah@github.com
https://bleh:belch@gitlab.com
`
	if string(b) != expectedGitCredentials {
		t.Errorf("got: %v, wanted: %v", string(b), expectedGitCredentials)
	}
}

func TestFlagHandlingMissingFiles(t *testing.T) {
	credentials.VolumePath = os.Getenv("TEST_TMPDIR")
	dir := credentials.VolumeName("not-found")
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		t.Fatalf("os.MkdirAll(%s) = %v", dir, err)
	}
	// No username / password files yields an error.

	cfg := basicGitConfig{make(map[string]basicEntry)}
	if err := cfg.Set("not-found=https://github.com"); err == nil {
		t.Error("Set(); got success, wanted error.")
	}
}

func TestFlagHandlingURLCollision(t *testing.T) {
	credentials.VolumePath = os.Getenv("TEST_TMPDIR")
	dir := credentials.VolumeName("foo")
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		t.Fatalf("os.MkdirAll(%s) = %v", dir, err)
	}
	if err := ioutil.WriteFile(filepath.Join(dir, "username"), []byte("bar"), 0777); err != nil {
		t.Fatalf("ioutil.WriteFile(username) = %v", err)
	}
	if err := ioutil.WriteFile(filepath.Join(dir, "password"), []byte("baz"), 0777); err != nil {
		t.Fatalf("ioutil.WriteFile(username) = %v", err)
	}

	cfg := basicGitConfig{make(map[string]basicEntry)}
	if err := cfg.Set("foo=https://github.com"); err != nil {
		t.Fatalf("First Set() = %v", err)
	}
	if err := cfg.Set("bar=https://github.com"); err == nil {
		t.Error("Second Set(); got success, wanted error.")
	}
}

func TestMalformedValueTooMany(t *testing.T) {
	cfg := basicGitConfig{make(map[string]basicEntry)}
	if err := cfg.Set("bar=baz=blah"); err == nil {
		t.Error("Second Set(); got success, wanted error.")
	}
}

func TestMalformedValueTooFew(t *testing.T) {
	cfg := basicGitConfig{make(map[string]basicEntry)}
	if err := cfg.Set("bar"); err == nil {
		t.Error("Second Set(); got success, wanted error.")
	}
}
