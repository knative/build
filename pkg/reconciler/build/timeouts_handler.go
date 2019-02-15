package build

import (
	"sync"
	"time"

	clientset "github.com/knative/build/pkg/client/clientset/versioned"
	"go.uber.org/zap"
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
		if isDone(&build.Status) {
			continue
		}
		r := Reconciler{
			buildclientset: t.buildclientset,
			kubeclientset:  t.kubeclientset,
			Logger:         t.logger,
		}
		if err = r.checkTimeout(&build); err != nil {
			t.logger.Errorf("timeout check error: %s", err)
		}
		if err = r.updateStatus(&build); err != nil {
			t.logger.Errorf("status update error: %s", err)
		}
	}
	return nil
}
