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
package builder

import (
	"regexp"

	corev1 "k8s.io/api/core/v1"

	"github.com/elafros/build/pkg/apis/build/v1alpha1"
	"github.com/elafros/build/pkg/builder/validation"
)

var nestedPlaceholderRE = regexp.MustCompile(`\${[^}]+\$`)

func validateSteps(steps []corev1.Container) error {
	// Build must not duplicate step names.
	names := map[string]struct{}{}
	for _, s := range steps {
		if s.Name == "" {
			continue
		}
		if _, ok := names[s.Name]; ok {
			return validation.NewError("DuplicateStepName", "duplicate step name %q", s.Name)
		}
		names[s.Name] = struct{}{}
	}
	return nil
}

func validateVolumes(volumes []corev1.Volume) error {
	// Build must not duplicate volume names.
	vols := map[string]struct{}{}
	for _, v := range volumes {
		if _, ok := vols[v.Name]; ok {
			return validation.NewError("DuplicateVolumeName", "duplicate volume name %q", v.Name)
		}
		vols[v.Name] = struct{}{}
	}
	return nil
}

// ValidateBuild returns a ValidationError if the build and optional template do not
// specify a valid build.
func ValidateBuild(u *v1alpha1.Build, tmpl *v1alpha1.BuildTemplate) error {
	if u.Spec.Template != nil && len(u.Spec.Steps) > 0 {
		return validation.NewError("TemplateAndSteps", "build cannot specify both template and steps")
	}

	if u.Spec.Template != nil {
		if u.Spec.Template.Name == "" {
			return validation.NewError("MissingTemplateName", "template instantiation is missing template name: %v", u.Spec.Template)
		}
	}

	// If a build specifies a template, all the template's parameters without
	// defaults must be satisfied by the build's parameters.
	var volumes []corev1.Volume
	if tmpl != nil {
		if !IsValidTemplate(&tmpl.Status) {
			return validation.NewError("InvalidTemplate", "The referenced template is invalid.")
		}
		if err := validateArguments(u.Spec.Template.Arguments, tmpl); err != nil {
			return err
		}
		volumes = tmpl.Spec.Volumes
	}
	if err := validateSteps(u.Spec.Steps); err != nil {
		return err
	}
	if err := validateVolumes(append(u.Spec.Volumes, volumes...)); err != nil {
		return err
	}

	return nil
}

func validateArguments(args []v1alpha1.ArgumentSpec, tmpl *v1alpha1.BuildTemplate) error {
	// Build must not duplicate argument names.
	seen := map[string]struct{}{}
	for _, a := range args {
		if _, ok := seen[a.Name]; ok {
			return validation.NewError("DuplicateArgName", "duplicate argument name %q", a.Name)
		}
		seen[a.Name] = struct{}{}
	}
	// If a build specifies a template, all the template's parameters without
	// defaults must be satisfied by the build's parameters.
	if tmpl != nil {
		tmplParams := map[string]string{} // value is the param description.
		for _, p := range tmpl.Spec.Parameters {
			if p.Default == "" {
				tmplParams[p.Name] = p.Description
			}
		}
		for _, p := range args {
			delete(tmplParams, p.Name)
		}
		if len(tmplParams) > 0 {
			type pair struct{ name, desc string }
			var unused []pair
			for k, v := range tmplParams {
				unused = append(unused, pair{k, v})
			}
			return validation.NewError("UnsatisfiedParameter", "build does not specify these required parameters: %s", unused)
		}
	}
	return nil
}

// ValidateTemplate returns a ValidationError if the build template is invalid.
func ValidateTemplate(tmpl *v1alpha1.BuildTemplate) error {
	if err := validateSteps(tmpl.Spec.Steps); err != nil {
		return err
	}
	if err := validateVolumes(tmpl.Spec.Volumes); err != nil {
		return err
	}
	if err := validateParameters(tmpl.Spec.Parameters); err != nil {
		return err
	}
	if err := validatePlaceholders(tmpl.Spec.Steps); err != nil {
		return err
	}
	return nil
}

func validateParameters(params []v1alpha1.ParameterSpec) error {
	// Template must not duplicate parameter names.
	seen := map[string]struct{}{}
	for _, p := range params {
		if _, ok := seen[p.Name]; ok {
			return validation.NewError("DuplicateParamName", "duplicate template parameter name %q", p.Name)
		}
		seen[p.Name] = struct{}{}
	}
	return nil
}

func validatePlaceholders(steps []corev1.Container) error {
	for si, s := range steps {
		if nestedPlaceholderRE.MatchString(s.Name) {
			return validation.NewError("NestedPlaceholder", "nested placeholder in step name %d: %q", si, s.Name)
		}
		for i, a := range s.Args {
			if nestedPlaceholderRE.MatchString(a) {
				return validation.NewError("NestedPlaceholder", "nested placeholder in step %d arg %d: %q", si, i, a)
			}
		}
		for i, e := range s.Env {
			if nestedPlaceholderRE.MatchString(e.Value) {
				return validation.NewError("NestedPlaceholder", "nested placeholder in step %d env value %d: %q", si, i, e.Value)
			}
		}
		if nestedPlaceholderRE.MatchString(s.WorkingDir) {
			return validation.NewError("NestedPlaceholder", "nested placeholder in step %d working dir %q", si, s.WorkingDir)
		}
		for i, c := range s.Command {
			if nestedPlaceholderRE.MatchString(c) {
				return validation.NewError("NestedPlaceholder", "nested placeholder in step %d command %d: %q", si, i, c)
			}
		}
	}
	return nil
}
