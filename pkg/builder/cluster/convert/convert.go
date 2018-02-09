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
	"flag"
	"fmt"
	"reflect"
	"regexp"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	v1alpha1 "github.com/google/build-crd/pkg/apis/cloudbuild/v1alpha1"
	"github.com/google/build-crd/pkg/builder"
)

// These are effectively const, but Go doesn't have such an annotation.
var (
	emptyVolumeSource = corev1.VolumeSource{
		EmptyDir: &corev1.EmptyDirVolumeSource{},
	}
	// These are injected into all of the source/step containers.
	implicitEnvVars = []corev1.EnvVar{{
		Name:  "HOME",
		Value: "/builder/home",
	}}
	implicitVolumeMounts = []corev1.VolumeMount{{
		Name:      "workspace",
		MountPath: "/workspace",
	}, {
		Name:      "home",
		MountPath: "/builder/home",
	}}
	implicitVolumes = []corev1.Volume{{
		Name:         "workspace",
		VolumeSource: emptyVolumeSource,
	}, {
		Name:         "home",
		VolumeSource: emptyVolumeSource,
	}}
	// A benign placeholder for when a container is required, but none was specified.
	nopContainer = corev1.Container{
		Name:    "nop",
		Image:   "busybox",
		Command: []string{"/bin/echo"},
		Args:    []string{"Nothing to push"},
	}
)

func validateVolumes(vs []corev1.Volume) error {
	seen := make(map[string]interface{})
	for i, v := range vs {
		if _, ok := seen[v.Name]; ok {
			return &builder.ValidationError{
				Reason:  "DuplicateVolume",
				Message: fmt.Sprintf("saw Volume %q defined multiple times", v.Name),
			}
		}
		seen[v.Name] = i
	}
	return nil
}

const (
	// Names for source containers
	gitSource    = "git-source"
	customSource = "custom-source"
)

var (
	// The container with Git that we use to implement the Git source step.
	gitImage = flag.String("git-image", "override-with-git:latest",
		"The container image container out Git binary.")
)

var (
	// Used to reverse the mapping from source to containers based on the
	// name given to the first step.  This is fragile, but predominantly for testing.
	containerToSourceMap = map[string]func(corev1.Container) (*v1alpha1.SourceSpec, error){
		gitSource:    containerToGit,
		customSource: containerToCustom,
	}
	// Used to recover the type of reference we checked out.
	reCommits  = regexp.MustCompile("^[0-9a-f]{40}$")
	reRefs     = regexp.MustCompile("^refs/")
	reTags     = regexp.MustCompile("^refs/tags/(.*)")
	reBranches = regexp.MustCompile("^refs/heads/(.*)")
)

// TODO(mattmoor): Should we move this somewhere common, because of the flag?
func gitToContainer(git *v1alpha1.GitSourceSpec) (*corev1.Container, error) {
	if git.Url == "" {
		return nil, &builder.ValidationError{
			Reason:  "MissingUrl",
			Message: fmt.Sprintf("git sources are expected to specify a Url, got: %v", git),
		}
	}
	var commitish string
	switch {
	case git.Tag != "":
		commitish = fmt.Sprintf("refs/tags/%s", git.Tag)
	case git.Branch != "":
		commitish = fmt.Sprintf("refs/heads/%s", git.Branch)
	case git.Commit != "":
		commitish = git.Commit
	case git.Ref != "":
		commitish = git.Ref
	default:
		return nil, &builder.ValidationError{
			Reason:  "MissingCommitish",
			Message: fmt.Sprintf("git sources are expected to specify one of commit/tag/branch/ref, got %v", git),
		}
	}
	return &corev1.Container{
		Name:  gitSource,
		Image: *gitImage,
		Args: []string{
			"-url", git.Url,
			"-commitish", commitish,
		},
	}, nil
}

func containerToGit(git corev1.Container) (*v1alpha1.SourceSpec, error) {
	if git.Image != *gitImage {
		return nil, fmt.Errorf("Unrecognized git source image: %v", git.Image)
	}
	if len(git.Args) < 3 {
		return nil, fmt.Errorf("Unexpectedly few arguments to git source container: %v", git.Args)
	}
	// Now undo what we did above
	url := git.Args[1]
	commitish := git.Args[3]
	switch {
	case reCommits.MatchString(commitish):
		return &v1alpha1.SourceSpec{
			Git: &v1alpha1.GitSourceSpec{
				Url:    url,
				Commit: commitish,
			},
		}, nil

	case reTags.MatchString(commitish):
		match := reTags.FindStringSubmatch(commitish)
		return &v1alpha1.SourceSpec{
			Git: &v1alpha1.GitSourceSpec{
				Url: url,
				Tag: match[1],
			},
		}, nil

	case reBranches.MatchString(commitish):
		match := reBranches.FindStringSubmatch(commitish)
		return &v1alpha1.SourceSpec{
			Git: &v1alpha1.GitSourceSpec{
				Url:    url,
				Branch: match[1],
			},
		}, nil

	case reRefs.MatchString(commitish):
		return &v1alpha1.SourceSpec{
			Git: &v1alpha1.GitSourceSpec{
				Url: url,
				Ref: commitish,
			},
		}, nil

	default:
		return nil, fmt.Errorf("Unable to determine type of commitish: %v", commitish)
	}
}

func customToContainer(source *corev1.Container) (*corev1.Container, error) {
	if source.Name != "" {
		return nil, &builder.ValidationError{
			Reason:  "OmitName",
			Message: fmt.Sprintf("custom source containers are expected to omit Name, got: %v", source.Name),
		}
	}
	custom := source.DeepCopy()
	custom.Name = customSource
	return custom, nil
}

