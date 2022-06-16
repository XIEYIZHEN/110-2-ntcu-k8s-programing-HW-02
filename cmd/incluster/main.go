package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path"
	"syscall"
	"time"

	appv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	coreV1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

//deployment
func createDeploy(clientset *kubernetes.Clientset) *appv1.Deployment {
	namespace := "default"
	var replicas int32 = 1
	deployment := &appv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "nginx",
			Labels: map[string]string{
				"app":      "nginx",
				"env":      "dev",
				"ntcu-k8s": "hw2",
			},
		},
		Spec: appv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "nginx",
					"env": "dev",
				},
			},
			Template: coreV1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "nginx",
					Labels: map[string]string{
						"app": "nginx",
						"env": "dev",
					},
				},
				Spec: coreV1.PodSpec{
					Containers: []coreV1.Container{
						{
							Name:  "nginx",
							Image: "nginx:1.16.1",
							Ports: []coreV1.ContainerPort{
								{
									Name:          "http",
									Protocol:      coreV1.ProtocolTCP,
									ContainerPort: 80,
								},
							},
						},
					},
				},
			},
		},
	}

	fmt.Println("creat deployment")
	deploymentList, err := clientset.AppsV1().Deployments(namespace).Create(context.TODO(), deployment, metav1.CreateOptions{})
	fmt.Println(err, deploymentList)
	//fmt.Println("name ->", deploymentList.GetObjectMeta().GetName())
	return deploymentList
}

//svc
func createService(clientset *kubernetes.Clientset) {

	namespace := "default"
	service := &apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: "nginx-svc",
			Labels: map[string]string{
				"app":      "nginx",
				"ntcu-k8s": "hw2",
			},
		},
		Spec: apiv1.ServiceSpec{
			Selector: map[string]string{
				"app": "nginx",
			},
			Type: coreV1.ServiceTypeNodePort,
			Ports: []apiv1.ServicePort{
				{
					Name:     "http",
					Port:     80,
					NodePort: 30440,
					Protocol: apiv1.ProtocolTCP,
				},
			},
		},
	}
	serviceresult, err := clientset.CoreV1().Services(namespace).Create(context.TODO(), service, metav1.CreateOptions{})
	fmt.Println(err, serviceresult)

}

//delete
func deleteall(clientset *kubernetes.Clientset) {
	emptyDeleteOptions := metav1.DeleteOptions{}

	namespace := "default"

	//刪除deplyment
	errdep := clientset.AppsV1().Deployments(namespace).Delete(context.TODO(), "nginx", emptyDeleteOptions)
	fmt.Println(errdep)

	// 删除svc
	errser := clientset.CoreV1().Services(namespace).Delete(context.TODO(), "nginx-svc", emptyDeleteOptions)
	fmt.Println(errser)

}
func deleteConfigMap(client kubernetes.Interface, cm *coreV1.ConfigMap) {
	err := client.
		CoreV1().
		ConfigMaps(cm.GetNamespace()).
		Delete(
			context.Background(),
			cm.GetName(),
			metav1.DeleteOptions{},
		)
	if err != nil {
		panic(err.Error())
	}

	fmt.Printf("Deleted ConfigMap %s/%s\n", cm.GetNamespace(), cm.GetName())
}

func readDeploy(clientset kubernetes.Interface, name string) *appv1.Deployment {
	namespace := "default"
	read, err := clientset.
		AppsV1().
		Deployments(namespace).
		Get(
			context.Background(),
			name,
			metav1.GetOptions{},
		)
	if err != nil {
		panic(err.Error())
	}

	fmt.Printf("name %s/%s\n", namespace, read.GetName())
	return read
}
func readConfigMap(clientset kubernetes.Interface, name string) *coreV1.ConfigMap {
	namespace := "default"
	read, err := clientset.
		CoreV1().
		ConfigMaps(namespace).
		Get(
			context.Background(),
			name,
			metav1.GetOptions{},
		)
	if err != nil {
		panic(err.Error())
	}

	fmt.Printf("Read ConfigMap %s/%s, value is %s\n", namespace, read.GetName(), read.Data["foo"])
	return read
}

func createConfigMap(client kubernetes.Interface) *coreV1.ConfigMap {
	namespace := "default"
	cm := &coreV1.ConfigMap{Data: map[string]string{"foo": "bar"}}
	cm.Namespace = namespace
	cm.GenerateName = "informer-typed-simple-"

	cm, err := client.
		CoreV1().
		ConfigMaps(namespace).
		Create(
			context.Background(),
			cm,
			metav1.CreateOptions{},
		)
	if err != nil {
		panic(err.Error())
	}

	fmt.Printf("Created ConfigMap %s/%s\n", cm.GetNamespace(), cm.GetName())
	return cm
}
func main() {
	outsideCluster := flag.Bool("outside-cluster", false, "set to true when run out of cluster. (default: false)")
	flag.Parse()

	var clientset *kubernetes.Clientset
	if *outsideCluster {
		// creates the out-cluster config
		home, err := os.UserHomeDir()
		if err != nil {
			panic(err)
		}
		config, err := clientcmd.BuildConfigFromFlags("", path.Join(home, ".kube/config"))
		if err != nil {
			panic(err.Error())
		}
		// creates the clientset
		clientset, err = kubernetes.NewForConfig(config)
		if err != nil {
			panic(err.Error())
		}
	} else {
		// creates the in-cluster config
		config, err := rest.InClusterConfig()
		if err != nil {
			panic(err.Error())
		}
		// creates the clientset
		clientset, err = kubernetes.NewForConfig(config)
		if err != nil {
			panic(err.Error())
		}
	}
	cm := createConfigMap(clientset)
	go func() {
		for {
			readConfigMap(clientset, cm.GetName())
			time.Sleep(5 * time.Second)
		}
	}()

	dm := createDeploy(clientset)
	createService(clientset)
	go func() {
		for {
			readDeploy(clientset, dm.GetName())
			time.Sleep(5 * time.Second)
		}
	}()

	fmt.Println("Waiting for Kill Signal...")
	var stopChan = make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	<-stopChan

	time.Sleep(1 * time.Second)
	fmt.Println("delete deployment and service")
	deleteConfigMap(clientset, cm)
	deleteall(clientset)
	fmt.Println("done!!")

	//time.Sleep(time.Millisecond * 500)

}
