package main

import (
	"flag"
	"fmt"
	"k8s.io/client-go/tools/clientcmd"
	"os"

	"github.com/ForestsoftGmbH/devctl/client"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var kubeClient *kubernetes.Clientset
var workingNamespace string
var config *rest.Config
var serviceCollection *client.ServiceCollection

func main() {
	version := "1.0.26"
	fmt.Println("Running devctl version", version)

	remotePort := flag.Int("p", 0, "Port for remote connection")

	rules, errList := clientcmd.NewDefaultClientConfigLoadingRules().Load()
	if errList != nil {
		fmt.Println("Could not find any kubernetes context. please define .kubeconfig")
		os.Exit(255)
	}
	if len(rules.Contexts) < 1 {
		fmt.Println("Could not find any kubernetes context. please define .kubeconfig")
		os.Exit(255)
	}

	kubeClient, config, workingNamespace = client.New()
	fmt.Println("Working namespace:", workingNamespace)

	serviceCollection = &client.ServiceCollection{}
	services := make([]client.Service, 0)
	serviceCollection.Services = services

	if *remotePort == 22 {
		addSSHService()
	} else if *remotePort == 3306 {
		addDatabaseService()
	} else {
		addSSHService()
		addDatabaseService()
	}
	client.Forward(serviceCollection, config)
}

func addDatabaseService() {
	databaseService := client.GetDatabaseService(kubeClient, workingNamespace)
	if databaseService.Port > 0 {
		serviceCollection.Services = append(serviceCollection.Services, *databaseService)
	} else {
		fmt.Println("No database service found")
	}
}
func addSSHService() {
	sshService := client.GetSSHService(kubeClient, workingNamespace)
	if sshService.Port > 0 {
		serviceCollection.Services = append(serviceCollection.Services, *sshService)
	} else {
		fmt.Println("No ssh service found")
	}
}
