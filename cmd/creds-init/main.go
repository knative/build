/*
Copyright 2018 Google, Inc. All rights reserved.

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
	"flag"

	"github.com/golang/glog"

	"github.com/elafros/build/pkg/credentials"
	"github.com/elafros/build/pkg/credentials/dockercreds"
	"github.com/elafros/build/pkg/credentials/gitcreds"
)

func main() {
	flag.Parse()

	builders := []credentials.Builder{dockercreds.NewBuilder(), gitcreds.NewBuilder()}
	for _, c := range builders {
		if err := c.Write(); err != nil {
			glog.Fatalf("Error initializing credentials: %v", err)
		}
	}
	glog.Infof("Credentials initialized.")
}
