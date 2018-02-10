load("@io_bazel_rules_k8s//k8s:objects.bzl", "k8s_objects")

k8s_objects(
    name = "all_tests",
    objects = [
        # Expected to complete
        "//tests/home:test",
        "//tests/custom-env:test",
        "//tests/default-workingdir:test",
        "//tests/workingdir:test",
        "//tests/workspace:test",
        "//tests/home-volume:test",
        "//tests/custom-volume:test",
        "//tests/template-volume:test",
        "//tests/custom-source:test",
        "//tests/template-args:test",
        "//tests/unnamed-steps:test",
        "//tests/docker-build:test",

        # Expected to be invalid.
        "//tests/template-duplicate-volume:test",

        # Expected to fail.
        "//tests/fail:test",
    ],
)

k8s_objects(
    name = "cluster_tests",
    objects = [
        ":all_tests",
        "//tests/configmap:test",
        "//tests/git-volume:test",
        "//tests/secret-volume:test",
        "//tests/secret-env:test",
        "//tests/serviceaccount:test",
        # This works on GCB too, but it's harder to test with private repositories.
        "//tests/git-source:test",
    ],
)
