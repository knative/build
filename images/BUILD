package(default_visibility = ["//visibility:public"])

load(
    "@package_bundle//file:packages.bzl",
    "packages",
)

load(
    "@io_bazel_rules_docker//container:container.bzl",
    "container_image",
    "container_push",
)

deb_packages = [
    # See WORKSPACE for the dependency tree
    # and links to the debian packages online
    packages["openssh-client"],
    packages["libedit2"],
    packages["libgssapi-krb5-2"],
    packages["libselinux1"],
    packages["libssl1.0.2"],
    # libedit2 deps
    packages["libncurses5"],
    packages["libbsd0"],
    packages["libtinfo5"],
    # libgssapi-krb5-2 deps
    packages["libcomerr2"],
    packages["libcom-err2"],
    packages["libk5crypto3"],
    packages["libkeyutils1"],
    packages["libkrb5-3"],
    packages["libkrb5support0"],

    packages["git"],
    packages["libcurl3-gnutls"],
    packages["liberror-perl"],
    packages["libexpat1"],
    packages["libpcre3"],
    packages["perl"],
    packages["zlib1g"],
]

container_image(
    name = "build-base",
    base = "@base//image",
    debs = deb_packages,
)

container_image(
    name = "build-base-debug",
    base = "@base-debug//image",
    debs = deb_packages,
)

container_push(
   name = "push-build-base",
   image = ":build-base",
   format = "Docker",
   registry = "$(registry)",
   repository = "$(repository)",
   tag = "latest",
)

container_push(
   name = "push-build-base-debug",
   image = ":build-base-debug",
   format = "Docker",
   registry = "$(registry)",
   repository = "$(repository)",
   tag = "debug",
)

