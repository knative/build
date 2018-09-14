/*
Copyright 2018 The Knative Authors
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

package buildtest

import (
	"encoding/json"

	"github.com/sergi/go-diff/diffmatchpatch"
)

func JSONDiff(l, r interface{}) string {
	lb, err := json.MarshalIndent(l, "", " ")
	if err != nil {
		panic(err.Error())
	}
	rb, err := json.MarshalIndent(r, "", " ")
	if err != nil {
		panic(err.Error())
	}

	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(string(lb), string(rb), true)
	for _, d := range diffs {
		if d.Type != diffmatchpatch.DiffEqual {
			return dmp.DiffPrettyText(diffs)
		}
	}
	return ""
}
