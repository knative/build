package v1alpha1

import (
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestMetadataInvalidLongName(t *testing.T) {

	invalidMetas := []*metav1.ObjectMeta{
		&metav1.ObjectMeta{Name: strings.Repeat("s", maxLength+1)},
		&metav1.ObjectMeta{Name: "bad.name"},
	}
	for _, invalidMeta := range invalidMetas {
		if err := validateObjectMetadata(invalidMeta); err == nil {
			t.Errorf("Failed to validate object meta data: %s", err)
		}
	}
}
