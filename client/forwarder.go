package client

import (
	"context"
	"fmt"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
)

type ServiceCollection struct {
	Services []Service
}

var wg sync.WaitGroup
var stream genericclioptions.IOStreams
var stopChannels []chan struct{}

func Forward(services *ServiceCollection, config *rest.Config) {

	// managing termination signal from the terminal. As you can see the stopCh
	// gets closed to gracefully handle its termination.
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)

	// stream is used to tell the port forwarder where to place its output or
	// where to expect input if needed. For the port forwarding we just need
	// the output eventually
	stream = genericclioptions.IOStreams{
		In:     os.Stdin,
		Out:    os.Stdout,
		ErrOut: os.Stderr,
	}

	stopChannels = make([]chan struct{}, len(services.Services))

	for i := range stopChannels {
		stopChannels[i] = make(chan struct{}, 1)
	}

	go func(stopChannels []chan struct{}) {
		<-sigs
		fmt.Println("Bye...")
		for _, stopCh := range stopChannels {
			close(stopCh)
			wg.Done()
		}
	}(stopChannels)

	// PortForward the pod specified from its port 9090 to the local port
	// 8080
	serviceCounter := 0
	for _, service := range services.Services {
		// stopCh control the port forwarding lifecycle. When it gets closed the
		// port forward will terminate

		// readyCh communicate when the port forward is ready to get traffic
		readyCh := make(chan struct{})
		wg.Add(1)
		request := PortForwardAPodRequest{
			RestConfig: config,
			Pod:        service.Pod,
			LocalPort:  service.LocalPort,
			PodPort:    service.Port,
			Streams:    stream,
			StopCh:     stopChannels[serviceCounter],
			ReadyCh:    readyCh,
		}
		service.PortForwardRequest = &request
		go func(service Service, serviceCounter int, request PortForwardAPodRequest) {

			fmt.Printf("Opening connection to %s:%s\n", service.Name, service.Pod.Name)
			err := PortForwardAPod(&request)
			if err != nil {
				panic(err)
			}
		}(service, serviceCounter, request)

		go func(svc Service) {
			err := watchPodForTermination(svc)
			if err != nil {
				log.Println("Could not watch for termination on pod " + service.Pod.Name)
			}
		}(service)

		select {
		case <-readyCh:
			break
		}

		serviceCounter++
	}
	wg.Wait()
}

func watchPodForTermination(svc Service) error {

	ctx := context.Background()

	// Create a watcher for pod events
	namespace := svc.Namespace
	// Create a watcher for pod events
	opts := metav1.ListOptions{
		TypeMeta:      metav1.TypeMeta{},
		LabelSelector: "",
		FieldSelector: fmt.Sprintf("metadata.name=%s", svc.Pod.Name),
	}
	watcher, err := svc.Client.CoreV1().Pods(namespace).Watch(ctx, opts)
	// Stop the watcher
	defer watcher.Stop()

	if err != nil {
		return err
	}
	log.Println("Watch for status changes " + svc.Pod.Name)
	for {
		select {
		case event := <-watcher.ResultChan():
			if event.Type == watch.Deleted {
				log.Printf("The POD \"%s\" is deleted", svc.Pod.Name)
				restartPortForwarding(svc)
				return nil
			}

		case <-ctx.Done():
			log.Printf("Exit from waitPodDeleted for POD \"%s\" because the context is done", svc.Pod.Name)
			return nil
		}
	}
}

func restartPortForwarding(svc Service) {
	log.Println("wait for new pod commin up")
	ctx := context.Background()

	// Create a watcher for pod events
	namespace := svc.Namespace
	// Create a watcher for pod events
	opts := metav1.ListOptions{
		TypeMeta:      metav1.TypeMeta{},
		LabelSelector: "",
		FieldSelector: fmt.Sprintf("metadata.name=%s", svc.Pod.Name),
	}
	watcher, err := svc.Client.CoreV1().Pods(namespace).Watch(ctx, opts)
	// Stop the watcher
	defer watcher.Stop()

	if err != nil {
		log.Println("error watching pod ", err)
	}
	for {
		select {
		case event := <-watcher.ResultChan():
			if event.Type == watch.Modified {
				pod := event.Object.(*v1.Pod)
				if pod.Status.Phase == v1.PodRunning {
					log.Printf("The new POD \"%s\" is running", svc.Pod.Name)
					closed := false
					for index, stopCh := range stopChannels {
						if stopCh == svc.PortForwardRequest.StopCh {
							close(stopCh)
							stopChannels[index] = make(chan struct{}, 1)
							svc.PortForwardRequest.StopCh = stopChannels[index]
							svc.PortForwardRequest.ReadyCh = make(chan struct{})
							closed = true
						}
					}
					if !closed {
						fmt.Println("The previous channel for pod", svc.Pod.Name, "couldnt been closed")
					}
					go func(service Service) {
						err := watchPodForTermination(svc)
						if err != nil {
							log.Println("Error on watching the new pod", err)
						}
					}(svc)
					go func(service Service) {
						err = PortForwardAPod(svc.PortForwardRequest)
						if err != nil {
							panic(err)
						}
					}(svc)
					watcher.Stop()
					break
				}
			}
		case <-ctx.Done():
			log.Printf("Exit from restartPortForwarding for POD \"%s\" because the context is done", svc.Pod.Name)
			watcher.Stop()
			break
		}
	}
}

func PortForwardAPod(req *PortForwardAPodRequest) error {
	path := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/portforward",
		req.Pod.Namespace, req.Pod.Name)
	hostIP := strings.TrimLeft(req.RestConfig.Host, "htps:/")

	transport, upgrader, err := spdy.RoundTripperFor(req.RestConfig)
	if err != nil {
		return err
	}

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, http.MethodPost, &url.URL{Scheme: "https", Path: path, Host: hostIP})
	fw, err := portforward.New(dialer, []string{fmt.Sprintf("%d:%d", req.LocalPort, req.PodPort)}, req.StopCh, req.ReadyCh, req.Streams.Out, req.Streams.ErrOut)
	if err != nil {
		return err
	}
	return fw.ForwardPorts()
}

type PortForwardAPodRequest struct {
	// RestConfig is the kubernetes config
	RestConfig *rest.Config
	// Pod is the selected pod for this port forwarding
	Pod v1.Pod
	// LocalPort is the local port that will be selected to expose the PodPort
	LocalPort int
	// PodPort is the target port for the pod
	PodPort int
	// Steams configures where to write or read input from
	Streams genericclioptions.IOStreams
	// StopCh is the channel used to manage the port forward lifecycle
	StopCh <-chan struct{}
	// ReadyCh communicates when the tunnel is ready to receive traffic
	ReadyCh chan struct{}
}