func containerToCustom(custom corev1.Container) (*v1alpha1.SourceSpec, error) {
	c := custom.DeepCopy()
	c.Name = ""
	return &v1alpha1.SourceSpec{Custom: c}, nil
}

func sourceToContainer(source *v1alpha1.SourceSpec) (*corev1.Container, error) {
	switch {
	case source == nil:
		return nil, nil
	case source.Git != nil:
		return gitToContainer(source.Git)
	case source.Custom != nil:
		return customToContainer(source.Custom)
	default:
		return nil, &builder.ValidationError{
			Reason:  "UnrecognizedSource",
			Message: fmt.Sprintf("saw SourceSpec with no supported contents: %v", source),
		}
	}
}

func FromCRD(build *v1alpha1.Build) (*batchv1.Job, error) {
	build = build.DeepCopy()

	var sourceContainers []corev1.Container
	if build.Spec.Source != nil {
		scm, err := sourceToContainer(build.Spec.Source)
		if err != nil {
			return nil, err
		}
		sourceContainers = append(sourceContainers, *scm)
	}

	// Add the implicit volume mounts to each step container.
	var initContainers []corev1.Container
	for i, step := range append(sourceContainers, build.Spec.Steps...) {
		if step.WorkingDir == "" {
			step.WorkingDir = "/workspace"
		}
		if step.Name == "" {
			step.Name = fmt.Sprintf("unnamed-step-%d", i)
		}
		step.Env = append(implicitEnvVars, step.Env...)
		// TODO(mattmoor): Check that volumeMounts match volumes.
		step.VolumeMounts = append(step.VolumeMounts, implicitVolumeMounts...)
		initContainers = append(initContainers, step)
	}
	// Add workspace to the explicitly declared user volumes.
	volumes := append(build.Spec.Volumes, implicitVolumes...)
	if err := validateVolumes(volumes); err != nil {
		return nil, err
	}

	zero := int32(0)
	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			// We execute the build's job in the same namespace as where the build was
			// created so that it can access colocated resources.
			Namespace: build.Namespace,
			// Ensure our Job gets a unique name.
			GenerateName: fmt.Sprintf("%s-", build.Name),
			// If our parent Build is deleted, then we should be as well.
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(build, schema.GroupVersionKind{
					Group:   v1alpha1.SchemeGroupVersion.Group,
					Version: v1alpha1.SchemeGroupVersion.Version,
					Kind:    "Build",
				}),
			},
		},
		Spec: batchv1.JobSpec{
			// Don't retry any failed builds.
			BackoffLimit: &zero,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					// If the build fails, don't restart it.
					RestartPolicy:  corev1.RestartPolicyNever,
					InitContainers: initContainers,
					Containers:     []corev1.Container{nopContainer},
					Volumes:        volumes,
					// TODO(mattmoor): We may need support for imagePullSecrets for pulling private images.
				},
			},
		},
	}, nil
}

func isImplicitEnvVar(ev corev1.EnvVar) bool {
	for _, iev := range implicitEnvVars {
		if ev.Name == iev.Name {
			return true
		}
	}
	return false
}

func filterImplicitEnvVars(evs []corev1.EnvVar) []corev1.EnvVar {
	var envs []corev1.EnvVar
	for _, ev := range evs {
		if isImplicitEnvVar(ev) {
			continue
		}
		envs = append(envs, ev)
	}
	return envs
}

func isImplicitVolumeMount(vm corev1.VolumeMount) bool {
	for _, ivm := range implicitVolumeMounts {
		if vm.Name == ivm.Name {
			return true
		}
	}
	return false
}

func filterImplicitVolumeMounts(vms []corev1.VolumeMount) []corev1.VolumeMount {
	var volumes []corev1.VolumeMount
	for _, vm := range vms {
		if isImplicitVolumeMount(vm) {
			continue
		}
		volumes = append(volumes, vm)
	}
	return volumes
}

func isImplicitVolume(v corev1.Volume) bool {
	for _, iv := range implicitVolumes {
		if v.Name == iv.Name {
			return true
		}
	}
	return false
}

func filterImplicitVolumes(vs []corev1.Volume) []corev1.Volume {
	var volumes []corev1.Volume
	for _, v := range vs {
		if isImplicitVolume(v) {
			continue
		}
		volumes = append(volumes, v)
	}
	return volumes
}

func ToCRD(job *batchv1.Job) (*v1alpha1.Build, error) {
	job = job.DeepCopy()
	podSpec := job.Spec.Template.Spec

	for _, c := range podSpec.Containers {
		if !reflect.DeepEqual(c, nopContainer) {
			return nil, fmt.Errorf("unrecognized container spec, got: %v", podSpec.Containers)
		}
	}

	var steps []corev1.Container
	for _, step := range podSpec.InitContainers {
		if step.WorkingDir == "/workspace" {
			step.WorkingDir = ""
		}
		step.Env = filterImplicitEnvVars(step.Env)
		step.VolumeMounts = filterImplicitVolumeMounts(step.VolumeMounts)
		steps = append(steps, step)
	}
	volumes := filterImplicitVolumes(podSpec.Volumes)

	var scm *v1alpha1.SourceSpec
	if conv, ok := containerToSourceMap[steps[0].Name]; ok {
		src, err := conv(steps[0])
		if err != nil {
			return nil, err
		}
		// The first init container is actually a source step.  Convert
		// it to our source spec and pop it off the list of steps.
		scm = src
		steps = steps[1:]
	}

	return &v1alpha1.Build{
		// TODO(mattmoor): What should we do for ObjectMeta stuff?
		Spec: v1alpha1.BuildSpec{
			Source:  scm,
			Steps:   steps,
			Volumes: volumes,
		},
	}, nil
}
