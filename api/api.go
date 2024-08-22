package api

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	_ "github.com/hydra-function/hydra-api/config"
	"github.com/hydra-function/hydra-api/db"
	"github.com/hydra-function/hydra-api/ingress"
	"github.com/labstack/echo"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	netV1 "k8s.io/api/networking/v1"
)

type PodInfo struct {
	Name      string
	Namespace string
}

func Start() *echo.Echo {
	e := echo.New()

	// cacheService, err := cache.NewCacheService()
	// if err != nil {
	// 	e.Logger.Fatal(err)
	// }

	ing := ingress.Ingress{
		Namespace: "foo",
		Slug:      "hydra-ingress",
		Host:      "localhost",
		Port:      80,
	}

	if err := ing.Create(); err != nil {
		log.Fatalf("Error creating Ingress: %v", err)
	}

	e.GET("/healthcheck", func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	e.PUT("/api/install/:namespace/:slug", func(c echo.Context) error {
		namespace := c.Param("namespace")
		slug := c.Param("slug")

		err := createPod(namespace, slug)
		if err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("Failed to create pod: %s", err.Error()))
		}

		err = createService(namespace, slug)
		if err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("Failed to create service: %s", err.Error()))
		}

		path := netV1.HTTPIngressPath{
			Path: fmt.Sprintf("/run/%s/%s/", namespace, slug),
			PathType: func() *netV1.PathType {
				pathType := netV1.PathTypePrefix
				return &pathType
			}(),
			Backend: netV1.IngressBackend{
				Service: &netV1.IngressServiceBackend{
					Name: fmt.Sprintf("%s-%s-service", namespace, slug),
					Port: netV1.ServiceBackendPort{
						Number: 3001,
					},
				},
			},
		}

		err = ing.AddPath(path)
		if err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("Failed to add path: %s", err.Error()))
		}

		dbInstance, err := db.New()
		if err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("Failed to connect to database: %s", err.Error()))
		}

		collection := dbInstance.Client.Database(viper.GetString("database.name")).Collection("functions")
		filter := bson.M{"Namespace": namespace, "Name": slug}
		update := bson.M{
			"$set": bson.M{
				"Name":      slug,
				"Namespace": namespace,
			},
		}

		opts := options.Update().SetUpsert(true)
		_, err = collection.UpdateOne(context.Background(), filter, update, opts)
		if err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("Failed to upsert document into MongoDB: %s", err.Error()))
		}

		return c.String(http.StatusOK, "Pod and service created successfully")
	})

	e.POST("/api/run/:namespace/:slug", func(c echo.Context) error {
		slug := c.Param("slug")
		namespace := c.Param("namespace")

		dbInstance, err := db.New()
		if err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("Failed to connect to database: %s", err.Error()))
		}

		collection := dbInstance.Client.Database(viper.GetString("database.name")).Collection("functions")
		filter := bson.M{"Namespace": namespace, "Name": slug}

		var podInfo map[string]interface{}

		err = collection.FindOne(context.Background(), filter).Decode(&podInfo)
		if err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("Failed to get document from MongoDB: %s", err.Error()))
		}

		// key := fmt.Sprintf("function:%s:%s", namespace, slug)

		// podInfo, err := cacheService.Get(key)
		// if err != nil {
		// 	return c.String(http.StatusInternalServerError, fmt.Sprintf("Failed to get cache %s", err.Error()))
		// }

		function_name := podInfo["Name"]
		function_namespace := podInfo["Namespace"]

		// headers := c.Request().Header

		// headers.Set("X-Function-Name", function_name.(string))
		// headers.Set("X-Function-Namespace", function_namespace.(string))

		client := &http.Client{}

		// command: curl -H "X-Function-Name: <function_name>" -H "X-Function-Namespace: <function_namespace>" http://hydra.local:1232/
		req, err := http.NewRequest("GET", fmt.Sprintf("http://localhost/run/%s/%s/", function_namespace.(string), function_name.(string)), nil)
		if err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("Failed to create request %s", err.Error()))
		}

		// req.Header = headers

		resp, err := client.Do(req)
		if err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("Failed to forward request %s", err.Error()))
		}

		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("Failed to read response body %s", err.Error()))
		}

		return c.String(http.StatusOK, string(body))
	})

	e.Logger.Fatal(e.Start(":1232"))
	return e
}

func createPod(namespace, slug string) error {
	// Load kubeconfig
	kubeconfig := viper.GetString("kubeconfig.path")
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return err
	}

	// Create clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	// Delete existing pod if it exists
	podsClient := clientset.CoreV1().Pods(namespace)
	err = podsClient.Delete(context.TODO(), slug, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	// Define the pod
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: slug,
			Labels: map[string]string{
				"app": slug,
			},
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:    "express-container",
					Image:   "node:14", // Imagem base para o Node.js
					Command: []string{"sh", "-c", "npm install express && node -e \"const express = require('express'); const app = express(); app.get('*', (req, res) => res.send('Hello World!')); app.listen(3000);\""},
					Ports: []v1.ContainerPort{
						{
							ContainerPort: 3000,
						},
					},
				},
			},
		},
	}

	// Create the pod in Kubernetes
	_, err = podsClient.Create(context.TODO(), pod, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	// Optionally, wait for pod to be running
	for {
		p, err := podsClient.Get(context.TODO(), slug, metav1.GetOptions{})
		if err != nil {
			return err
		}

		podLogsReq := podsClient.GetLogs(slug, &v1.PodLogOptions{})
		podLogs, err := podLogsReq.Stream(context.TODO())
		if err != nil {
			fmt.Printf("Erro ao obter logs: %v\n", err)
		} else {
			defer podLogs.Close()
			buf := new(bytes.Buffer)
			_, err := io.Copy(buf, podLogs)
			if err != nil {
				fmt.Printf("Erro ao copiar logs: %v\n", err)
			}
			fmt.Printf("Logs do Pod:\n%s\n", buf.String())
		}

		if p.Status.Phase == v1.PodRunning {
			break
		}
		time.Sleep(2 * time.Second)
	}

	return nil
}

func createService(namespace, slug string) error {
	// Load kubeconfig
	kubeconfig := viper.GetString("kubeconfig.path")
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return err
	}

	// Create clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	// Define the service
	service := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace + "-" + slug + "-service",
			Labels: map[string]string{
				"app": slug,
			},
		},
		Spec: v1.ServiceSpec{
			Selector: map[string]string{
				"app": slug,
			},
			Ports: []v1.ServicePort{
				{
					Protocol:   v1.ProtocolTCP,
					Port:       3001,                 // Porta alocada automaticamente
					TargetPort: intstr.FromInt(3000), // Porta no pod
				},
			},
			Type: v1.ServiceTypeClusterIP,
		},
	}

	// Create the service in Kubernetes
	servicesClient := clientset.CoreV1().Services(namespace)
	_, err = servicesClient.Create(context.TODO(), service, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	for _, port := range service.Spec.Ports {
		fmt.Printf("Porta alocada: %d\n", port.Port)
	}

	return nil
}
