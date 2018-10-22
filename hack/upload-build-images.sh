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

# Build the base image for creds-init and git images.

: ${BUILD_BASE_REGISTRY:="gcr.io"}
: ${BUILD_BASE_REPO:="knative-releases/github.com/knative/build/build-base"}
: ${BUILD_BASE_TAG:="latest"}
: ${BUILD_BASE_IMAGE:="${BUILD_BASE_REGISTRY}/${BUILD_BASE_REPO}:${BUILD_BASE_TAG}"}

readonly BUILD_BASE_REGISTRY
readonly BUILD_BASE_REPO
readonly BUILD_BASE_TAG
readonly BUILD_BASE_IMAGE

# Build the base image for creds-init and git images.
echo "Building the build-base image"
docker build -t "${BUILD_BASE_IMAGE}" -f images/Dockerfile images/

if (( ! SKIP_BUILD_BASE_PUSH )); then
  echo "Pushing the build-base image to ${BUILD_BASE_IMAGE}"
  docker push "${BUILD_BASE_IMAGE}"
fi
