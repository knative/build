package(default_visibility = ["//visibility:public"])

load("@io_bazel_rules_go//go:def.bzl", "go_binary", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = ["main.go"],
    importpath = "github.com/elafros/build/cmd/gcs-fetcher",
    visibility = ["//visibility:private"],
    deps = [
        "//vendor/cloud.google.com/go/storage:go_default_library",
        "//vendor/github.com/golang/glog:go_default_library",
        "//vendor/golang.org/x/oauth2:go_default_library",
        "//vendor/golang.org/x/oauth2/google:go_default_library",
        "//vendor/google.golang.org/api/option:go_default_library",
    ],
)

go_binary(
    name = "gcs-fetcher",
    embed = [":go_default_library"],
    importpath = "github.com/elafros/build/cmd/gcs-fetcher",
    pure = "on",
    visibility = ["//visibility:public"],
)

load("@io_bazel_rules_docker//go:image.bzl", "go_image")

go_image(
    name = "image",
    binary = ":gcs-fetcher",
)

go_test(
    name = "go_default_test",
    srcs = ["fetcher_test.go"],
    embed = [":go_default_library"],
    importpath = "github.com/elafros/build/cmd/gcs-fetcher",
)
