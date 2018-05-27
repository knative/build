# Authentication

This document serves to define how authentication is provided during execution
of a build.

Two basic types of authentication are supported, using the Kubernetes-native
types:

* `kubernetes.io/basic-auth`
* `kubernetes.io/ssh-auth`

`Secret`s of these types can be made available to the `Build` by attaching them
to the `ServiceAccount` as which it runs.

### Exposing credentials to the build

In their native form, these secrets are unsuitable for consumption by Git and
Docker. For Git, they need to be turned into (some form of) `.gitconfig`. For
Docker, they need to be turned into a `~/.docker/config.json` file. Also,
while each of these supports having multiple credentials for multiple domains,
those credentials typically need to be blended into a single canonical keyring.

To solve this, prior to even the `Source` step, all builds execute a credential
initialization process that accesses each of its secrets and aggregates them
into their respective files in `$HOME`.

## Git SSH authentication

First, define a `Secret` containing your SSH private key.

```yaml
metadata:
  name: ssh-key
type: kubernetes.io/ssh-auth
data:
  ssh-privatekey: <base64 encoded>
  # This is non-standard, but its use is encouraged to make this more secure.
  known_hosts: <base64 encoded>
```

To generate the value of `ssh-privatekey`, copy the value of (for example) `cat id_rsa | base64`.

Then copy the value of `cat ~/.ssh/known_hosts | base64` to the `known_hosts` field.

Next, give access to this `Secret` to a `ServiceAccount`:

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: build-bot
secrets:
- name: ssh-key
```

Then use that `ServiceAccount` in your `Build`:

```yaml
apiVersion: build.dev/v1alpha1
kind: Build
metadata:
  name: build-with-ssh-auth
spec:
  serviceAccountName: build-bot
  steps:
  ...
```

When this build executes, a `~/.gitconfig` will be generated at the beginning
of the build that uses the specified SSH key to authenticate remote Git
operations.

## Docker Basic Authentication

First, define a `Secret` containing the base64-encoded username and password
the build should use to authenticate to a Docker registry.

```yaml
metadata:
  name: basic-user-pass
type: kubernetes.io/basic-auth
data:
  username: <base64 encoded>
  password: <base64 encoded>
```

Next, give access to this `Secret` to a `ServiceAccount`:

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: build-bot
secrets:
- name: basic-user-pass
```

Then use that `ServiceAccount` in your `Build`:

```yaml
apiVersion: build.dev/v1alpha1
kind: Build
metadata:
  name: build-with-basic-auth
spec:
  serviceAccountName: build-bot
  steps:
  ...
```

When this build executes, a `~/.docker/config.json` will be generated at the
beginning of the build that uses the specified username and password to
authenticate remote Docker registry operations.

This method can also be used to pass basic username/password credentials to a
Git repository.

### Guiding credential selection

A build may require many different types of authentication. For instance, a
build might require access to multiple private Git repositories, and access to
many private Docker repositories. You can use annotations to guide which secret
to use to authenticate to different resources, for example:

```yaml
metadata:
  annotations:
    build.dev/git-0: https://github.com
    build.dev/git-1: https://gitlab.com
    build.dev/docker-0: https://gcr.io
type: kubernetes.io/basic-auth
data:
  username: <base64 encoded>
  password: <base64 encoded>
```

This describes a "Basic Auth" (username and password) secret which should be
used to access Git repos at github.com and gitlab.com, as well as Docker
repositories at gcr.io.

Similarly, for SSH:

```yaml
metadata:
  annotations:
    build.dev/git-0: github.com
type: kubernetes.io/ssh-auth
data:
  ssh-privatekey: <base64 encoded>
  # This is non-standard, but its use is encouraged to make this more secure.
  # Omitting this results in the use of ssh-keyscan (see below).
  known_hosts: <base64 encoded>
```

This describes an SSH key secret which should be used to access Git repos at
github.com only.

## Implementation Detail

### Docker `basic-auth`

Given URLs, usernames, and passwords of the form: `https://url{n}.com`,
`user{n}`, and `pass{n}`. We will generate the following for Docker:

```json
=== ~/.docker/config.json ===
{
  "auths": {
    "https://url1.com": {
      "auth": "$(echo -n user1:pass1 | base64)",
      "email": "not@val.id",
    },
    "https://url2.com": {
      "auth": "$(echo -n user2:pass2 | base64)",
      "email": "not@val.id",
    },
    ...
  }
}
```

Docker doesn't support `kubernetes.io/ssh-auth`, so annotations on these types
will be ignored.

### Git `basic-auth`

Given URLs, usernames, and passwords of the form: `https://url{n}.com`,
`user{n}`, and `pass{n}`. We will generate the following for Git:
```
=== ~/.gitconfig ===
[credential]
	helper = store
[credential "https://url1.com"]
    username = "user1"
[credential "https://url2.com"]
    username = "user2"
...
=== ~/.git-credentials ===
https://user1:pass1@url1.com
https://user2:pass2@url2.com
...
```

### Git `ssh-auth`

Given hostnames, private keys, and `known_hosts` of the form: `url{n}.com`,
`key{n}`, and `known_hosts{n}`. We will generate the following for Git:
```
=== ~/.ssh/id_key1 ===
{contents of key1}
=== ~/.ssh/id_key2 ===
{contents of key2}
...
=== ~/.ssh/config ===
Host url1.com
    HostName url1.com
    IdentityFile ~/.ssh/id_key1
Host url2.com
    HostName url2.com
    IdentityFile ~/.ssh/id_key2
...
=== ~/.ssh/known_hosts ===
{contents of known_hosts1}
{contents of known_hosts2}
...
```

NOTE: Since `known_hosts` is a non-standard extension of
`kubernetes.io/ssh-auth`, when it is not present this will be generated via
`ssh-keygen url{n}.com ` instead.

### Least Privilege

The secrets as outlined here will be stored into `$HOME` (by convention the
volume: `/builder/home`), and will be available to `Source` and all `Steps`.

For sensitive credentials that should not be made available to some steps, the
mechanisms outlined here should not be used. Instead the user should declare an
explicit `Volume` from the `Secret` and manually `VolumeMount` it into the
`Step`.
