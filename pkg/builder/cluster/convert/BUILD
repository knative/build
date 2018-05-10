load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = ["convert.go"],
    importpath = "github.com/elafros/build/pkg/builder/cluster/convert",
    visibility = ["//visibility:public"],
    deps = [
        "//pkg/apis/build/v1alpha1:go_default_library",
        "//pkg/builder/validation:go_default_library",
        "//pkg/credentials:go_default_library",
        "//pkg/credentials/dockercreds:go_default_library",
        "//pkg/credentials/gitcreds:go_default_library",
        "//vendor/k8s.io/api/core/v1:go_default_library",
        "//vendor/k8s.io/apimachinery/pkg/apis/meta/v1:go_default_library",
        "//vendor/k8s.io/apimachinery/pkg/runtime/schema:go_default_library",
        "//vendor/k8s.io/client-go/kubernetes:go_default_library",
    ],
)

go_test(
    name = "go_default_test",
    srcs = ["convert_test.go"],
    data = glob(["testdata/**"]),
    embed = [":go_default_library"],
    deps = [
        "//pkg/apis/build/v1alpha1:go_default_library",
        "//pkg/buildtest:go_default_library",
        "//vendor/k8s.io/api/core/v1:go_default_library",
        "//vendor/k8s.io/apimachinery/pkg/apis/meta/v1:go_default_library",
        "//vendor/k8s.io/client-go/kubernetes/fake:go_default_library",
    ],
)
