# Cloud Events Listener

The cloud events listener proposal defines a process by which Knative Eventing can directly trigger Builds.

To do this, an optional CRD `CloudEventsListeners` is provided, which provides a method for Builds to act as an Eventing sink and accept CloudEvents. This makes it possible for any event to trigger any build, providing a lightweight pipeline for common tasks like Image Building.

Inside the CEL spec, a build is defined. Once applied, this new CRD will deploy a small listener which will listen for a specific cloud event from a specific source. Once that has been received, the listener populates the SHA we wish to build inside the custom Build that was provided in the Spec.

This build is then created by the listener in the namespace defined in the listener config, as any other build. The end result should likely be a new image in a registry, freshly built entirely by knative after CI went green.

The only event supported by this proposal is `com.github.checksuite` but the design should allow for handling nearly any event type.

This service could pretty easily be extended to allow it to be able to emit a CloudEvent of its own to whatever endpoint is defined.
