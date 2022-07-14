package client_test

import (
	"context"
	"testing"

	"github.com/ForestsoftGmbH/devctl/client"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	testclient "k8s.io/client-go/kubernetes/fake"
)

func TestGetSSHService(t *testing.T) {

	kubeClient := testclient.NewSimpleClientset()

	nsSpec := &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "default"}}

	kubeClient.CoreV1().Namespaces().Create(context.TODO(), nsSpec, metav1.CreateOptions{})

	svc := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sebastian-dev",
			Namespace: "default",
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{Port: 22, TargetPort: intstr.FromInt(2222)},
			},
			Selector: map[string]string{
				"app":                    "web",
				"app.kubernetes.io/name": "sebastian-dev",
			},
		},
	}

	kubeClient.CoreV1().Services("default").Create(context.TODO(), svc, metav1.CreateOptions{})

	pod := &v1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},

		ObjectMeta: metav1.ObjectMeta{
			Name:      "sebastian-dev-0",
			Namespace: "default",
			Labels: map[string]string{
				"app":                    "web",
				"app.kubernetes.io/name": "sebastian-dev",
			},
		},
		Spec: v1.PodSpec{

			Containers: []v1.Container{
				{
					Name:            "nginx",
					Image:           "nginx",
					ImagePullPolicy: "Always",
				},
			},
		},
	}

	kubeClient.CoreV1().Pods("default").Create(context.TODO(), pod, metav1.CreateOptions{})

	actual := client.GetSSHService(kubeClient, "default")

	assert.Equal(t, 22, actual.Port)
	assert.Equal(t, "sebastian-dev", actual.Name)
}
