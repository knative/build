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
package v1alpha1

import (
	"strings"

	"github.com/knative/pkg/apis"
)

// Node is constructed to track targetPath structure in source
// Value tracks the value of node. Parent holds the address of parent nodes
type Node struct {
	Parent *Node
	Value  string
}

// insert function breaks the path and inserts every element into nodeMap
// insert function throws error if the path is already traversed by any
// previous targetPaths
func insertNode(path string, nodeMap map[string]*Node) *apis.FieldError {
	parts := strings.Split(path, "/")
	visitedNodes := 0

	for i, part := range parts {

		_, ok := nodeMap[part]
		if ok {
			if visitedNodes > 1 {
				return apis.ErrMultipleOneOf("b.spec.sources.targetPath")
			}
			visitedNodes++
			continue
		}

		// build and add node since it does not exist in map
		var parent *Node
		if i-1 >= 0 {
			parent = nodeMap[parts[i-1]]
		}
		n := &Node{
			Value:  part,
			Parent: parent,
		}
		nodeMap[part] = n
	}
	return nil
}
