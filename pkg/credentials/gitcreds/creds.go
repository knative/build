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
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/knative/build/pkg/credentials"
)

const (
	annotationPrefix = "build.dev/git-"
)

var (
	basicConfig basicGitConfig
	sshConfig   sshGitConfig
)

func flags(fs *flag.FlagSet) {
	basicConfig = basicGitConfig{entries: make(map[string]basicEntry)}
	fs.Var(&basicConfig, "basic-git", "List of secret=url pairs.")

	sshConfig = sshGitConfig{entries: make(map[string]sshEntry)}
	fs.Var(&sshConfig, "ssh-git", "List of secret=url pairs.")
}

func init() {
	flags(flag.CommandLine)
}

type GitConfigBuilder struct{}

func NewBuilder() credentials.Builder {
	return &GitConfigBuilder{}
}

// MatchingAnnotations extracts flags for the credential helper
// from the supplied secret and returns a slice (of length 0 or
// greater) of applicable domains.
func (dcb *GitConfigBuilder) MatchingAnnotations(secret *corev1.Secret) []string {
	var flagName string
	var flags []string
	switch secret.Type {
	case corev1.SecretTypeBasicAuth:
		flagName = "basic-git"

	case corev1.SecretTypeSSHAuth:
		flagName = "ssh-git"

	default:
		return flags
	}

	for k, v := range secret.Annotations {
		if strings.HasPrefix(k, annotationPrefix) {
			flags = append(flags, fmt.Sprintf("-%s=%s=%s", flagName, secret.Name, v))
		}
	}
	return flags
}

func (dcb *GitConfigBuilder) Write() error {
	if err := basicConfig.Write(); err != nil {
		return err
	}
	return sshConfig.Write()
}
