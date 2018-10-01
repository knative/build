package v1alpha1

import (
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSetDefault(t *testing.T) {
	emptyBuild := &Build{}
	emptyBuild.SetDefaults()
	if emptyBuild.Spec.ServiceAccountName != "default" {
		t.Errorf("Expect default to be the serviceaccount name but got %s", emptyBuild.Spec.ServiceAccountName)
	}
	if emptyBuild.Spec.Timeout.Duration != DefaultTimeout {
		t.Errorf("Expect build timeout to be set")
	}
}

func TestAlreadySetDefault(t *testing.T) {
	setAccountName := "test-account-name"
	setTimeout := metav1.Duration{Duration: 20 * time.Minute}
	setDefaultBuild := &Build{
		Spec: BuildSpec{
			ServiceAccountName: setAccountName,
			Timeout:            setTimeout,
		},
	}
	setDefaultBuild.SetDefaults()
	if setDefaultBuild.Spec.ServiceAccountName != setAccountName {
		t.Errorf("Expect build.spec.serviceaccount name not to be overridden; but got %s", setDefaultBuild.Spec.ServiceAccountName)
	}
	if setDefaultBuild.Spec.Timeout != setTimeout {
		t.Errorf("Expect build.spec.timeout not to be overridden; but got %s", setDefaultBuild.Spec.Timeout)
	}
}
