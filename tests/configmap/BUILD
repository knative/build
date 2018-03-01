package(default_visibility = ["//visibility:public"])

load("@io_bazel_rules_go//go:def.bzl", "go_binary", "go_library")

go_library(
    name = "go_default_library",
    srcs = ["main.go"],
    importpath = "github.com/elafros/build-crd/tests/configmap",
    deps = ["//vendor/github.com/golang/glog:go_default_library"],
)

go_binary(
    name = "configmap",
    embed = [":go_default_library"],
    importpath = "github.com/elafros/build-crd/tests/configmap",
    pure = "on",
)

load("@io_bazel_rules_docker//go:image.bzl", "go_image")

go_image(
    name = "test-image",
    binary = ":configmap",
)

load("@k8s_object//:defaults.bzl", "k8s_object")

k8s_object(
    name = "cfgmap",
    template = "configmap.yaml",
)

k8s_object(
    name = "build",
    images = {
        "test:latest": ":test-image",
    },
    template = "build.yaml",
)

load("@io_bazel_rules_k8s//k8s:objects.bzl", "k8s_objects")

k8s_objects(
    name = "test",
    objects = [
        ":cfgmap",
        ":build",
    ],
)
