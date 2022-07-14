package main

import (
	"flag"
	"fmt"

	"github.com/ForestsoftGmbH/devctl/client"
)

func main() {

	remotePort := flag.Int("port", 22, "Port for remote connection")
	kubeClient, config, workingNamespace := client.New()

	fmt.Println("Working namespace:", workingNamespace)

	var service *client.Service
	if *remotePort == 22 {
		service = client.GetSSHService(kubeClient, workingNamespace)
	} else if *remotePort == 3306 {
		service = client.GetDatabaseService(kubeClient, workingNamespace)
	} else {
		panic("Unknown port")
	}

	client.Forward(service, config)
}
