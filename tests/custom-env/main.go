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

package main

import (
	"os"

	"github.com/golang/glog"
)

const (
	expectedVar1 = "foo"
	expectedVar2 = "bar"
)

func main() {
	var1 := os.Getenv("MY_VAR1")
	if var1 != expectedVar1 {
		glog.Fatalf("Unexpected value for $MY_VAR1, want %q, but got %q", expectedVar1, var1)
	}
	var2 := os.Getenv("MY_VAR2")
	if var2 != expectedVar2 {
		glog.Fatalf("Unexpected value for $MY_VAR2, want %q, but got %q", expectedVar2, var2)
	}
}
