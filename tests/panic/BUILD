package(default_visibility = ["//visibility:public"])

load("@io_bazel_rules_go//go:def.bzl", "go_binary", "go_library")

go_library(
    name = "go_default_library",
    srcs = ["main.go"],
    importpath = "github.com/elafros/build/tests/panic",
    visibility = ["//visibility:private"],
)

go_binary(
    name = "panic",
    embed = [":go_default_library"],
    importpath = "github.com/elafros/build/tests/panic",
    pure = "on",
    visibility = ["//visibility:public"],
)

load("@io_bazel_rules_docker//go:image.bzl", "go_image")

go_image(
    name = "test-image",
    binary = ":panic",
)

load("@k8s_object//:defaults.bzl", "k8s_object")

k8s_object(
    name = "test",
    images = {
        "test:latest": ":test-image",
    },
    template = "test.yaml",
)
