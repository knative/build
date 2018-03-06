load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = ["initialize.go"],
    importpath = "github.com/elafros/build/pkg/credentials",
    visibility = ["//visibility:public"],
    deps = ["//vendor/k8s.io/api/core/v1:go_default_library"],
)
