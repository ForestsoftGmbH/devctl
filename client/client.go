package client

import (
	"flag"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func New() (*kubernetes.Clientset, *rest.Config, string) {

	var kubeconfig *string

	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}

	ns := flag.String("namespace", "", "namespace to use")

	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)

	if *ns == "" {
		rules, _ := clientcmd.NewDefaultClientConfigLoadingRules().Load()
		*ns = rules.Contexts[rules.CurrentContext].Namespace
	}

	if err != nil {
		panic(err.Error())
	}

	clientset, err := kubernetes.NewForConfig(config)

	if err != nil {
		panic(err.Error())
	}

	return clientset, config, *ns

}
