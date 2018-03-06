load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = ["fake.go"],
    importpath = "github.com/elafros/build/pkg/builder/google/fakecloudbuild",
    visibility = ["//visibility:public"],
    deps = ["//vendor/google.golang.org/api/cloudbuild/v1:go_default_library"],
)
