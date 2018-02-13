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

# HACK HACK HACK
# Git seems to ignore $HOME/.ssh and look in /root/.ssh for unknown reasons.
# As a workaround, symlink /root/.ssh to where we expect the $HOME to land.
# This means SSH auth only works for our built-in git support, and not
# corev1.Containers.
container_image(
    name = "fix_dotssh",
    base = "@git_base//image",
    symlinks = {
        "/root/.ssh": "/builder/home/.ssh",
    },
)

load("@io_bazel_rules_docker//go:image.bzl", "go_image")

go_image(
    name = "image",
    base = ":fix_dotssh",
    binary = ":git-init",
)
