package main

import (
	"context"
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"sync"

	"github.com/cloudevents/sdk-go/pkg/cloudevents"
	"github.com/cloudevents/sdk-go/pkg/cloudevents/client"
	v1alpha1 "github.com/knative/build/pkg/apis/build/v1alpha1"
	buildclientset "github.com/knative/build/pkg/client/clientset/versioned"
	clientset "github.com/knative/build/pkg/client/clientset/versioned"
	"github.com/knative/pkg/logging"
	"github.com/pkg/errors"
	gh "gopkg.in/go-playground/webhooks.v5/github"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	ReceiverPort             = 8088
	SupportCloudEventVersion = "0.2"
)

var (
	eventType  = flag.String("event-type", "com.github.checksuite", "The event type to listen for. Currently only supports com.github.checksuite")
	namespace  = flag.String("namespace", "default", "The namespace to create the build in.")
	buildPath  = flag.String("build-path", "/root/build.json", "The path to the build spec.")
	masterURL  = flag.String("master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	kubeconfig = flag.String("kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
)

type CloudEventListener struct {
	eventType      string
	branch         string
	repo           string
	namespace      string
	build          *v1alpha1.Build
	buildclientset clientset.Interface
	mux            *sync.Mutex
}

func main() {
	flag.Parse()
	logger, _ := logging.NewLogger("", "cloudevent-listener")
	defer logger.Sync()

	build, err := loadBuildSpec()
	if err != nil {
		log.Fatalf("Failed loading build spec from volume:", err)
	}

	cfg, err := clientcmd.BuildConfigFromFlags(*masterURL, *kubeconfig)
	if err != nil {
		logger.Fatalf("Error building kubeconfig: %v", err)
	}

	buildClient, err := buildclientset.NewForConfig(cfg)
	if err != nil {
		logger.Fatalf("Error building Build clientset: %v", err)
	}

	c := &CloudEventListener{
		eventType:      *eventType,
		build:          build,
		namespace:      *namespace,
		mux:            &sync.Mutex{},
		buildclientset: buildClient,
	}

	log.Print("Starting web server")

	client, err := client.NewDefault()
	if err != nil {
		log.Fatalf("Failed to create cloudevent client: %q", err)
	}

	log.Fatalf("Failed to start cloudevent receiver: %q", client.StartReceiver(context.Background(), c.HandleRequest))
}

// HandleRequest will decode the body into cloudevent, match on the event type and submit build from repo/branch.
// Only check suite events are supported by this proposal.
func (r *CloudEventListener) HandleRequest(ctx context.Context, event cloudevents.Event) error {
	// todo: contribute nil check upstream
	if event.Context == nil {
		return errors.New("Empty event context")
	}

	if event.SpecVersion() != "0.2" {
		return errors.New("Only cloudevents version 0.2 supported")
	}
	if event.Type() != r.eventType {
		return errors.New("Mismatched event type submitted")

	}
	ec, ok := event.Context.(cloudevents.EventContextV02)
	if !ok {
		return errors.New("Cloudevent context missing")
	}

	log.Printf("Handling event ID: %q Type: %q", ec.ID, ec.GetType())

	switch event.Type() {
	case "com.github.checksuite":
		cs := &gh.CheckSuitePayload{}
		if err := event.DataAs(cs); err != nil {
			return errors.Wrap(err, "Error handling check suite payload")
		}
		if err := r.handleCheckSuite(event, cs); err != nil {
			return err
		}
	}

	r.createBuild()

	return nil
}

func (r *CloudEventListener) handleCheckSuite(event cloudevents.Event, cs *gh.CheckSuitePayload) error {
	if cs.CheckSuite.Conclusion == "success" {
		build, err := r.createBuild()
		if err != nil {
			return errors.Wrapf(err, "Error creating build for check_suite event ID: %q", event.Context.AsV02().ID)
		}
		log.Printf("Created build %q!", build.Name)
	}
	return nil
}

func (r *CloudEventListener) createBuild() (*v1alpha1.Build, error) {
	r.mux.Lock()
	defer r.mux.Unlock()

	newBuild, err := r.buildclientset.BuildV1alpha1().Builds(r.build.Namespace).Create(r.build)
	if err != nil {
		return nil, err
	}
	return newBuild, nil
}

// Read in the build spec info we have prepared to handle builds!
func loadBuildSpec() (*v1alpha1.Build, error) {
	b := new(v1alpha1.Build)
	data, err := ioutil.ReadFile(*buildPath)
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal([]byte(data), &b); err != nil {
		return nil, err
	}
	return b, nil
}
