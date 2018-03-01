package(default_visibility = ["//visibility:public"])

load("@io_bazel_rules_go//go:def.bzl", "go_binary", "go_library")

go_library(
    name = "go_default_library",
    srcs = ["main.go"],
    importpath = "github.com/elafros/build-crd/tests/default-workingdir",
    visibility = ["//visibility:private"],
    deps = ["//vendor/github.com/golang/glog:go_default_library"],
)

go_binary(
    name = "default-workingdir",
    embed = [":go_default_library"],
    importpath = "github.com/elafros/build-crd/tests/default-workingdir",
    pure = "on",
    visibility = ["//visibility:public"],
)

load("@io_bazel_rules_docker//go:image.bzl", "go_image")

go_image(
    name = "test-image",
    binary = ":default-workingdir",
)

load("@k8s_object//:defaults.bzl", "k8s_object")

k8s_object(
    name = "test",
    images = {
        "test:latest": ":test-image",
    },
    template = "test.yaml",
)
