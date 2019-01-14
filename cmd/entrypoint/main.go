/*
Copyright 2018 The Knative Authors.

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

package main

import (
	"os"

	"github.com/knative/build/pkg/entrypoint"
	"github.com/knative/build/pkg/entrypoint/options"
	"github.com/knative/pkg/logging"
)

/*
The tool is used to rewrite the entrypoint of a container image.
To override the base shell image update `.ko.yaml` file.

To use it, run
```
image: github.com/knative/build/cmd/entrypoint
```

It used in knative/build as a method of running containers in
order that are in the same pod this is done by:
1) for the Pod(containing user Steps) created by a Build,
create a shared directory with the entrypoint binary
2) change the entrypoint of all the user specified containers in Steps to be the
entrypoint binary with configuration to run the user specified entrypoint with some custom logic
3) one piece of "custom logic" is having the entrypoint binary wait for the previous step
as seen in knative/build/pkg/entrypoint/run.go -- waitForPrevStep()

Example use case:
```
image: github.com/build-pipeline/cmd/entrypoint
args: ['-args', 'ARGUMENTS_FOR_SHELL_COMMAND']
"args":["/kaniko/executor"],shouldWaitForPrevStep":false,"preRunFile":"/tools/0","shouldRunPostRun":true,"postRunFile":"/tools/1"
```
*/

func main() {
	logger, _ := logging.NewLogger("", "entrypoint")
	defer logger.Sync()

	o := entrypoint.NewOptions()
	if err := options.Load(o); err != nil {
		logger.Fatalf("Could not resolve options: %v", err)
	}

	if err := o.Validate(); err != nil {
		logger.Fatalf("Invalid options: %v", err)
	}

	os.Exit(o.Run(logger))
}
