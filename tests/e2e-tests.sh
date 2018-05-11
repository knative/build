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

# This script runs the end-to-end tests against the build controller
# built from source.
# It is started by prow for each PR.
# For convenience, it can also be executed manually.

# If you already have the *_OVERRIDE environment variables set, call
# this script with the --run-tests arguments and it will use the cluster
# and run the tests.

# Calling this script without arguments will create a new cluster in
# project $PROJECT_ID, start the controller, run the tests and delete
# the cluster.
# $KO_DOCKER_REPO must point to a valid writable docker repo.

# Test cluster parameters and location of generated test images
readonly E2E_CLUSTER_NAME=buildcrd-e2e-cluster
readonly E2E_CLUSTER_ZONE=us-central1-a
readonly E2E_CLUSTER_NODES=2
readonly E2E_CLUSTER_MACHINE=n1-standard-2
readonly GKE_VERSION=v1.9.6-gke.1
readonly TEST_RESULT_FILE=/tmp/buildcrd-e2e-result

# Unique identifier for this test execution
# uuidgen is not available in kubekins images
readonly UUID=$(cat /proc/sys/kernel/random/uuid)

# Useful environment variables
[[ $USER == "prow" ]] && IS_PROW=1 || IS_PROW=0
readonly IS_PROW
readonly SCRIPT_CANONICAL_PATH="$(readlink -f ${BASH_SOURCE})"
readonly BUILD_ROOT="$(dirname ${SCRIPT_CANONICAL_PATH})/.."

# Save state to restore when complete
readonly OG_DOCKER_REPO="${KO_DOCKER_REPO}"
readonly OG_K8S_CONTEXT=$(kubectl config current-context || true)

go get -u github.com/google/go-containerregistry/cmd/ko

function header() {
  echo "================================================="
  echo $1
  echo "================================================="
}

function cleanup_env() {
  header "Restoring Environment"
  export KO_DOCKER_REPO="${OG_DOCKER_REPO}"
  kubectl config use-context "${OG_K8S_CLUSTER}" || true
}

function teardown() {
  header "Tearing down test environment"
  # Free resources in GCP project.
  ko delete --ignore-not-found=true -R -f tests/
  ko delete --ignore-not-found=true -f config/

  # Delete images when using prow.
  if (( IS_PROW )); then
    delete_build_images
  else
    cleanup_env
  fi
}

function delete_build_images() {
  local all_images=""
  for image in build-controller creds-image git-image test ; do
    all_images="${all_images} ${KO_DOCKER_REPO}/${image}"
  done
  gcloud -q container images delete ${all_images}
}

function exit_if_test_failed() {
  [[ $? -ne 0 ]] && exit 1
}

function dump_tests_status() {
  # If formatting fail for any reason, use yaml as a fall back.
  kubectl get builds -o=custom-columns-file=./tests/columns.txt || \
    kubectl get builds -oyaml
}

# Script entry point.

cd ${BUILD_ROOT}

# Show help if bad arguments are passed.
if [[ -n $1 && $1 != "--run-tests" ]]; then
  echo "usage: $0 [--run-tests]"
  exit 1
fi

# No argument provided, create the test cluster.

