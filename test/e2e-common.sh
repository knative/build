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

function teardown() {
  ko delete --ignore-not-found=true -R -f test/
  ko delete --ignore-not-found=true -f config/
}

function run_yaml_tests() {
  echo ">> Starting tests"
  ko apply -R -f test/ || return 1

  # Wait for tests to finish.
  echo ">> Waiting for tests to finish"
  local tests_finished=0
  for i in {1..60}; do
    local finished="$(kubectl get build.build.knative.dev --output=jsonpath='{.items[*].status.conditions[*].status}')"
    if [[ ! "$finished" == *"Unknown"* ]]; then
      tests_finished=1
      break
    fi
    sleep 5
  done
  if (( ! tests_finished )); then
    echo "ERROR: tests timed out"
    return 1
  fi

  # Check that tests passed.
  local failed=0
  echo ">> Checking test results"
  for expected_status in succeeded failed; do
    results="$(kubectl get build.build.knative.dev -l expect=${expected_status} \
	--output=jsonpath='{range .items[*]}{.metadata.name}={.status.conditions[*].type}{.status.conditions[*].status}{" "}{end}')"
    case $expected_status in
      succeeded)
	want=succeededtrue
	;;
      failed)
	want=succeededfalse
	;;
      *)
	echo "ERROR: Invalid expected status '${expected_status}'"
	failed=1
	;;
    esac
    for result in ${results}; do
      if [[ ! "${result,,}" == *"=${want}" ]]; then
	echo "ERROR: test ${result} but should be ${want}"
	failed=1
      fi
    done
  done
  (( failed )) && return 1
  echo ">> All YAML tests passed"
  return 0
}
