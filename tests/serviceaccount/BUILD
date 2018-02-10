package(default_visibility = ["//visibility:public"])

load("@k8s_object//:defaults.bzl", "k8s_object")

k8s_object(
    name = "secret",
    template = "secret.yaml",
)

k8s_object(
    name = "serviceaccount",
    template = "serviceaccount.yaml",
)

k8s_object(
    name = "build",
    template = "build.yaml",
)

load("@io_bazel_rules_k8s//k8s:objects.bzl", "k8s_objects")

k8s_objects(
    name = "test",
    objects = [
        ":secret",
        ":serviceaccount",
        ":build",
    ],
)
