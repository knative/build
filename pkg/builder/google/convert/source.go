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

package convert

import (
	"fmt"
	"regexp"

	"google.golang.org/api/cloudbuild/v1"

	v1alpha1 "github.com/elafros/build/pkg/apis/build/v1alpha1"

	"github.com/elafros/build/pkg/builder/validation"
)

var (
	csr = regexp.MustCompile("^https://source.developers.google.com/p/([^/]+)/r/(.*)$")
)

func ToRepoSourceFromGit(og *v1alpha1.GitSourceSpec) (*cloudbuild.RepoSource, error) {
	if !csr.MatchString(og.Url) {
		// TODO(mattmoor): This could fall back on logic as in the on-cluster builder.
		// https://github.com/elafros/build/issues/22
		return nil, validation.NewError("UnsupportedGitUrl", "git.url must match %v for the Google builder, got %q", csr, og.Url)
	}
	// Extract the capture groups.
	match := csr.FindStringSubmatch(og.Url)
	projectId := match[1]
	repoName := match[2]

	switch {
	case og.Commit != "":
		return &cloudbuild.RepoSource{
			ProjectId: projectId,
			RepoName:  repoName,
			CommitSha: og.Commit,
		}, nil
	case og.Tag != "":
		return &cloudbuild.RepoSource{
			ProjectId: projectId,
			RepoName:  repoName,
			TagName:   og.Tag,
		}, nil
	case og.Branch != "":
		return &cloudbuild.RepoSource{
			ProjectId:  projectId,
			RepoName:   repoName,
			BranchName: og.Branch,
		}, nil
	case og.Ref != "":
		return nil, validation.NewError("UnsupportedRef", "git.ref is unsupported by the Googler builder, got: %v", og.Ref)
	default:
		return nil, validation.NewError("MissingCommitish", "missing one of branch/tag/ref/commit, got: %v", og)
	}

}

func ToGitFromRepoSource(og *cloudbuild.RepoSource) (*v1alpha1.GitSourceSpec, error) {
	if og.Dir != "" {
		return nil, validation.NewError("UnsupportedDir", "the Build CRD doesn't support 'dir', got: %v", og.Dir)
	}
	return &v1alpha1.GitSourceSpec{
		Url:    fmt.Sprintf("https://source.developers.google.com/p/%s/r/%s", og.ProjectId, og.RepoName),
		Branch: og.BranchName,
		Tag:    og.TagName,
		Commit: og.CommitSha,
	}, nil
}
