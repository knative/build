package v1alpha1

import (
	"errors"
	"regexp"
	"strings"

	"github.com/knative/pkg/apis"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1 "k8s.io/api/core/v1"
)

// Validate build template
func (b *BuildTemplate) Validate() *apis.FieldError {
	return validateObjectMetadata(b.GetObjectMeta()).ViaField("metadata").Also(b.Spec.Validate().ViaField("spec"))
}

var errInvalidBuildTemplate = errors.New("failed to convert to BuildTemplate")

// Validate Build Template
func (b *BuildTemplateSpec) Validate() *apis.FieldError {
	if err := validateSteps(b.Steps); err != nil {
		return err
	}
	if err := validateVolumes(b.Volumes); err != nil {
		return err
	}
	if err := validateParameters(b.Parameters); err != nil {
		return err
	}
	return nil
	// return validatePlaceholders(tmpl.TemplateSpec().Steps)
}

func validateObjectMetadata(meta metav1.Object) *apis.FieldError {
	name := meta.GetName()

	if strings.Contains(name, ".") {
		return &apis.FieldError{
			Message: "Invalid resource name: special character . must not be present",
			Paths:   []string{"name"},
		}
	}

	if len(name) > 63 {
		return &apis.FieldError{
			Message: "Invalid resource name: length must be no more than 63 characters",
			Paths:   []string{"name"},
		}
	}
	return nil
}

var nestedPlaceholderRE = regexp.MustCompile(`\${[^}]+\$`)

func validatePlaceholders(steps []corev1.Container) *apis.FieldError {
	for _, s := range steps {
		if nestedPlaceholderRE.MatchString(s.Name) {
			return apis.ErrInvalidValue("nestedPlaceHolderStepName", s.Name)
		}
		for _, a := range s.Args {
			if nestedPlaceholderRE.MatchString(a) {
				return apis.ErrInvalidValue("nestedPlaceHolderArgs", a)
			}
		}
		for _, e := range s.Env {
			if nestedPlaceholderRE.MatchString(e.Value) {
				return apis.ErrInvalidValue("nestedPlaceHolderEnv", e.Value)
			}
		}
		if nestedPlaceholderRE.MatchString(s.WorkingDir) {
			return apis.ErrInvalidValue("nestedPlaceHolderEnv", s.WorkingDir)
		}
		for _, c := range s.Command {
			if nestedPlaceholderRE.MatchString(c) {
				return apis.ErrInvalidValue("nestedPlaceHolderCmd", c)
			}
		}
	}
	return nil
}

func validateParameters(params []ParameterSpec) *apis.FieldError {
	// Template must not duplicate parameter names.
	seen := map[string]struct{}{}
	for _, p := range params {
		if _, ok := seen[p.Name]; ok {
			return apis.ErrMultipleOneOf("ParamName")
		}
		seen[p.Name] = struct{}{}
	}
	return nil
}

func validateVolumes(volumes []corev1.Volume) *apis.FieldError {
	// Build must not duplicate volume names.
	vols := map[string]struct{}{}
	for _, v := range volumes {
		if _, ok := vols[v.Name]; ok {
			return apis.ErrMultipleOneOf("volumeName")
		}
		vols[v.Name] = struct{}{}
	}
	return nil
}

func validateSteps(steps []corev1.Container) *apis.FieldError {
	// Build must not duplicate step names.
	names := map[string]struct{}{}
	for _, s := range steps {
		if s.Image == "" {
			return apis.ErrMissingField("Image")
		}

		if s.Name == "" {
			continue
		}
		if _, ok := names[s.Name]; ok {
			return apis.ErrMultipleOneOf("stepName")
		}
		names[s.Name] = struct{}{}
	}
	return nil
}
