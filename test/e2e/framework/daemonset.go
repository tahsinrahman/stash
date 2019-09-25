package framework

import (
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
	kutil "kmodules.xyz/client-go"
	"stash.appscode.dev/stash/pkg/util"

	"github.com/appscode/go/crypto/rand"
	. "github.com/onsi/gomega"
	apps "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func (fi *Invocation) DaemonSet(pvcName string) apps.DaemonSet {
	labels := map[string]string{
		"app":  fi.app,
		"kind": "daemonset",
	}
	daemon := apps.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix("stash"),
			Namespace: fi.namespace,
			Labels:    labels,
		},
		Spec: apps.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: fi.PodTemplate(labels, pvcName),
			UpdateStrategy: apps.DaemonSetUpdateStrategy{
				RollingUpdate: &apps.RollingUpdateDaemonSet{MaxUnavailable: &intstr.IntOrString{IntVal: 1}},
			},
		},
	}
	return daemon
}

func (f *Framework) CreateDaemonSet(obj apps.DaemonSet) (*apps.DaemonSet, error) {
	return f.KubeClient.AppsV1().DaemonSets(obj.Namespace).Create(&obj)
}

func (f *Framework) DeleteDaemonSet(meta metav1.ObjectMeta) error {
	return f.KubeClient.AppsV1().DaemonSets(meta.Namespace).Delete(meta.Name, deleteInBackground())
}

func (f *Framework) EventuallyDaemonSet(meta metav1.ObjectMeta) GomegaAsyncAssertion {
	return Eventually(func() *apps.DaemonSet {
		obj, err := f.KubeClient.AppsV1().DaemonSets(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		return obj
	})
}

func (f *Framework) EventuallyPodAccessible(meta metav1.ObjectMeta) GomegaAsyncAssertion {
	return Eventually(func() bool {
		labelSelector := fields.SelectorFromSet(meta.Labels)
		podList, err := f.KubeClient.CoreV1().Pods(meta.Namespace).List(metav1.ListOptions{LabelSelector: labelSelector.String()})
		Expect(err).NotTo(HaveOccurred())

		for _, pod := range podList.Items {
			_, err := f.ExecOnPod(&pod, "ls", "-R")
			if err == nil {
				return true
			}
		}
		return false
	},
		time.Minute*2,
		time.Second*2,
	)

}

func (f *Invocation) WaitUntilDaemonSetReadyWithSidecar(meta metav1.ObjectMeta) error {
	return wait.PollImmediate(kutil.RetryInterval, kutil.ReadinessTimeout, func() (bool, error) {
		if obj, err := f.KubeClient.AppsV1().DaemonSets(meta.Namespace).Get(meta.Name, metav1.GetOptions{}); err == nil {
			if obj.Status.DesiredNumberScheduled == obj.Status.NumberReady {
				pods, err := f.GetAllPods(obj.ObjectMeta)
				if err != nil {
					return false, err
				}

				for i := range pods {
					hasSidecar := false
					for _, c := range pods[i].Spec.Containers {
						if c.Name == util.StashContainer {
							hasSidecar = true
						}
					}
					if !hasSidecar {
						return false, nil
					}
				}
				return true, nil
			}
			return false, nil
		}
		return false, nil
	})
}

func (f *Invocation) WaitUntilDaemonSetReadyWithInitContainer(meta metav1.ObjectMeta) error {
	return wait.PollImmediate(kutil.RetryInterval, kutil.ReadinessTimeout, func() (bool, error) {
		if obj, err := f.KubeClient.AppsV1().DaemonSets(meta.Namespace).Get(meta.Name, metav1.GetOptions{}); err == nil {
			if obj.Status.DesiredNumberScheduled == obj.Status.NumberReady {
				pods, err := f.GetAllPods(obj.ObjectMeta)
				if err != nil {
					return false, err
				}

				for i := range pods {
					hasInitContainer := false
					for _, c := range pods[i].Spec.InitContainers {
						if c.Name == util.StashInitContainer {
							hasInitContainer = true
						}
					}
					if !hasInitContainer {
						return false, nil
					}
				}
				return true, nil
			}
			return false, nil
		}
		return false, nil
	})
}
