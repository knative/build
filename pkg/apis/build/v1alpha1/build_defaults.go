package v1alpha1

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SetDefaults for build
func (b *Build) SetDefaults() {
	saName := b.Spec.ServiceAccountName
	if saName == "" {
		saName = "default"
	}
	if b.Spec.Timeout.Duration == 0 {
		b.Spec.Timeout = metav1.Duration{Duration: 10 * time.Minute}
	}
}
