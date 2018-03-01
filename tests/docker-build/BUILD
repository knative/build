package(default_visibility = ["//visibility:public"])

load("@k8s_object//:defaults.bzl", "k8s_object")

k8s_object(
    name = "template",
    template = "template.yaml",
)

# Generate a build.yaml based on the DOCKER_REPO_OVERRIDE that folks set.
genrule(
    name = "gen-with-repo",
    srcs = [],
    outs = ["build.yaml"],
    cmd = """
DOCKER_REPO_OVERRIDE=$$(cat bazel-out/stable-status.txt | grep STABLE_DOCKER_REPO | cut -d' ' -f 2)
cat > $(location build.yaml) <<EOF
apiVersion: build.dev/v1alpha1
kind: Build
metadata:
  name: test-docker-build
  labels:
    expect: complete
spec:
  source:
    custom:
      image: gcr.io/cloud-builders/git
      args: ["clone", "https://github.com/GoogleCloudPlatform/cloud-builders.git", "."]

  template:
    name: docker-build-push
    arguments:
    - name: DIRECTORY
      value: /workspace/wget
    - name: IMAGE
      value: $${DOCKER_REPO_OVERRIDE}/on-cluster-test/wget
EOF
""",
    stamp = 1,
)

k8s_object(
    name = "build",
    template = ":build.yaml",
)

load("@io_bazel_rules_k8s//k8s:objects.bzl", "k8s_objects")

k8s_objects(
    name = "test",
    objects = [
        ":template",
        ":build",
    ],
)
