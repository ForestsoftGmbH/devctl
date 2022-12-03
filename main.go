package main

import (
	"flag"
	"fmt"

	"github.com/ForestsoftGmbH/devctl/client"
)

func main() {

	remotePort := flag.Int("p", 22, "Port for remote connection")
	kubeClient, config, workingNamespace := client.New()

	fmt.Println("Working namespace:", workingNamespace)

	var service *client.Service

	serviceCollection := &client.ServiceCollection{}

	if *remotePort == 22 {
		service = client.GetSSHService(kubeClient, workingNamespace)
		services := []client.Service{
			*service,
		}
		serviceCollection.Services = services
	} else if *remotePort == 3306 {
		service = client.GetDatabaseService(kubeClient, workingNamespace)
		services := []client.Service{
			*service,
		}
		serviceCollection.Services = services
	} else {
		sshService := client.GetSSHService(kubeClient, workingNamespace)
		databaseService := client.GetDatabaseService(kubeClient, workingNamespace)
		services := []client.Service{
			*sshService,
			*databaseService,
		}
		serviceCollection.Services = services
	}

	client.Forward(serviceCollection, config)
}
