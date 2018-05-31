package(default_visibility = ["//visibility:public"])

load("@io_bazel_rules_go//go:def.bzl", "go_binary", "go_library")

go_library(
    name = "go_default_library",
    srcs = ["main.go"],
    importpath = "github.com/knative/build/tests/custom-env",
    visibility = ["//visibility:private"],
    deps = ["//vendor/github.com/golang/glog:go_default_library"],
)

go_binary(
    name = "custom-env",
    embed = [":go_default_library"],
    pure = "on",
    visibility = ["//visibility:public"],
)

load("@io_bazel_rules_docker//go:image.bzl", "go_image")

go_image(
    name = "test-image",
    binary = ":custom-env",
)

load("@k8s_object//:defaults.bzl", "k8s_object")

k8s_object(
    name = "test",
    images = {
        "github.com/knative/build/tests/custom-env": ":test-image",
    },
    template = "test.yaml",
)