if [[ -z $1 ]]; then
  header "Creating test cluster"
  # Smallest cluster required to run the end-to-end-tests
  CLUSTER_CREATION_ARGS=(
    --gke-create-args="--enable-autoscaling --min-nodes=1 --max-nodes=${E2E_CLUSTER_NODES} --scopes=cloud-platform"
    --gke-shape={\"default\":{\"Nodes\":${E2E_CLUSTER_NODES}\,\"MachineType\":\"${E2E_CLUSTER_MACHINE}\"}}
    --provider=gke
    --deployment=gke
    --gcp-node-image=cos
    --cluster="${E2E_CLUSTER_NAME}"
    --gcp-zone="${E2E_CLUSTER_ZONE}"
    --gcp-network=ela-e2e-net
    --gke-environment=prod
  )
  if (( ! IS_PROW )); then
    CLUSTER_CREATION_ARGS+=(--gcp-project=${PROJECT_ID:?"PROJECT_ID must be set to the GCP project where the tests are run."})
  else
    # On prow, set bogus SSH keys for kubetest, we're not using them.
    touch $HOME/.ssh/google_compute_engine.pub
    touch $HOME/.ssh/google_compute_engine
  fi
  # Assume test failed (see more details at the end of this script).
  echo -n "1"> ${TEST_RESULT_FILE}
  kubetest "${CLUSTER_CREATION_ARGS[@]}" \
    --up \
    --down \
    --extract "${GKE_VERSION}" \
    --test-cmd "${SCRIPT_CANONICAL_PATH}" \
    --test-cmd-args --run-tests
  exit $(cat ${TEST_RESULT_FILE})
fi

# --run-tests passed as first argument, run the tests.

# Set the required variables if necessary.

if [[ -z ${KO_DOCKER_REPO} ]]; then
  export KO_DOCKER_REPO=gcr.io/$(gcloud config get-value project)
fi

# Fresh new test cluster, set cluster-admin.
# Get the password of the admin and use it, as the service account (or the user)
# might not have the necessary permission.
passwd=$(gcloud container clusters describe ${E2E_CLUSTER_NAME} --zone=${E2E_CLUSTER_ZONE} | \
    grep password | cut -d' ' -f4)
kubectl --username=admin --password=$passwd create clusterrolebinding cluster-admin-binding \
    --clusterrole=cluster-admin \
    --user=${K8S_USER_OVERRIDE}

# Build and start the controller.

echo "================================================="
echo "* Cluster is $(kubectl config current-context)"
echo "* Docker is ${KO_DOCKER_REPO}"

header "Building and starting the controller"
trap teardown EXIT

echo "Deleting any previous controller instance"
ko delete --ignore-not-found=true -f config/

if (( IS_PROW )); then
  echo "Authenticating to GCR"
  # kubekins-e2e images lack docker-credential-gcr, install it manually.
  # TODO(adrcunha): Remove this step once docker-credential-gcr is available.
  gcloud components install docker-credential-gcr
  docker-credential-gcr configure-docker
  echo "Successfully authenticated"
fi

ko apply -f config/
exit_if_test_failed
# Make sure that are no builds or build templates in the current namespace.
kubectl delete builds --all
kubectl delete buildtemplates --all

# Run the tests

header "Running tests"

ko apply -R -f tests/
exit_if_test_failed

# Wait for tests to finish.
tests_finished=0
for i in {1..60}; do
  finished="$(kubectl get builds --output=jsonpath='{.items[*].status.conditions[*].status}')"
  if [[ ! "$finished" == *"False"* ]]; then
    tests_finished=1
    break
  fi
  sleep 5
done
if (( ! tests_finished )); then
  echo "ERROR: tests timed out"
  dump_tests_status
  exit 1
fi

# Check that tests passed.
tests_passed=1
for expected_status in complete failed invalid; do
  results="$(kubectl get builds -l expect=${expected_status} \
      --output=jsonpath='{range .items[*]}{.metadata.name}={.status.conditions[*].state}{" "}{end}')"
  for result in ${results}; do
    if [[ ! "${result,,}" == *"=${expected_status}" ]]; then
      echo "ERROR: test ${result} but should be ${expected_status}"
      tests_passed=0
    fi
  done
done
if (( ! tests_passed )); then
  echo "ERROR: one or more tests failed"
  dump_tests_status
  exit 1
fi

echo "*** ALL TESTS PASSED ***"

# kubetest teardown might fail and thus incorrectly report failure of the
# script, even if the tests pass.
# We store the real test result to return it later, ignoring any teardown
# failure in kubetest.
# TODO(adrcunha): Get rid of this workaround.
echo -n "0"> ${TEST_RESULT_FILE}

exit 0
