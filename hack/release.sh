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

set -o errexit
set -o pipefail

source "$(dirname $(readlink -f ${BASH_SOURCE}))/../tests/library.sh"

function cleanup() {
  restore_override_vars
}

cd ${BUILD_ROOT_DIR}
trap cleanup EXIT

install_ko

echo "@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@"
echo "@@@@ RUNNING RELEASE VALIDATION TESTS @@@@"
echo "@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@"

# Run tests.
./tests/presubmit-tests.sh

echo "@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@"
echo "@@@@     BUILDING THE RELEASE    @@@@"
echo "@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@"

# Build and push the base image for creds-init and git images.
docker build -t $BUILD_RELEASE_GCR/build-base -f images/Dockerfile images/
docker push $BUILD_RELEASE_GCR/build-base

# Set the repository
export KO_DOCKER_REPO=${BUILD_RELEASE_GCR}
# Build should not try to deploy anything, use a bogus value for cluster.
export K8S_CLUSTER_OVERRIDE=CLUSTER_NOT_SET
export K8S_USER_OVERRIDE=USER_NOT_SET
export DOCKER_REPO_OVERRIDE=DOCKER_NOT_SET

# If this is a prow job, authenticate against GCR.
(( IS_PROW )) && gcr_auth

echo "Building build-crd"
ko resolve -f config/ > release.yaml

echo "Publishing release.yaml"
gsutil cp release.yaml gs://build-crd/latest/release.yaml

echo "New release published successfully"

# TODO(mattmoor): Create other aliases?
