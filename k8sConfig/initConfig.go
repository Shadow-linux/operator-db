package k8sConfig

import (
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"log"
	"os"
)

const (
	NSFile = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
)

//POD里  体内
func K8sRestConfigInPod() *rest.Config {

	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatal(err)
	}
	return config
}

func K8sRestConfig() *rest.Config {

	if os.Getenv("release") == "1" { //自定义环境
		log.Println("run in cluster")
		return K8sRestConfigInPod()
	}

	log.Println("run outside cluster")
	config, err := clientcmd.BuildConfigFromFlags("", "/Users/liangyedong/.kube/config")
	if err != nil {
		log.Fatal(err)
	}

	config.Insecure = false
	return config
}
