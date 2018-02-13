package(default_visibility = ["//visibility:public"])

load("@io_bazel_rules_go//go:def.bzl", "go_binary", "go_library")

go_library(
    name = "go_default_library",
    srcs = ["main.go"],
    importpath = "github.com/google/build-crd/cmd/git-init",
    visibility = ["//visibility:private"],
    deps = ["//vendor/github.com/golang/glog:go_default_library"],
)

go_binary(
    name = "git-init",
    embed = [":go_default_library"],
    importpath = "github.com/google/build-crd/cmd/git-init",
    visibility = ["//visibility:public"],
)

load("@io_bazel_rules_docker//container:container.bzl", "container_image")

container_image(
    name = "strip_cred_helper",
    base = "@git_base//image",
    directory = "/etc",
    files = ["gitconfig"],
)

load("@io_bazel_rules_docker//go:image.bzl", "go_image")

go_image(
    name = "image",
    base = ":strip_cred_helper",
    binary = ":git-init",
)
