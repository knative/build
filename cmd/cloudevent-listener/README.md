# Cloud Events Listener

The cloud events listener proposal defines a process by which Knative Eventing can directly trigger Builds.

To do this, an optional CRD `CloudEventsListeners` is provided, which provides a method for Builds to act as an Eventing sink and accept CloudEvents. This makes it possible for any event to trigger any build, providing a lightweight pipeline for common tasks like Image Building.

Inside the CEL spec, a build is defined. Once applied, this new CRD will deploy a small listener which will listen for a specific cloud event from a specific source. Once that has been received, the listener populates the SHA we wish to build inside the custom Build that was provided in the Spec.

This build is then created by the listener in the namespace defined in the listener config, as any other build. The end result should likely be a new image in a registry, freshly built entirely by knative after CI went green.

The only event supported by this proposal is `com.github.checksuite` but the design should allow for handling nearly any event type.

This service could pretty easily be extended to allow it to be able to emit a CloudEvent of its own to whatever endpoint is defined.

# Minikube instructions

To get this bootstrapped locally:


* Get the `ko` command: `go get -u github.com/google/ko/cmd/ko`
* Load your docker enviroment vars: `eval $(minikube docker-env)`
* Start a registry: `docker run -it -d -p 5000:5000 registry:2`
* Set `KO_DOCKER_REPO` to local registry: `export KO_DOCKER_REPO=localhost:5000/<myproject>`
* Apply build components: `ko apply -L -f config/`
* Create a CloudEventsListener (such as the example below) and await cloud events.
* Listener is configured for port `8082`.


```
apiVersion: build.knative.dev/v1alpha1
kind: CloudEventsListener
metadata:
  name: build-cloud-events-listener
  namespace: knative-build
spec:
  selector:
    matchLabels:
      app: build-cloud-event-listener
  serviceName: build-cloud-event-listener
  template:
    metadata:
      labels:
        role: build-cloud-event-listener
    spec:
      serviceAccountName: build-controller
  listener-image: github.com/knative/build/cmd/cloudevent-listener
  cloud-event-type: com.github.checksuite
  branch: master
  namespace: knative-build
  service-account: build-controller
  build:
    name: cel-example-build
    metadata:
      name: cel-example-build
    namespace: knative-build
    spec:
      serviceAccountName: build-auth-example
      source:
        git:
          url: https://github.com/example/build-example.git
          revision: master
      steps:
      - name: ubuntu-example
        image: ubuntu
        args: ["ubuntu-build-example", "SECRETS-example.md"]
```
