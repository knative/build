package v1alpha1

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DefaultTimeout is 10min
const DefaultTimeout = 10 * time.Minute

// SetDefaults for build
func (b *Build) SetDefaults() {
	if b == nil {
		return
	}
	if b.Spec.ServiceAccountName == "" {
		b.Spec.ServiceAccountName = "default"
	}
	if b.Spec.Timeout.Duration == 0 {
		b.Spec.Timeout = metav1.Duration{Duration: DefaultTimeout}
	}
	if b.Spec.Template != nil && b.Spec.Template.Kind == "" {
		b.Spec.Template.Kind = BuildTemplateKind
	}
}
