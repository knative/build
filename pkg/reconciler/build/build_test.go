package build

import (
	"context"
	"testing"
	"time"

	informers "github.com/knative/build/pkg/client/informers/externalversions"
	logtesting "github.com/knative/pkg/logging/testing"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	fakeclientset "k8s.io/client-go/kubernetes/fake"
	clientgotesting "k8s.io/client-go/testing"

	"github.com/knative/build/pkg/apis/build/v1alpha1"
	fakebuildclientset "github.com/knative/build/pkg/client/clientset/versioned/fake"
)

const namespace = "foo"

func TestReconcile(t *testing.T) {
	for _, c := range []struct {
		desc    string
		key     string
		wantErr bool

		// State of the world pre-reconciliation.
		kubeObjects  []runtime.Object
		buildObjects []runtime.Object

		// Behavior changes we'll inject.
		withReactors []clientgotesting.ReactionFunc

		// Things we expect to see happen during reconciliation.
		wantCreates []clientgotesting.CreateActionImpl
		wantUpdates []clientgotesting.UpdateActionImpl
	}{{
		desc: "key not found",
		key:  "foo/not-found",
	}, {
		desc: "no updates",
		key:  "foo/found",
		buildObjects: []runtime.Object{
			&v1alpha1.Build{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "found",
					Namespace: namespace,
				},
			},
		},
		// TODO: Received invalid build (past webhook)
	}} {
		t.Run(c.desc, func(t *testing.T) {
			kubeClient := fakeclientset.NewSimpleClientset(c.kubeObjects...)
			buildClient := fakebuildclientset.NewSimpleClientset(c.buildObjects...)

			for _, r := range c.withReactors {
				kubeClient.PrependReactor("*", "*", r)
				buildClient.PrependReactor("*", "*", r)
			}

			logger := logtesting.TestLogger(t)
			buildsLister := informers.NewSharedInformerFactory(buildClient, time.Second*30).Build().V1alpha1().Builds()
			r := NewController(
				logger,
				kubeClient,
				buildClient,
				buildsLister,
			).Reconciler
			if err := r.Reconcile(context.Background(), c.key); (err != nil) != c.wantErr {
				t.Errorf("Reconcile: unexpected error %v", err)
			}

			allActions := append(buildClient.Actions(), kubeClient.Actions()...)
			var gotCreates []clientgotesting.CreateAction
			var gotUpdates []clientgotesting.UpdateAction
			for _, a := range allActions {
				switch a.GetVerb() {
				case "create":
					gotCreates = append(gotCreates, a.(clientgotesting.CreateAction))
				case "update":
					gotUpdates = append(gotUpdates, a.(clientgotesting.UpdateAction))
				}
				logger.Infof("Got action: %v", a)
			}
		})
	}

}
