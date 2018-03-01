package(default_visibility = ["//visibility:public"])

load("@io_bazel_rules_go//go:def.bzl", "go_binary", "go_library")

go_library(
    name = "go_default_library",
    srcs = ["main.go"],
    importpath = "github.com/elafros/build-crd/cmd/creds-init",
    visibility = ["//visibility:private"],
    deps = [
        "//pkg/credentials:go_default_library",
        "//pkg/credentials/dockercreds:go_default_library",
        "//pkg/credentials/gitcreds:go_default_library",
        "//vendor/github.com/golang/glog:go_default_library",
    ],
)

go_binary(
    name = "creds-init",
    embed = [":go_default_library"],
    importpath = "github.com/elafros/build-crd/cmd/creds-init",
    visibility = ["//visibility:public"],
)

load("@io_bazel_rules_docker//go:image.bzl", "go_image")

go_image(
    name = "image",
    # Share the git base, since we need to download it anyways.
    base = "@git_base//image",
    binary = ":creds-init",
)
