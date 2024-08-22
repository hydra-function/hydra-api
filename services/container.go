package services

import (
	"context"
	"flag"
	"time"

	"github.com/spf13/viper"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	// Load kubeconfig
	kubeconfig := viper.GetString("kubeconfig.path")

	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	// Define the pod
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "express-app",
			Labels: map[string]string{
				"app": "express-app",
			},
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:    "express-container",
					Image:   "node:14", // Imagem base para o Node.js
					Command: []string{"sh", "-c", "npm install express && node -e \"const express = require('express'); const app = express(); app.get('/', (req, res) => res.send('Hello World!')); app.listen(3000);\""},
					Ports: []v1.ContainerPort{
						{
							ContainerPort: 3000,
						},
					},
				},
			},
		},
	}

	podsClient := clientset.CoreV1().Pods("default")
	result, err := podsClient.Create(context.TODO(), pod, metav1.CreateOptions{})
	if err != nil {
		panic(err.Error())
	}

	println("Created pod", result.GetObjectMeta().GetName())

	for {
		p, err := podsClient.Get(context.TODO(), "express-app", metav1.GetOptions{})
		if err != nil {
			panic(err.Error())
		}
		if p.Status.Phase == v1.PodRunning {
			break
		}
		time.Sleep(2 * time.Second)
	}

	println("Pod is running!")
}
