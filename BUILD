package(default_visibility = ["//visibility:public"])

load("@io_bazel_rules_go//go:def.bzl", "gazelle", "go_prefix")

go_prefix("github.com/google/build-crd")

gazelle(
    name = "gazelle",
    external = "vendored",
)

load("@k8s_object//:defaults.bzl", "k8s_object")

k8s_object(
    name = "controller",
    images = {
        "build-controller:latest": "//cmd/controller:image",
        "git-image:latest": "//cmd/git-init:image",
    },
    template = "controller.yaml",
)

k8s_object(
    name = "namespace",
    template = "namespace.yaml",
)

k8s_object(
    name = "serviceaccount",
    template = "serviceaccount.yaml",
)

k8s_object(
    name = "clusterrolebinding",
    template = "clusterrolebinding.yaml",
)

k8s_object(
    name = "build",
    template = "build.yaml",
)

k8s_object(
    name = "buildtemplate",
    template = "buildtemplate.yaml",
)

load("@io_bazel_rules_k8s//k8s:objects.bzl", "k8s_objects")

k8s_objects(
    name = "authz",
    objects = [
        ":serviceaccount",
        ":clusterrolebinding",
    ],
)

k8s_objects(
    name = "everything",
    objects = [
        ":namespace",
        ":authz",
        ":build",
        ":buildtemplate",
        ":controller",
    ],
)

k8s_object(
    name = "example-build",
    template = "example-build.yaml",
)

k8s_object(
    name = "example-build-template",
    template = "example-build-template.yaml",
)

k8s_object(
    name = "example-build-from-template",
    template = "example-build-from-template.yaml",
)
