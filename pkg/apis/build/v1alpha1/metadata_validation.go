package v1alpha1

import (
	"strings"

	"github.com/knative/pkg/apis"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	maxLength = 63
)

func validateObjectMetadata(meta metav1.Object) *apis.FieldError {
	name := meta.GetName()

	if strings.Contains(name, ".") {
		return &apis.FieldError{
			Message: "Invalid resource name: special character . must not be present",
			Paths:   []string{"name"},
		}
	}

	if len(name) > maxLength {
		return &apis.FieldError{
			Message: "Invalid resource name: length must be no more than 63 characters",
			Paths:   []string{"name"},
		}
	}
	return nil
}
