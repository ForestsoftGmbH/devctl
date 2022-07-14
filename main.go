package main

import (
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/ForestsoftGmbH/devctl/client"
	v1 "k8s.io/api/core/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

func main() {

	var wg sync.WaitGroup
	wg.Add(1)

	remotePort := flag.Int("port", 22, "Port for remote connection")
	kubeClient, config, workingNamespace := client.New()

	fmt.Println("Working namespace:", workingNamespace)
	// stopCh control the port forwarding lifecycle. When it gets closed the
	// port forward will terminate
	stopCh := make(chan struct{}, 1)
	// readyCh communicate when the port forward is ready to get traffic
	readyCh := make(chan struct{})
	// stream is used to tell the port forwarder where to place its output or
	// where to expect input if needed. For the port forwarding we just need
	// the output eventually
	stream := genericclioptions.IOStreams{
		In:     os.Stdin,
		Out:    os.Stdout,
		ErrOut: os.Stderr,
	}

	// managing termination signal from the terminal. As you can see the stopCh
	// gets closed to gracefully handle its termination.
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	var service *client.Service

	if *remotePort == 22 {
		service = client.GetSSHService(kubeClient, workingNamespace)
	} else if *remotePort == 3306 {
		service = client.GetDatabaseService(kubeClient, workingNamespace)
	} else {
		panic("Unknown port")
	}

	go func() {
		<-sigs
		fmt.Println("Bye...")
		close(stopCh)
		wg.Done()
	}()

	go func() {
		// PortForward the pod specified from its port 9090 to the local port
		// 8080
		fmt.Printf("Opening connection to %s:%s\n", service.Name, service.Pod.Name)
		err := PortForwardAPod(PortForwardAPodRequest{
			RestConfig: config,
			Pod:        service.Pod,
			LocalPort:  service.LocalPort,
			PodPort:    service.Port,
			Streams:    stream,
			StopCh:     stopCh,
			ReadyCh:    readyCh,
		})
		if err != nil {
			panic(err)
		}
	}()

	select {
	case <-readyCh:
		break
	}
	fmt.Printf("Port forwarding to %v in namespace %v is ready to get traffic. have fun!\n", service.Name, workingNamespace)

	wg.Wait()

}

func PortForwardAPod(req PortForwardAPodRequest) error {
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
