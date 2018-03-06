package(default_visibility = ["//visibility:public"])

load("@io_bazel_rules_go//go:def.bzl", "gazelle", "go_binary", "go_library")

go_library(
    name = "go_default_library",
    srcs = ["main.go"],
    importpath = "github.com/elafros/build/cmd/controller",
    visibility = ["//visibility:private"],
    deps = [
        "//pkg/builder:go_default_library",
        "//pkg/builder/cluster:go_default_library",
        "//pkg/builder/google:go_default_library",
        "//pkg/client/clientset/versioned:go_default_library",
        "//pkg/client/informers/externalversions:go_default_library",
        "//pkg/controller:go_default_library",
        "//pkg/controller/build:go_default_library",
        "//pkg/controller/buildtemplate:go_default_library",
        "//pkg/signals:go_default_library",
        "//vendor/cloud.google.com/go/compute/metadata:go_default_library",
        "//vendor/github.com/golang/glog:go_default_library",
        "//vendor/golang.org/x/oauth2:go_default_library",
        "//vendor/golang.org/x/oauth2/google:go_default_library",
        "//vendor/google.golang.org/api/cloudbuild/v1:go_default_library",
        "//vendor/k8s.io/client-go/informers:go_default_library",
        "//vendor/k8s.io/client-go/kubernetes:go_default_library",
        "//vendor/k8s.io/client-go/tools/clientcmd:go_default_library",
    ],
)

go_binary(
    name = "controller",
    embed = [":go_default_library"],
    importpath = "github.com/elafros/build/cmd/controller",
    pure = "on",
    visibility = ["//visibility:public"],
)

load("@io_bazel_rules_docker//go:image.bzl", "go_image")

go_image(
    name = "image",
    binary = ":controller",
)
