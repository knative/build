workspace(name = "buildcrd")

# Load docker rules
git_repository(
    name = "io_bazel_rules_docker",
    remote = "https://github.com/bazelbuild/rules_docker.git",
    tag = "v0.5.1",
)

load(
    "@io_bazel_rules_docker//container:container.bzl",
    "container_pull",
    container_repositories = "repositories",
)

container_repositories()

container_pull(
    name = "base",
    registry = "gcr.io",
    repository = "distroless/base",
    # "gcr.io/distroless/base:latest" circa 2018-10-16 21:01 -0400
    digest = "sha256:472206d4c501691d9e72cafca4362f2adbc610fecff3dfa42e5b345f9b7d05e5"
)

container_pull(
    name = "base-debug",
    registry = "gcr.io",
    repository = "distroless/base",
    # "gcr.io/distroless/base:debug" circa 2018-10-16 21:01 -0400
    digest = "sha256:bb7b331d3132e95c48556dbd3b28079f0eb3014f2726f5ddd7225b9c9df16a91",
)

# Load distroless rules
git_repository(
    name = "distroless",
    remote = "https://github.com/GoogleContainerTools/distroless",
    commit = "f93e9b5c88a0ac6b98bc4ff94d206702cfb4e7e3",
)

load(
    "@distroless//package_manager:package_manager.bzl",
    "dpkg_list",
    "dpkg_src",
    "package_manager_repositories",
)

package_manager_repositories()

dpkg_src(
    name = "debian_stretch",
    arch = "amd64",
    distro = "stretch",
    sha256 = "9e7870c3c3b5b0a7f8322c323a3fa641193b1eee792ee7e2eedb6eeebf9969f3",
    snapshot = "20180719T151130Z",
    url = "https://snapshot.debian.org/archive",
)

dpkg_src(
    name = "debian_stretch_backports",
    arch = "amd64",
    distro = "stretch-backports",
    sha256 = "29524787f58bc4e139e30e66bc476ff1ea33c0aa939d11638626dbe07c64b30d",
    snapshot = "20180919T095426Z",
    url = "http://snapshot.debian.org/archive",
)

dpkg_src(
    name = "debian_stretch_security",
    package_prefix = "https://snapshot.debian.org/archive/debian-security/20180919T095426Z/",
    packages_gz_url = "https://snapshot.debian.org/archive/debian-security/20180919T095426Z/dists/stretch/updates/main/binary-amd64/Packages.gz",
    sha256 = "4b7df485333ed77ccc9bb4fea9bffe302dc4d0e2303f27b39dc6a4cfcfe5fca5",
)

dpkg_list(
    name = "package_bundle",
    packages = [
      # https://packages.debian.org/stretch/openssh-client
      "openssh-client",
        # openssh-clients deps
        "libedit2",
          # libedit2 deps
          "libncurses5",
          "libbsd0",
          "libtinfo5",
        "libgssapi-krb5-2",
          # libgssapi-krb5-2 deps
          "libcomerr2",
          "libcom-err2", # This is pulled due to backport selection
          "libk5crypto3",
          "libkeyutils1",
          "libkrb5-3",
          "libkrb5support0",
        "libselinux1",
          # libselinux1 deps
          # "libpcre3",  #provided by git below
        "libssl1.0.2",
      # "zlib1g", # provided by git below

      # https://packages.debian.org/stretch/git
      "git",
      # "libc6" # provided by distroless/base
      "libcurl3-gnutls",
      "liberror-perl",
      "libexpat1",
      "libpcre3",
      "perl",
      "zlib1g",
    ],
    # Takes the first package found: security updates should go first
    # If there was a security fix to a package before the stable release, this will find
    # the older security release. This happened for stretch libc6.
    sources = [
        "@debian_stretch_security//file:Packages.json",
        "@debian_stretch_backports//file:Packages.json",
        "@debian_stretch//file:Packages.json",
    ],
)
