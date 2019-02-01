/*
Copyright 2019 The Knative Authors

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

package buildtemplate

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	caching "github.com/knative/caching/pkg/apis/caching/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestmissingImageCaches(t *testing.T) {
	tests := []struct {
		name     string
		desired  []caching.Image
		observed []*caching.Image
		want     []caching.Image
	}{{
		name: "missing",
		desired: []caching.Image{{
			ObjectMeta: metav1.ObjectMeta{
				Name: "bar",
			},
		}, {
			ObjectMeta: metav1.ObjectMeta{
				Name: "foo",
			},
		},
		},
		observed: []*caching.Image{&caching.Image{
			ObjectMeta: metav1.ObjectMeta{
				Name: "coo",
			},
		}, &caching.Image{
			ObjectMeta: metav1.ObjectMeta{
				Name: "foo",
			},
		},
		},
		want: []caching.Image{{
			ObjectMeta: metav1.ObjectMeta{
				Name: "coo",
			},
		},
		},
	},
		{
			name: "no missing",
			desired: []caching.Image{{
				ObjectMeta: metav1.ObjectMeta{
					Name: "bar",
				},
			}, {
				ObjectMeta: metav1.ObjectMeta{
					Name: "foo",
				},
			},
			},
			observed: []*caching.Image{&caching.Image{
				ObjectMeta: metav1.ObjectMeta{
					Name: "bar",
				},
			}, &caching.Image{
				ObjectMeta: metav1.ObjectMeta{
					Name: "foo",
				},
			},
			},
			want: []caching.Image{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := missingImageCaches(test.desired, test.observed)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("MakeImageCache (-want, +got) = %v", diff)
			}
		})
	}
}
