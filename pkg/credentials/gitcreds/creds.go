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

	"github.com/google/build-crd/pkg/credentials"
)

const (
	annotationPrefix = "cloudbuild.googleapis.com/git-"
)

var (
	basicConfig basicGitConfig
)

func flags(fs *flag.FlagSet) {
	basicConfig = basicGitConfig{make(map[string]basicEntry)}
	fs.Var(&basicConfig, "basic-git", "List of secret=url pairs.")
}

func init() {
	flags(flag.CommandLine)
}

type GitConfigBuilder struct{}

func NewBuilder() credentials.Builder {
	return &GitConfigBuilder{}
}

func (dcb *GitConfigBuilder) HasMatchingAnnotation(secret *corev1.Secret) (string, bool) {
	var flagName string
	switch secret.Type {
	case corev1.SecretTypeBasicAuth:
		flagName = "basic-git"

	// TODO(mattmoor): Support SSH
	default:
		return "", false
	}

	for k, v := range secret.Annotations {
		if strings.HasPrefix(k, annotationPrefix) {
			return fmt.Sprintf("-%s=%s=%s", flagName, secret.Name, v), true
		}
	}
	return "", false
}

func (dcb *GitConfigBuilder) Write() error {
	return basicConfig.Write()
}
