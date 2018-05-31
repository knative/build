package(default_visibility = ["//visibility:public"])

load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = [
        "build_template_types.go",
        "build_types.go",
        "doc.go",
        "register.go",
        "zz_generated.deepcopy.go",
    ],
    importpath = "github.com/knative/build/pkg/apis/build/v1alpha1",
    visibility = ["//visibility:public"],
    deps = [
        "//pkg/apis/build:go_default_library",
        "//vendor/k8s.io/api/core/v1:go_default_library",
        "//vendor/k8s.io/apimachinery/pkg/apis/meta/v1:go_default_library",
        "//vendor/k8s.io/apimachinery/pkg/runtime:go_default_library",
        "//vendor/k8s.io/apimachinery/pkg/runtime/schema:go_default_library",
    ],
)

go_test(
    name = "go_default_test",
    srcs = [
        "build_template_types_test.go",
        "build_types_test.go",
    ],
    data = glob(["testdata/**"]),
    embed = [":go_default_library"],
    deps = ["//pkg/buildtest:go_default_library"],
)
