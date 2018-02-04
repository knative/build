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
    importpath = "github.com/google/build-crd/pkg/apis/cloudbuild/v1alpha1",
    visibility = ["//visibility:public"],
    deps = [
        "//pkg/apis/cloudbuild:go_default_library",
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
    data = [
        "@cloudbuilders//:bazel/cloudbuild.yaml",
    ],
    embed = [":go_default_library"],
    importpath = "github.com/google/build-crd/pkg/apis/cloudbuild/v1alpha1",
    deps = ["//pkg/buildtest:go_default_library"],
)
