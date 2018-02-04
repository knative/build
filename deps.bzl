load("@io_bazel_rules_docker//container:container.bzl", "container_pull")

def repositories():
  container_pull(
      name = "git_base",
      registry = "gcr.io",
      repository = "cloud-builders/git",
      tag = "latest",
  )
