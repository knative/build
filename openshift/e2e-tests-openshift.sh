#!/bin/sh 

source $(dirname $0)/../vendor/github.com/knative/test-infra/scripts/e2e-tests.sh

set -x

export USER=$KUBE_SSH_USER #satisfy e2e_flags.go#initializeFlags()
export OPENSHIFT_REGISTRY=registry.svc.ci.openshift.org
export TEST_NAMESPACE=build-tests
export BUILD_NAMESPACE=knative-build

env

function install_build(){
  header "Installing Knative Build"
  # Grant the necessary privileges to the service accounts Knative will use:
  oc adm policy add-scc-to-user anyuid -z build-controller -n knative-build
  oc adm policy add-cluster-role-to-user cluster-admin -z build-controller -n knative-build
  
  create_build

  wait_until_pods_running $BUILD_NAMESPACE || return 1
  
  header "Knative Build Installed successfully"
}

function create_build(){
  resolve_resources config/ build-resolved.yaml
  oc apply -f build-resolved.yaml
}

function resolve_resources(){
  local dir=$1
  local resolved_file_name=$2
  local registry_prefix="$OPENSHIFT_REGISTRY/$OPENSHIFT_BUILD_NAMESPACE/stable"
  for yaml in $(find $dir -name "*.yaml"); do
    echo "---" >> $resolved_file_name
    #first prefix all test images with "test-", then replace all image names with proper repository and prefix images with "build-"
    sed -e 's%\(.* image: \)\(github.com\)\(.*\/\)\(test\/\)\(.*\)%\1\2 \3\4test-\5%' $yaml | \
    sed -e 's%\(.* image: \)\(github.com\)\(.*\/\)\(.*\)%\1 '"$registry_prefix"'\:build-\4%' | \
    # process these images separately as they're passed as arguments to other containers
    sed -e 's%github.com/knative/build/cmd/creds-init%'"$registry_prefix"'\:build-creds-init%g' | \
    sed -e 's%github.com/knative/build/cmd/git-init%'"$registry_prefix"'\:build-git-init%g' | \
    sed -e 's%github.com/knative/build/cmd/nop%'"$registry_prefix"'\:build-nop%g' \
    >> $resolved_file_name
  done
}

function enable_docker_schema2(){
  cat > config.yaml <<EOF
  version: 0.1
  log:
    level: debug
  http:
    addr: :5000
  storage:
    cache:
      blobdescriptor: inmemory
    filesystem:
      rootdirectory: /registry
    delete:
      enabled: true
  auth:
    openshift:
      realm: openshift
  middleware:
    registry:
      - name: openshift
    repository:
      - name: openshift
        options:
          acceptschema2: true
          pullthrough: true
          enforcequota: false
          projectcachettl: 1m
          blobrepositorycachettl: 10m
    storage:
      - name: openshift
  openshift:
    version: 1.0
    metrics:
      enabled: false
      secret: <secret>
EOF
  oc project default
  oc create configmap registry-config --from-file=./config.yaml
  oc set volume dc/docker-registry --add --type=configmap --configmap-name=registry-config -m /etc/docker/registry/
  oc set env dc/docker-registry REGISTRY_CONFIGURATION_PATH=/etc/docker/registry/config.yaml
  oc project $TEST_NAMESPACE
}

function create_test_namespace(){
  oc new-project $TEST_NAMESPACE
  oc policy add-role-to-group system:image-puller system:serviceaccounts:$TEST_NAMESPACE -n $OPENSHIFT_BUILD_NAMESPACE
}

function run_go_e2e_tests(){
  header "Running Go e2e tests"
  go_test_e2e ./test/e2e/... --kubeconfig $KUBECONFIG || fail_test
}

function run_yaml_e2e_tests() {
  header "Running YAML e2e tests"
  resolve_resources test/ tests-resolved.yaml
  oc apply -f tests-resolved.yaml

  # The rest of this function copied from test/e2e-common.sh#run_yaml_tests()
  # The only change is "kubectl get builds" -> "oc get builds.build.knative.dev"
  oc get project
  # Wait for tests to finish.
  echo ">> Waiting for tests to finish"
  local tests_finished=0
  for i in {1..60}; do
    local finished="$(oc get builds.build.knative.dev --output=jsonpath='{.items[*].status.conditions[*].status}')"
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
    results="$(oc get builds.build.knative.dev -l expect=${expected_status} \
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

function delete_build_openshift() {
  echo ">> Bringing down Build"
  oc delete --ignore-not-found=true -f build-resolved.yaml
  # Make sure that are no builds or build templates in the knative-build namespace.
  oc delete --ignore-not-found=true builds.build.knative.dev --all -n $BUILD_NAMESPACE
  oc delete --ignore-not-found=true buildtemplates.build.knative.dev --all -n $BUILD_NAMESPACE
}

function delete_test_resources_openshift() {
  echo ">> Removing test resources (test/)"
  oc delete --ignore-not-found=true -f tests-resolved.yaml
}

 function delete_test_namespace(){
   echo ">> Deleting test namespace $TEST_NAMESPACE"
   oc policy remove-role-from-group system:image-puller system:serviceaccounts:$TEST_NAMESPACE -n $OPENSHIFT_BUILD_NAMESPACE
   oc delete project $TEST_NAMESPACE
 }

function teardown() {
  delete_test_namespace
  delete_test_resources_openshift
  delete_build_openshift
}

teardown

create_test_namespace

enable_docker_schema2

install_build

run_go_e2e_tests

run_yaml_e2e_tests
