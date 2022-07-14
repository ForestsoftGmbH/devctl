package client

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

type Service struct {
	Port      int
	LocalPort int
	Name      string
	Namespace string
	Pod       v1.Pod
}

func GetSSHService(client kubernetes.Interface, namespace string) *Service {
	svc := Service{Port: 0, Name: "Not found"}
	return svc.getServiceWithPort(client, namespace, 22, 2222)
}

func GetDatabaseService(client kubernetes.Interface, namespace string) *Service {
	svc := Service{Port: 0, Name: "Not found"}
	return svc.getServiceWithPort(client, namespace, 3306, 3306)
}

func (svc Service) getServiceWithPort(client kubernetes.Interface, namespace string, requiredPort int32, localPort int32) *Service {

	services, err := client.CoreV1().Services(namespace).List(context.TODO(), metav1.ListOptions{})

	if err != nil {
		panic(err.Error())
	}

	for _, service := range services.Items {
		for _, port := range service.Spec.Ports {
			if port.Port == requiredPort {
				if pods, err := client.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: labels.SelectorFromSet(service.Spec.Selector).String()}); err != nil {
					fmt.Printf("List Pods of service[%s] error:%v", service.GetName(), err)
				} else {
					for _, v := range pods.Items {
						svc.Pod = v
						svc.Name = service.Name
						svc.Namespace = namespace
						svc.Port = int(requiredPort)
						svc.LocalPort = int(localPort)
						return &svc
					}
				}
			}
		}
	}

	return &svc
}
