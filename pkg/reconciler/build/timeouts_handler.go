package build

import (
	"fmt"
	"sync"
	"time"

	"github.com/knative/build/pkg/apis/build/v1alpha1"
	clientset "github.com/knative/build/pkg/client/clientset/versioned"
	duckv1alpha1 "github.com/knative/pkg/apis/duck/v1alpha1"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const checkInterval = 10 * time.Second

// TimeoutSet contains required objects to handle build timeouts
type TimeoutSet struct {
	logger         *zap.SugaredLogger
	kubeclientset  kubernetes.Interface
	buildclientset clientset.Interface
	stopCh         <-chan struct{}
	buildStatus    *sync.Mutex
}

// NewTimeoutHandler returns TimeoutSet filled structure
func NewTimeoutHandler(logger *zap.SugaredLogger,
	kubeclientset kubernetes.Interface,
	buildclientset clientset.Interface,
	stopCh <-chan struct{},
	buildStatus *sync.Mutex) *TimeoutSet {
	return &TimeoutSet{
		logger:         logger,
		kubeclientset:  kubeclientset,
		buildclientset: buildclientset,
		stopCh:         stopCh,
		buildStatus:    buildStatus,
	}
}

// TimeoutHandler walks through all namespaces, gets list of builds and checks timeouts
func (t *TimeoutSet) TimeoutHandler() error {
	t.buildStatus.Lock()
	defer t.buildStatus.Unlock()

	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-t.stopCh:
			return nil
		case <-ticker.C:
			namespaces, err := t.kubeclientset.CoreV1().Namespaces().List(metav1.ListOptions{})
			if err != nil {
				t.logger.Errorf("%s", err)
				continue
			}
			for _, namespace := range namespaces.Items {
				if err := t.checkPodTimeouts(namespace.GetName()); err != nil {
					t.logger.Errorf("%s", err)
				}
			}
		}
	}
}

func (t *TimeoutSet) checkPodTimeouts(namespace string) error {
	builds, err := t.buildclientset.BuildV1alpha1().Builds(namespace).List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, build := range builds.Items {
		cond := build.Status.GetCondition(v1alpha1.BuildSucceeded)
		// Check pod timeout only for ongoing build
		if cond == nil || cond.Status != corev1.ConditionUnknown {
			continue
		}
		// Build doesn't have pod yet
		if build.Status.StartTime.IsZero() {
			continue
		}
		timeout := defaultTimeout
		if build.Spec.Timeout != nil {
			timeout = build.Spec.Timeout.Duration
		}
		runtime := time.Since(build.Status.StartTime.Time)
		// Build timeout is not exceeded
		if runtime < timeout {
			continue
		}
		if build.Status.Cluster == nil {
			continue
		}
		t.logger.Infof("Build %s timeout (runtime %s over %s timeout)", build.Name, runtime, timeout)
		if err := t.deletePodAndUpdateStatus(build.Status.Cluster.PodName, &build); err != nil {
			t.logger.Error(err)
		}
	}
	return nil
}

func (t *TimeoutSet) deletePodAndUpdateStatus(pod string, build *v1alpha1.Build) error {
	build.Status.SetCondition(&duckv1alpha1.Condition{
		Type:    v1alpha1.BuildSucceeded,
		Status:  corev1.ConditionFalse,
		Reason:  "BuildTimeout",
		Message: fmt.Sprintf("Build %q timeout", build.Name),
	})
	build.Status.CompletionTime = &metav1.Time{
		Time: time.Now(),
	}
	if _, err := t.buildclientset.BuildV1alpha1().Builds(build.Namespace).UpdateStatus(build); err != nil {
		return fmt.Errorf("Failed to update build status: %v", err)
	}
	if err := t.kubeclientset.CoreV1().Pods(build.Namespace).Delete(pod, &metav1.DeleteOptions{}); err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("Failed to terminate pod: %v", err)
	}
	return nil
}
