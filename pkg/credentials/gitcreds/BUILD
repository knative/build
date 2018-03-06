load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = [
        "basic.go",
        "creds.go",
        "ssh.go",
    ],
    importpath = "github.com/elafros/build/pkg/credentials/gitcreds",
    visibility = ["//visibility:public"],
    deps = [
        "//pkg/credentials:go_default_library",
        "//vendor/k8s.io/api/core/v1:go_default_library",
    ],
)

go_test(
    name = "go_default_test",
    srcs = ["creds_test.go"],
    embed = [":go_default_library"],
    importpath = "github.com/elafros/build/pkg/credentials/gitcreds",
    deps = [
        "//pkg/credentials:go_default_library",
        "//vendor/k8s.io/api/core/v1:go_default_library",
    ],
)
