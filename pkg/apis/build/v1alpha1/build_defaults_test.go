package v1alpha1

import (
	"testing"
)

func TestSetDefault(t *testing.T) {
	emptyBuild := &Build{}
	emptyBuild.SetDefaults()
	if emptyBuild.Spec.ServiceAccountName != "default" {
		t.Errorf("Expect default to be the serviceaccount name but got %s", emptyBuild.Spec.ServiceAccountName)
	}
	if emptyBuild.Spec.Timeout.Duration != DefaultTime {
		t.Errorf("Expect build timeout to be set")
	}
}
