package(default_visibility = ["//visibility:public"])

load("@io_bazel_rules_go//go:def.bzl", "go_binary", "go_library")

go_library(
    name = "go_default_library",
    srcs = ["main.go"],
    importpath = "github.com/elafros/build/tests/workingdir",
    visibility = ["//visibility:private"],
    deps = ["//vendor/github.com/golang/glog:go_default_library"],
)

go_binary(
    name = "workingdir",
    embed = [":go_default_library"],
    pure = "on",
    visibility = ["//visibility:public"],
)

load("@io_bazel_rules_docker//go:image.bzl", "go_image")

go_image(
    name = "test-image",
    binary = ":workingdir",
)

load("@io_bazel_rules_docker//container:container.bzl", "container_image")

container_image(
    name = "test-image-with-files",
    base = ":test-image",
    # Create a file under the directory, so that it exists.
    directory = "/a/path/too/far",
    files = [":test.yaml"],
    # Set the built-in working directory elsewhere
    workdir = "/workspace",
)

load("@k8s_object//:defaults.bzl", "k8s_object")

k8s_object(
    name = "test",
    images = {
        "test:latest": ":test-image-with-files",
    },
    template = "test.yaml",
)
