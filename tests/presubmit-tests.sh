#!/bin/bash

# Copyright 2018 Google, Inc. All rights reserved.
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

# This script runs the presubmit tests, in the right order.
# It is started by prow for each PR.
# For convenience, it can also be executed manually.

set -o errexit
set -o pipefail

source "$(dirname $(readlink -f ${BASH_SOURCE}))/library.sh"

function cleanup() {
  echo "Cleaning up for teardown"
  restore_override_vars
  # --expunge is a workaround for https://github.com/elafros/elafros/issues/366
  bazel clean --expunge || true
}

cd ${BUILD_ROOT_DIR}

# Set the required env vars to dummy values to satisfy bazel.
export DOCKER_REPO_OVERRIDE=REPO_NOT_SET

# For local runs, cleanup before and after the tests.
if (( ! IS_PROW )); then
  trap cleanup EXIT
  echo "Cleaning up for setup"
  # --expunge is a workaround for https://github.com/elafros/elafros/issues/366
  bazel clean --expunge
fi

# Tests to be performed.

# Build tests, to ensure nothing is broken.
header "Running build tests"
bazel build //cmd/... //pkg/...

# Unit tests.
header "Running unit tests"
bazel test //cmd/... //pkg/...
go test ./...

# Integration tests.
# Make sure environment variables are intact.
restore_override_vars
./tests/e2e-tests.sh
