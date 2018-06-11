/*
Copyright 2018 Google LLC
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

package logkey

const (
	// ControllerType is the key used for controller type in structured logs
	ControllerType = "build.dev/controller"

	// Namespace is the key used for namespace in structured logs
	Namespace = "build.dev/namespace"

	// Build is the key used for build name in structured logs
	Build = "build.dev/build"

	// BuildTemplate is the key used for build name in structured logs
	BuildTemplate = "build.dev/buildtemplate"

	// JSONConfig is the key used for JSON configurations (not to be confused by the Configuration object)
	JSONConfig = "build.dev/jsonconfig"

	// Kind is the key used to represent kind of an object in logs
	Kind = "build.dev/kind"

	// Name is the key used to represent name of an object in logs
	Name = "build.dev/name"

	// Operation is the key used to represent an operation in logs
	Operation = "build.dev/operation"

	// Resource is the key used to represent a resource in logs
	Resource = "build.dev/resource"

	// SubResource is a generic key used to represent a sub-resource in logs
	SubResource = "build.dev/subresource"

	// UserInfo is the key used to represent a user information in logs
	UserInfo = "build.dev/userinfo"

	// Pod is the key used to represent a pod's name in logs
	Pod = "build.dev/pod"
)
