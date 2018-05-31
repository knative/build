load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = [
        "convert.go",
        "env.go",
        "source.go",
        "step.go",
        "volume.go",
    ],
    importpath = "github.com/knative/build/pkg/builder/google/convert",
    visibility = ["//visibility:public"],
    deps = [
        "//pkg/apis/build/v1alpha1:go_default_library",
        "//pkg/builder/validation:go_default_library",
        "//vendor/google.golang.org/api/cloudbuild/v1:go_default_library",
        "//vendor/k8s.io/api/core/v1:go_default_library",
    ],
)

go_test(
    name = "go_default_test",
    srcs = [
        "convert_test.go",
        "env_test.go",
        "step_test.go",
        "volume_test.go",
    ],
    data = glob(["testdata/**"]),
    embed = [":go_default_library"],
    deps = [
        "//pkg/apis/build/v1alpha1:go_default_library",
        "//pkg/buildtest:go_default_library",
        "//vendor/google.golang.org/api/cloudbuild/v1:go_default_library",
        "//vendor/k8s.io/api/core/v1:go_default_library",
    ],
)
