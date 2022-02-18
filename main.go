package main

import "github.com/shenyisyn/dbcore/k8sConfig"

func main()  {
	//k8sCfg := k8sConfig.K8sRestConfig()
	//client, err := clientv1.NewForConfig(k8sCfg)
	//
	//if err != nil {
	//	log.Fatal(err)
	//}
	//
	//dcList, _ := client.DbConfigs("default").List(context.Background(), metav1.ListOptions{})
	//fmt.Println(dcList)

	// 初始化并启动manager
	k8sConfig.InitManager()

}