package build

import (
	"fmt"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/watch"

	"github.com/knative/build/pkg/apis/build/v1alpha1"
	clientset "github.com/knative/build/pkg/client/clientset/versioned"
	duckv1alpha1 "github.com/knative/pkg/apis/duck/v1alpha1"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type TimeoutSet struct {
	logger         *zap.SugaredLogger
	kubeclientset  kubernetes.Interface
	buildclientset clientset.Interface
	stopCh         <-chan struct{}
	buildStatus    *sync.Mutex
}

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

func (t *TimeoutSet) TimeoutHandler() error {
	namespaces, err := t.kubeclientset.CoreV1().Namespaces().List(metav1.ListOptions{})
	if err != nil {
		t.logger.Errorf("%s", err)
	}
	for _, namespace := range namespaces.Items {
		namespace := namespace
		if err := t.checkPodTimeouts(namespace.GetName()); err != nil {
			t.logger.Errorf("%s", err)
		}
		go func() {
			if err := t.watchBuildEvents(namespace.GetName()); err != nil {
				t.logger.Errorf("%s", err)
			}
		}()
	}
	// Watching for builds in new namespaces
	w, err := t.kubeclientset.CoreV1().Namespaces().Watch(metav1.ListOptions{})
	if err != nil {
		t.logger.Errorf("%s", err)
	}
	for {
		select {
		case <-t.stopCh:
			w.Stop()
			return nil
		case event := <-w.ResultChan():
			namespace, ok := event.Object.(*corev1.Namespace)
			if !ok {
				return nil
			}
			if event.Type != watch.Added {
				continue
			}
			go func() {
				if err := t.watchBuildEvents(namespace.GetName()); err != nil {
					t.logger.Errorf("%s", err)
				}
			}()
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
		if err := t.deletePodAndUpdateStatus(build.Status.Cluster.PodName, &build); err != nil {
			t.logger.Error(err)
		}
	}
	return nil
}

func (t *TimeoutSet) watchBuildEvents(namespace string) error {
	buildEvents, err := t.buildclientset.BuildV1alpha1().Builds(namespace).Watch(metav1.ListOptions{})
	if err != nil {
		return err
	}
	for {
		select {
		case <-t.stopCh:
			buildEvents.Stop()
			return nil
		case event := <-buildEvents.ResultChan():
			build, ok := event.Object.(*v1alpha1.Build)
			if !ok {
				return nil
			}
			// Watch only for new builds
			if event.Type != watch.Added {
				continue
			}
			// New build doesn't have pod yet
			for build.Status.StartTime.IsZero() {
				if build, err = t.buildclientset.BuildV1alpha1().Builds(namespace).Get(build.Name, metav1.GetOptions{}); err != nil {
					t.logger.Error(err)
					continue
				}
				// Build started and ended immediately?
				if !build.Status.CompletionTime.IsZero() {
					continue
				}
				time.Sleep(1 * time.Second)
			}
			timeout := defaultTimeout
			if build.Spec.Timeout != nil {
				timeout = build.Spec.Timeout.Duration
			}
			select {
			case <-t.stopCh:
				return nil
			// wait for timeout
			case <-time.After(timeout):
				// Build pod... disappeared?
				if build.Status.Cluster == nil || len(build.Status.Cluster.PodName) == 0 {
					continue
				}
				if err := t.deletePodAndUpdateStatus(build.Status.Cluster.PodName, build); err != nil {
					t.logger.Error(err)
				}
			}
		}
	}
}

func (t *TimeoutSet) deletePodAndUpdateStatus(pod string, build *v1alpha1.Build) error {
	t.buildStatus.Lock()
	defer t.buildStatus.Unlock()

	// Not to update modified object
	build, err := t.buildclientset.BuildV1alpha1().Builds(build.Namespace).Get(build.Name, metav1.GetOptions{})
	if err != nil {
		t.logger.Error(err)
		return fmt.Errorf("Failed to get build status: %v", err)
	}

	cond := build.Status.GetCondition(v1alpha1.BuildSucceeded)
	//Update status only for ongoing build
	if cond == nil || cond.Status != corev1.ConditionUnknown {
		return nil
	}

	message := fmt.Sprintf("Build %q is timeout", build.Name)
	t.logger.Info(message)
	build.Status.SetCondition(&duckv1alpha1.Condition{
		Type:    v1alpha1.BuildSucceeded,
		Status:  corev1.ConditionFalse,
		Reason:  "BuildTimeout",
		Message: message,
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
