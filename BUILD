package(default_visibility = ["//visibility:public"])

load("@io_bazel_rules_go//go:def.bzl", "gazelle", "go_prefix")

go_prefix("github.com/knative/build")

gazelle(
    name = "gazelle",
    external = "vendored",
)

load("@io_bazel_rules_k8s//k8s:objects.bzl", "k8s_objects")

# This exists as a legacy alias, but folks should use //config:everything.
k8s_objects(
    name = "everything",
    objects = [
        "//config:everything",
    ],
)
