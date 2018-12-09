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
	"fmt"
	"time"

	"github.com/knative/pkg/apis"
)

// Validate Build
func (b *Build) Validate() *apis.FieldError {
	return validateObjectMetadata(b.GetObjectMeta()).ViaField("metadata").Also(b.Spec.Validate().ViaField("spec"))
}

// Validate for build spec
func (bs *BuildSpec) Validate() *apis.FieldError {
	if bs.Template == nil && len(bs.Steps) == 0 {
		return apis.ErrMissingOneOf("template", "steps")
	}
	if bs.Template != nil && len(bs.Steps) > 0 {
		return apis.ErrMultipleOneOf("template", "steps")
	}
	//removing below conditional because bs.Template.Name is already being checked in bs.Template.Validate()
	/*
		if bs.Template != nil && bs.Template.Name == "" {
			apis.ErrMissingField("name").ViaField("template")
		}*/
	// If a build specifies a template, all the template's parameters without
	// defaults must be satisfied by the build's parameters.
	if bs.Template != nil {
		return bs.Template.Validate().ViaField("template")
	}

	/*below method potentially has a bug:
	it does not Validate if only a "Source" has been set, it only validates if multiple sources have been set
	*/
	if err := bs.validateSources().Also(ValidateVolumes(bs.Volumes)).Also(bs.validateTimeout()).Also(validateSteps(bs.Steps)); err != nil {
		return err
	}

	/*if err := ValidateVolumes(bs.Volumes).; err != nil {
		return err
	}*/
	/*
		if err := bs.validateTimeout(); err != nil {
			return err
		}

		if err := validateSteps(bs.Steps); err != nil {
			return err
		}*/
	return nil
}

// Validate templateKind
func (b *TemplateInstantiationSpec) Validate() *apis.FieldError {
	if b == nil {
		return nil
	}
	if b.Name == "" {
		return apis.ErrMissingField("name")
	}
	if b.Kind != "" {
		switch b.Kind {
		case ClusterBuildTemplateKind,
			BuildTemplateKind:
			return nil
		default:
			return apis.ErrInvalidValue(string(b.Kind), "kind")
		}
	}
	return nil
}

// Validate build timeout
func (bs *BuildSpec) validateTimeout() *apis.FieldError {
	if bs.Timeout == nil {
		return nil
	}
	maxTimeout := time.Duration(24 * time.Hour)

	if bs.Timeout.Duration > maxTimeout {
		return apis.ErrInvalidValue(fmt.Sprintf("%s should be < 24h", bs.Timeout), "timeout")
	} else if bs.Timeout.Duration < 0 {
		return apis.ErrInvalidValue(fmt.Sprintf("%s should be > 0", bs.Timeout), "timeout")
	}
	return nil
}

// Validate source
func (bs BuildSpec) validateSources() *apis.FieldError {
	var subPathExists bool
	var emptyTargetPath bool
	names := map[string]string{}
	pathtree := pathTree{
		nodeMap: map[string]map[string]string{},
	}

	// both source and sources cannot be defined in build
	if len(bs.Sources) > 0 && bs.Source != nil {
		return apis.ErrMultipleOneOf("source", "sources")
	}
	for _, source := range bs.Sources {
		// check all source have unique names
		if _, ok := names[source.Name]; ok {
			return apis.ErrInvalidKeyName("SourceName", "name", "Duplicate Found").ViaField("source").ViaField("sources")
		}
		// multiple sources cannot have subpath defined
		if source.SubPath != "" {
			if subPathExists {
				return apis.ErrInvalidValue(source.SubPath, "subpath").ViaField("source").ViaField("sources")
			}
			subPathExists = true
		}
		names[source.Name] = ""

		if source.TargetPath == "" {
			if source.Custom != nil {
				continue
			}
			if emptyTargetPath {
				return apis.ErrInvalidValue("Empty Target Path", "targetpath").ViaField("source").ViaField("sources")
			}
			emptyTargetPath = true
		} else {
			if source.Custom != nil {
				return apis.ErrInvalidValue(source.TargetPath, "targetpath").ViaField("source").ViaField("sources")
			}
			if err := insertNode(source.TargetPath, pathtree); err != nil {
				return err
			}
		}
	}
	return nil
}
