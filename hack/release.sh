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

set -o errexit
set -o pipefail

readonly BUILD_ROOT=$(dirname ${BASH_SOURCE})/..
readonly OG_DOCKER_REPO="${KO_DOCKER_REPO}"

function header() {
  echo "*************************************************"
  echo "** $1"
  echo "*************************************************"
}

function cleanup() {
  export KO_DOCKER_REPO="${OG_DOCKER_REPO}"
}

cd ${BUILD_ROOT}
trap cleanup EXIT

header "TEST PHASE"

# Run tests.
./tests/presubmit-tests.sh

header "BUILD PHASE"

# Set the repository to the official one:
export KO_DOCKER_REPO=gcr.io/build-crd

# If this is a prow job, authenticate against GCR.
if [[ $USER == "prow" ]]; then
  echo "Authenticating to GCR"
  # kubekins-e2e images lack docker-credential-gcr, install it manually.
  # TODO(adrcunha): Remove this step once docker-credential-gcr is available.
  gcloud components install docker-credential-gcr
  docker-credential-gcr configure-docker
  echo "Successfully authenticated"
fi

echo "Building build-crd"
ko resolve -f config/ > release.yaml

echo "Publishing release.yaml"
gsutil cp release.yaml gs://build-crd/latest/release.yaml

echo "New release published successfully"

# TODO(mattmoor): Create other aliases?
