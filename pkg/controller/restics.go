package controller

import (
	acrt "github.com/appscode/go/runtime"
	sapi "github.com/appscode/stash/api"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/cache"
)

// Blocks caller. Intended to be called as a Go routine.
func (c *Controller) WatchRestics() {
	defer acrt.HandleCrash()

	lw := &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return c.StashClient.Restics(apiv1.NamespaceAll).List(metav1.ListOptions{})
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return c.StashClient.Restics(apiv1.NamespaceAll).Watch(metav1.ListOptions{})
		},
	}
	_, ctrl := cache.NewInformer(lw,
		&sapi.Restic{},
		c.SyncPeriod,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				if resource, ok := obj.(*sapi.Restic); ok {
					c.EnsureSidecar(resource)
				}
			},
			DeleteFunc: func(obj interface{}) {
				if resource, ok := obj.(*sapi.Restic); ok {
					c.EnsureSidecarDeleted(resource)
				}
			},
		},
	)
	ctrl.Run(wait.NeverStop)
}

func (c *Controller) EnsureSidecar(restic *sapi.Restic) {
	if resources, err := c.KubeClient.CoreV1().ReplicationControllers(restic.Namespace).List(metav1.ListOptions{LabelSelector: restic.Spec.Selector.String()}); err == nil {
		for _, resource := range resources.Items {
			c.EnsureReplicationControllerSidecar(&resource, restic)
		}
	}

	if resources, err := c.KubeClient.ExtensionsV1beta1().ReplicaSets(restic.Namespace).List(metav1.ListOptions{LabelSelector: restic.Spec.Selector.String()}); err == nil {
		for _, resource := range resources.Items {
			c.EnsureReplicaSetSidecar(&resource, restic)
		}
	}

	if resources, err := c.KubeClient.ExtensionsV1beta1().Deployments(restic.Namespace).List(metav1.ListOptions{LabelSelector: restic.Spec.Selector.String()}); err == nil {
		for _, resource := range resources.Items {
			c.EnsureDeploymentExtensionSidecar(&resource, restic)
		}
	}

	if resources, err := c.KubeClient.AppsV1beta1().Deployments(restic.Namespace).List(metav1.ListOptions{LabelSelector: restic.Spec.Selector.String()}); err == nil {
		for _, resource := range resources.Items {
			c.EnsureDeploymentAppSidecar(&resource, restic)
		}
	}

	if resources, err := c.KubeClient.ExtensionsV1beta1().DaemonSets(restic.Namespace).List(metav1.ListOptions{LabelSelector: restic.Spec.Selector.String()}); err == nil {
		for _, resource := range resources.Items {
			c.EnsureDaemonSetSidecar(&resource, restic)
		}
	}
}

func (c *Controller) EnsureSidecarDeleted(restic *sapi.Restic) {
	if resources, err := c.KubeClient.CoreV1().ReplicationControllers(restic.Namespace).List(metav1.ListOptions{LabelSelector: restic.Spec.Selector.String()}); err == nil {
		for _, resource := range resources.Items {
			c.EnsureReplicationControllerSidecarDeleted(&resource, restic)
		}
	}

	if resources, err := c.KubeClient.ExtensionsV1beta1().ReplicaSets(restic.Namespace).List(metav1.ListOptions{LabelSelector: restic.Spec.Selector.String()}); err == nil {
		for _, resource := range resources.Items {
			c.EnsureReplicaSetSidecarDeleted(&resource, restic)
		}
	}

	if resources, err := c.KubeClient.ExtensionsV1beta1().Deployments(restic.Namespace).List(metav1.ListOptions{LabelSelector: restic.Spec.Selector.String()}); err == nil {
		for _, resource := range resources.Items {
			c.EnsureDeploymentExtensionSidecarDeleted(&resource, restic)
		}
	}

	if resources, err := c.KubeClient.AppsV1beta1().Deployments(restic.Namespace).List(metav1.ListOptions{LabelSelector: restic.Spec.Selector.String()}); err == nil {
		for _, resource := range resources.Items {
			c.EnsureDeploymentAppSidecarDeleted(&resource, restic)
		}
	}

	if resources, err := c.KubeClient.ExtensionsV1beta1().DaemonSets(restic.Namespace).List(metav1.ListOptions{LabelSelector: restic.Spec.Selector.String()}); err == nil {
		for _, resource := range resources.Items {
			c.EnsureDaemonSetSidecarDeleted(&resource, restic)
		}
	}
}