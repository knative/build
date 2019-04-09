package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"sync"

	"github.com/cloudevents/sdk-go/pkg/cloudevents"
	"github.com/cloudevents/sdk-go/pkg/cloudevents/client"
	"github.com/cloudevents/sdk-go/pkg/cloudevents/transport/http"
	"github.com/joeshaw/envdecode"
	v1alpha1 "github.com/knative/build/pkg/apis/build/v1alpha1"
	buildclientset "github.com/knative/build/pkg/client/clientset/versioned"
	clientset "github.com/knative/build/pkg/client/clientset/versioned"
	"github.com/knative/pkg/logging"
	"github.com/pkg/errors"
	gh "gopkg.in/go-playground/webhooks.v5/github"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	SupportCloudEventVersion = "0.2"
	listenerPort             = 8082
	listenerPath             = "/events"
)

type Config struct {
	EventType  string `env:"EVENT_TYPE,default=com.github.checksuite"`
	BuildPath  string `env:"BUILD_PATH,default=/root/builddata/build.json"`
	Branch     string `env:"BRANCH,default=master"`
	MasterURL  string `env:"MASTER_URL"`
	Kubeconfig string `env:"KUBECONFIG"`
	Namespace  string `env:"NAMESPACE"`
}

// CloudEventListener boots cloudevent receiver and awaits a particular event to build
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
	var cfg Config
	err := envdecode.Decode(&cfg)
	if err != nil {
		log.Fatalf("Failed loading env config: %q", err)
	}

	logger, _ := logging.NewLogger("", "cloudevent-listener")
	defer logger.Sync()

	if cfg.Namespace == "" {
		log.Fatal("NAMESPACE env var can not be empty")
	}

	// Load the build spec from the provided secret.
	build, err := loadBuildSpec(cfg.BuildPath)
	if err != nil {
		log.Fatalf("Failed loading build spec from volume: %s", err)
	}

	clientcfg, err := clientcmd.BuildConfigFromFlags(cfg.MasterURL, cfg.Kubeconfig)
	if err != nil {
		logger.Fatalf("Error building kubeconfig: %v", err)
	}

	buildClient, err := buildclientset.NewForConfig(clientcfg)
	if err != nil {
		logger.Fatalf("Error building Build clientset: %v", err)
	}

	c := &CloudEventListener{
		eventType:      cfg.EventType,
		build:          build,
		namespace:      cfg.Namespace,
		mux:            &sync.Mutex{},
		buildclientset: buildClient,
		branch:         cfg.Branch,
	}

	log.Printf("Starting listener on port %d", listenerPort)

	t, err := http.New(
		http.WithPort(listenerPort),
		http.WithPath(listenerPath),
	)
	if err != nil {
		log.Fatalf("failed to create http client, %v", err)
	}
	client, err := client.New(t, client.WithTimeNow(), client.WithUUIDs())
	if err != nil {
		log.Fatalf("failed to create client, %v", err)
	}

	log.Fatalf("Failed to start cloudevent receiver: %q", client.StartReceiver(context.Background(), c.HandleRequest))
}

// HandleRequest will decode the body of the cloudevent into the correct payload type based on event type,
// match on the event type and submit build from repo/branch.
// Only check_suite events are supported by this proposal.
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

	return nil
}

func (r *CloudEventListener) handleCheckSuite(event cloudevents.Event, cs *gh.CheckSuitePayload) error {
	if cs.CheckSuite.Conclusion == "success" {
		if cs.CheckSuite.HeadBranch != r.branch {
			return fmt.Errorf("Mismatched branches. Expected %s Received %s", cs.CheckSuite.HeadBranch, r.branch)

		}

		build, err := r.createBuild(cs.CheckSuite.HeadSHA)
		if err != nil {
			return errors.Wrapf(err, "Error creating build for check_suite event ID: %q", event.Context.AsV02().ID)
		}

		log.Printf("Created build %q!", build.Name)
	}
	return nil
}

func (r *CloudEventListener) createBuild(sha string) (*v1alpha1.Build, error) {
	r.mux.Lock()
	defer r.mux.Unlock()

	build := r.build.DeepCopy()
	// Set the builds git revision to the github events SHA
	build.Spec.Source.Git.Revision = sha
	// Set namespace from config. If they dont match, create will fail.
	build.Namespace = r.namespace

	log.Printf("Creating build %q sha %q namespace %q", build.Name, sha, build.Namespace)

	newBuild, err := r.buildclientset.BuildV1alpha1().Builds(build.Namespace).Create(build)
	if err != nil {
		return nil, err
	}
	return newBuild, nil
}

// Read in the build spec info we have prepared to handle builds!
func loadBuildSpec(path string) (*v1alpha1.Build, error) {
	b := new(v1alpha1.Build)
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal([]byte(data), &b); err != nil {
		return nil, err
	}
	return b, nil
}
