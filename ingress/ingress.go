package ingress

import (
	"context"
	"fmt"
	"path/filepath"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

type Ingress struct {
	Namespace string
	Slug      string
	Host      string
	Port      int32
}

func (i *Ingress) Create() error {
	if err := i.createService(); err != nil {
		return err
	}
	return i.createIngress()
}

func (i *Ingress) createService() error {
	kubeconfig := filepath.Join(homedir.HomeDir(), ".kube", "config")
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: i.Namespace + "-" + i.Slug + "-service",
			Labels: map[string]string{
				"app": i.Slug,
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": i.Slug,
			},
			Ports: []corev1.ServicePort{
				{
					Protocol:   corev1.ProtocolTCP,
					Port:       i.Port,
					TargetPort: intstr.FromInt(80), // Porta no pod
				},
			},
			Type: corev1.ServiceTypeClusterIP,
		},
	}

	servicesClient := clientset.CoreV1().Services(i.Namespace)
	_, err = servicesClient.Get(context.TODO(), service.Name, metav1.GetOptions{})
	if err == nil {
		// Serviço já existe, então atualize-o
		_, err = servicesClient.Update(context.TODO(), service, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
		fmt.Printf("Updated Service %q.\n", service.GetObjectMeta().GetName())
		return nil
	}

	if !errors.IsNotFound(err) {
		// Erro diferente de "não encontrado"
		return err
	}

	// Serviço não existe, então crie-o
	_, err = servicesClient.Create(context.TODO(), service, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	fmt.Printf("Created Service %q.\n", service.GetObjectMeta().GetName())
	return nil
}

func (i *Ingress) createIngress() error {
	kubeconfig := filepath.Join(homedir.HomeDir(), ".kube", "config")
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	configSnippet := `
      set $service_name "";

      if ($http_x_function_namespace != "") {
        set $service_name $http_x_function_namespace;
      }
      if ($http_x_function_id != "") {
        set $service_name "${service_name}-$http_x_function_id";
      }
      
      set $service_name "http://$service_name-service:3001";
      
      proxy_pass $service_name;
    `

	ingress := &v1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      i.Slug,
			Namespace: i.Namespace,
			Annotations: map[string]string{
				"nginx.ingress.kubernetes.io/configuration-snippet": configSnippet,
			},
		},
		Spec: v1.IngressSpec{
			Rules: []v1.IngressRule{
				{
					Host: i.Host,
					IngressRuleValue: v1.IngressRuleValue{
						HTTP: &v1.HTTPIngressRuleValue{
							Paths: []v1.HTTPIngressPath{
								{
									Path: "/",
									PathType: func() *v1.PathType {
										pathType := v1.PathTypePrefix
										return &pathType
									}(),
									Backend: v1.IngressBackend{
										Service: &v1.IngressServiceBackend{
											Name: fmt.Sprintf("%s-%s-service", i.Namespace, i.Slug),
											Port: v1.ServiceBackendPort{
												Number: i.Port,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	ingressClient := clientset.NetworkingV1().Ingresses(i.Namespace)
	existingIngress, err := ingressClient.Get(context.TODO(), i.Slug, metav1.GetOptions{})
	if err == nil {
		ingress.ResourceVersion = existingIngress.ResourceVersion
		_, err = ingressClient.Update(context.TODO(), ingress, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
		fmt.Printf("Updated Ingress %q.\n", i.Slug)
		return nil
	}

	if !errors.IsNotFound(err) {
		return err
	}

	result, err := ingressClient.Create(context.TODO(), ingress, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	fmt.Printf("Created Ingress %q.\n", result.GetObjectMeta().GetName())
	return nil
}
