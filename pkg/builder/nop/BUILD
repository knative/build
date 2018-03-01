load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = ["builder.go"],
    importpath = "github.com/elafros/build-crd/pkg/builder/nop",
    visibility = ["//visibility:public"],
    deps = [
        "//pkg/apis/cloudbuild/v1alpha1:go_default_library",
        "//pkg/builder:go_default_library",
        "//vendor/k8s.io/api/core/v1:go_default_library",
        "//vendor/k8s.io/apimachinery/pkg/apis/meta/v1:go_default_library",
    ],
)

go_test(
    name = "go_default_test",
    srcs = ["builder_test.go"],
    embed = [":go_default_library"],
    importpath = "github.com/elafros/build-crd/pkg/builder/nop",
    deps = [
        "//pkg/apis/cloudbuild/v1alpha1:go_default_library",
        "//pkg/builder:go_default_library",
    ],
)
