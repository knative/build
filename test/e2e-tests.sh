#!/bin/bash

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

# This script runs the end-to-end tests against the build controller
# built from source. It is started by prow for each PR.
# For convenience, it can also be executed manually.

# If you already have the *_OVERRIDE environment variables set, call
# this script with the --run-tests arguments and it will use the cluster
# and run the tests.

# Calling this script without arguments will create a new cluster in
# project $PROJECT_ID, start the controller, run the tests and delete
# the cluster.

# Load github.com/knative/test-infra/images/prow-tests/scripts/e2e-tests.sh
[ -f /workspace/e2e-tests.sh ] \
  && source /workspace/e2e-tests.sh \
  || eval "$(docker run --entrypoint sh gcr.io/knative-tests/test-infra/prow-tests -c 'cat e2e-tests.sh')"
[ -v KNATIVE_TEST_INFRA ] || exit 1

# Helper functions.

function teardown() {
  ko delete --ignore-not-found=true -R -f test/
  ko delete --ignore-not-found=true -f config/
}

function abort_test() {
  echo "$1"
  # If formatting fails for any reason, use yaml as a fall back.
  kubectl get builds -o=custom-columns-file=./test/columns.txt || \
    kubectl get builds -oyaml
  fail_test
}

# Script entry point.

initialize $@

# Fail fast during setup.
set -o errexit
set -o pipefail

header "Building and starting the controller"
export KO_DOCKER_REPO=${DOCKER_REPO_OVERRIDE}
ko apply -f config/ || fail_test

# Handle test failures ourselves, so we can dump useful info.
set +o errexit
set +o pipefail

# Make sure that are no builds or build templates in the current namespace.
kubectl delete builds --all
kubectl delete buildtemplates --all

# Run the tests

header "Running tests"

ko apply -R -f test/ || fail_test

# Wait for tests to finish.
tests_finished=0
for i in {1..60}; do
  finished="$(kubectl get builds --output=jsonpath='{.items[*].status.conditions[*].status}')"
  if [[ ! "$finished" == *"Unknown"* ]]; then
    tests_finished=1
    break
  fi
  sleep 5
done
(( tests_finished )) || abort_test "ERROR: tests timed out"

# Check that tests passed.
tests_passed=1
for expected_status in succeeded failed; do
  results="$(kubectl get builds -l expect=${expected_status} \
      --output=jsonpath='{range .items[*]}{.metadata.name}={.status.conditions[*].state}{.status.conditions[*].status}{" "}{end}')"
  case $expected_status in
    succeeded)
      want=succeededtrue
      ;;
    failed)
      want=succeededfalse
      ;;
    *)
      echo "Invalid expected status '${expected_status}'"
      fail_test
  esac
  for result in ${results}; do
    if [[ ! "${result,,}" == *"=${want}" ]]; then
      echo "ERROR: test ${result} but should be ${want}"
      tests_passed=0
    fi
  done
done
(( tests_passed )) || abort_test "ERROR: one or more tests failed"

success
