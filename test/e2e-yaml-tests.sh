#!/usr/bin/env bash

# Copyright 2018 The Knative Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# This script runs the YAML end-to-end tests against the build controller
# built from source. It is not run by prow for each PR, see e2e-tests.sh for that.

source $(dirname $0)/../vendor/github.com/knative/test-infra/scripts/e2e-tests.sh
source $(dirname $0)/e2e-common.sh

# Fail fast during setup.
set -o errexit
set -o pipefail

header "Building and starting the controller"
ko apply -f config/ || fail_test

# Handle test failures ourselves, so we can dump useful info.
set +o errexit
set +o pipefail

# Make sure that are no builds or build templates in the current namespace.
kubectl delete --ignore-not-found=true builds.build.knative.dev --all
kubectl delete --ignore-not-found=true buildtemplates --all

# Run the tests

failed=0

header "Running YAML e2e tests"
if ! run_yaml_tests; then
  failed=1
  echo "ERROR: one or more YAML tests failed"
  # If formatting fails for any reason, use yaml as a fall back.
  kubectl get builds.build.knative.dev -o=custom-columns-file=./test/columns.txt || \
    kubectl get builds.build.knative.dev -oyaml
fi

(( failed )) && fail_test
success
