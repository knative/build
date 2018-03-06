## Credential Initializer

This tool sets up credentials for the builder images that run as part of
a `Build`.  That process is outlined in detail here.

### Building on K8s-native secret types

The following types are supported:
* `kubernetes.io/basic-auth`
* `kubernetes.io/ssh-auth`

These will be made available to the `Build` by attaching them to the
`ServiceAccount` as which it runs.

### Guiding credential integration.

Having one of these secret types in insufficient for turning it into a usable
secret.  e.g. basic auth (username / password) is usable with both Git and
Docker repositories, and I may have *N* Git secrets and *M* Docker secrets.

The answer to this (learning from Openshift's solution) is to guide it with
annotations on the `Secret` objects, which will take the form:
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

Or for SSH:
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


### Exposing the credential integration.

In their native form, these credentials are unsuitable for consumption by Git
and Docker.  For Git, they need to be turned into (some form of) `.gitconfig`.
For Docker, they need to be turned into a `~/.docker/config.json` file.  Also,
while each of these supports having multiple credentials for multiple domains,
those credentials typically need to be blended into a single canonical keyring.

To solve this, prior to even the `Source` step, we will run a credential
initialization process that accesses each of its secrets and aggregates them
into their respective files in `$HOME`.


### Docker `basic-auth`

Given URLs, usernames, and passwords of the form: `https://url{n}.com`,
`user{n}`, and `pass{n}`.  We will generate the following for Docker:
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
`user{n}`, and `pass{n}`.  We will generate the following for Git:
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
`key{n}`, and `known_hosts{n}`.  We will generate the following for Git:
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
mechanisms outlined here should not be used.  Instead the user should declare an
explicit `Volume` from the `Secret` and manually `VolumeMount` it into the
`Step`.
