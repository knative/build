package v1alpha1

import (
	"fmt"
	"time"

	"github.com/knative/pkg/apis"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Validate Build
func (b *Build) Validate() *apis.FieldError {
	return validateObjectMetadata(b.GetObjectMeta()).ViaField("metadata").Also(b.Spec.Validate().ViaField("spec"))

	// TODO: Builder Validation
	// return ac.builder.Validate(b)
	//	return nil
}

// Validate for build spec
func (bs *BuildSpec) Validate() *apis.FieldError {
	if bs.Template == nil && len(bs.Steps) == 0 {
		return apis.ErrMissingField("b.spec.template").Also(apis.ErrMissingField("b.spec.steps"))
		//validationError("NoTemplateOrSteps", "build must specify either template or steps")
	}
	if bs.Template != nil && len(bs.Steps) > 0 {
		return apis.ErrMissingField("b.spec.template").Also(apis.ErrMissingField("b.spec.steps"))
		//return validationError("TemplateAndSteps", "build cannot specify both template and steps")
	}

	if bs.Template != nil && bs.Template.Name == "" {
		apis.ErrMissingField("build.spec.template.name")
		//return validationError("MissingTemplateName", "template instantiation is missing template name: %v", b.Spec.Template)
	}

	if err := bs.ValidateSecrets(); err != nil {
		return err
	}
	// If a build specifies a template, all the template's parameters without
	// defaults must be satisfied by the build's parameters.
	var volumes []corev1.Volume
	//var tmpl BuildTemplateInterface
	if bs.Template != nil {
		tmplName := bs.Template.Name
		if tmplName == "" {
			return apis.ErrMissingField("build.spec.template.name")
			//validationError("MissingTemplateName", "the build specifies a template without a name")
		}
		if bs.Template.Kind != "" {
			return bs.Template.Validate()
		}
	}
	if err := validateVolumes(append(bs.Volumes, volumes...)); err != nil {
		return err
	}

	if err := validateSteps(bs.Steps); err != nil {
		return err
	}

	// Need kubeclient to fetch template
	// if err := validateArguments(bs.Template.Arguments, bs.Template.); err != nil {
	// 	return err
	// }

	if err := validateTimeout(bs.Timeout); err != nil {
		return err
	}
	return nil
}

// Validate templateKind
func (b *TemplateInstantiationSpec) Validate() *apis.FieldError {
	switch b.Kind {
	case ClusterBuildTemplateKind,
		BuildTemplateKind:
		return nil
	default:
		return apis.ErrInvalidValue(string(b.Kind), apis.CurrentField)
	}
}

// ValidateSecrets for build
func (bs *BuildSpec) ValidateSecrets() *apis.FieldError {
	// TODO: move this logic into set defaults
	saName := bs.ServiceAccountName
	if saName == "" {
		saName = "default"
	}
	return nil
	// sa, err := ac.client.CoreV1().ServiceAccounts(b.Namespace).Get(saName, metav1.GetOptions{})
	// if err != nil {
	// 	return err
	// }

	// for _, se := range sa.Secrets {
	// 	sec, err := ac.client.CoreV1().Secrets(b.Namespace).Get(se.Name, metav1.GetOptions{})
	// 	if err != nil {
	// 		return err
	// 	}

	// 	// Check that the annotation value "index.docker.io" is not
	// 	// present. This annotation value can be misleading, since
	// 	// Dockerhub expects the fully-specified value
	// 	// "https://index.docker.io/v1/", and other registries accept
	// 	// other variants (e.g., "gcr.io" or "https://gcr.io/v1/",
	// 	// etc.). See https://github.com/knative/build/issues/195
	// 	//
	// 	// TODO(jasonhall): Instead of validating a Secret when a Build
	// 	// uses it, set up webhook validation for Secrets, and reject
	// 	// them outright before a Build ever uses them. This would
	// 	// remove latency at Build-time.
	// 	for k, v := range sec.Annotations {
	// 		if strings.HasPrefix(k, "build.dev/docker-") && v == "index.docker.io" {
	// 			return validationError("BadSecretAnnotation", `Secret %q has incorrect annotation %q / %q, value should be "https://index.docker.io/v1/"`, se.Name, k, v)
	// 		}
	// 	}
	// }
	//	return nil
}

func validateArguments(args []ArgumentSpec, tmpl BuildTemplateInterface) *apis.FieldError {
	// Build must not duplicate argument names.
	seen := map[string]struct{}{}
	for _, a := range args {
		if _, ok := seen[a.Name]; ok {
			return nil
			//alidationError("DuplicateArgName", "duplicate argument name %q", a.Name)
		}
		seen[a.Name] = struct{}{}
	}
	// If a build specifies a template, all the template's parameters without
	// defaults must be satisfied by the build's parameters.
	if tmpl != nil {
		tmplParams := map[string]string{} // value is the param description.
		for _, p := range tmpl.TemplateSpec().Parameters {
			if p.Default == nil {
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
			return apis.ErrMissingField(fmt.Sprintf("%s", unused))
			//validationError("UnsatisfiedParameter", "build does not specify these required parameters: %s", unused)
		}
	}
	return nil
}

type verror struct {
	reason, message string
}

func (ve *verror) Error() string { return fmt.Sprintf("%s: %s", ve.reason, ve.message) }

func validationError(reason, format string, fmtArgs ...interface{}) error {
	return &verror{
		reason:  reason,
		message: fmt.Sprintf(format, fmtArgs...),
	}
}

func validateTimeout(timeout metav1.Duration) *apis.FieldError {
	maxTimeout := time.Duration(24 * time.Hour)

	if timeout.Duration > maxTimeout {
		return apis.ErrInvalidValue(fmt.Sprintf("%s should be < 24h", timeout), "b.spec.timeout")
		//validationError("InvalidTimeout", "build timeout exceeded 24h")
	} else if timeout.Duration < 0 {
		return apis.ErrInvalidValue(fmt.Sprintf("%s should be > 0", timeout), "b.spec.timeout")
		//validationError("InvalidFormat", "build timeout should be greater than 0")
	}
	return nil
}
