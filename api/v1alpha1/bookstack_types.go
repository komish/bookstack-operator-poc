/*
Copyright 2022 The OpDev Team.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// BookStackSpec defines the desired state of BookStack
type BookStackSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of BookStack. Edit bookstack_types.go to remove/update
	Foo string `json:"foo,omitempty"`
}

// BookStackStatus defines the observed state of BookStack
type BookStackStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// BookStack is the Schema for the bookstacks API
type BookStack struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BookStackSpec   `json:"spec,omitempty"`
	Status BookStackStatus `json:"status,omitempty"`
}

func (b *BookStack) NamespacedName() string {
	return client.ObjectKeyFromObject(b).String()
}

func (b *BookStack) GetServiceName() string {
	// TODO(): This will run into issues
	// if the instance's name hits max character
	// limits.
	return b.Name + "-svc"
}

func (b *BookStack) NewServiceAccount() corev1.ServiceAccount {
	return corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      b.GetName() + "-sa",
			Namespace: b.GetNamespace(),
			Labels:    labelsForInstance(*b),
		},
	}
}

// TODO Come back to this, PV and PVC first.
func (b *BookStack) NewDeployment() appsv1.Deployment {
	var one int32 = 1
	return appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      b.GetName(),
			Namespace: b.GetNamespace(),
			Labels:    labelsForInstance(*b),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &one,
			Selector: &metav1.LabelSelector{
				MatchLabels: selectorForInstance(*b),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      b.GetName(),
					Namespace: b.GetNamespace(),
					Labels:    labelsForInstance(*b),
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: "app-config",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: b.GetName() + "-pvc",
								},
							},
						},
						{
							Name: "db-config",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: b.GetName() + "-db-pvc",
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:  "bookstack",
							Image: "lscr.io/linuxserver/bookstack:latest",
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									HostPort:      6785,
									ContainerPort: 80,
									Protocol:      "TCP",
								},
							},
							EnvFrom: []corev1.EnvFromSource{
								{
									ConfigMapRef: &corev1.ConfigMapEnvSource{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: b.GetName() + "-cm",
										},
									},
								},
								{
									SecretRef: &corev1.SecretEnvSource{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: b.GetName() + "-secret",
										},
									},
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "app-config",
									MountPath: "/config",
								},
							},
						},
						{
							Name:  "bookstack-db",
							Image: "lscr.io/linuxserver/mariadb:latest",
							EnvFrom: []corev1.EnvFromSource{
								{
									ConfigMapRef: &corev1.ConfigMapEnvSource{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: b.GetName() + "-db-cm",
										},
									},
								},
								{
									SecretRef: &corev1.SecretEnvSource{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: b.GetName() + "-db-secret",
										},
									},
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "db-config",
									MountPath: "/config",
								},
							},
						},
					},
					ServiceAccountName: b.GetName() + "-sa",
				},
			},
		},
	}
}

func (b *BookStack) NewService() corev1.Service {
	return corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      b.GetServiceName(),
			Namespace: b.GetNamespace(),
			Labels:    labelsForInstance(*b),
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Protocol:   "TCP",
					Port:       80,
					TargetPort: intstr.IntOrString{IntVal: 80},
				},
			},
			Selector: selectorForInstance(*b),
			Type:     "NodePort",
		},
	}
}

func (b *BookStack) NewAppConfigMap() corev1.ConfigMap {
	return corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      b.GetName() + "-cm",
			Namespace: b.GetNamespace(),
		},
		Data: map[string]string{
			"APP_URL":     "http://example.com/", // placeholder, modified at creationtime
			"DB_DATABASE": "bookstackapp",
			"DB_HOST":     "localhost",
			"DB_USER":     "bookstack",
			"PGID":        "1000",
			"PUID":        "1000",
			"TZ":          "America/Chicago",
		},
	}
}

func (b *BookStack) NewDBConfigMap() corev1.ConfigMap {
	return corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      b.GetName() + "-db-cm",
			Namespace: b.GetNamespace(),
			Labels:    labelsForInstance(*b),
		},
		Data: map[string]string{
			"APP_URL":        "http://127.0.0.1:6875/",
			"MYSQL_DATABASE": "bookstackapp",
			"MYSQL_USER":     "bookstack",
			"PGID":           "1000",
			"PUID":           "1000",
			"TZ":             "America/Chicago",
		},
	}
}

func (b *BookStack) NewDBSecret() corev1.Secret {
	// TODO(): Change this before production use.
	return corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      b.GetName() + "-db-secret",
			Namespace: b.GetNamespace(),
			Labels:    labelsForInstance(*b),
		},
		Data: map[string][]byte{
			"MYSQL_ADMIN_PASS": []byte("superhunter2"),
			"MYSQL_PASSWORD":   []byte("hunter2"),
		},
	}
}

func (b *BookStack) NewAppSecret() corev1.Secret {
	// TODO(): Change this before production use.
	return corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      b.GetName() + "-secret",
			Namespace: b.GetNamespace(),
			Labels:    labelsForInstance(*b),
		},
		Data: map[string][]byte{
			"DB_PASS": []byte("hunter2"),
		},
	}
}

func (b *BookStack) NewAppPersistentVolume() corev1.PersistentVolume {
	storageSize, _ := resource.ParseQuantity("2Gi")
	return corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:      b.GetName() + "-pv",
			Namespace: b.GetNamespace(),
			Labels:    labelsForInstance(*b),
		},
		Spec: corev1.PersistentVolumeSpec{
			Capacity: map[corev1.ResourceName]resource.Quantity{
				"storage": storageSize,
			},
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			PersistentVolumeSource: corev1.PersistentVolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/mnt/app-config",
				},
			},
			StorageClassName:              "manual",                             // PoC using Kubernetes in Docker Desktop.
			PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimDelete, // for PoC
		},
	}
}

func (b *BookStack) NewDBPersistentVolume() corev1.PersistentVolume {
	storageSize, _ := resource.ParseQuantity("2Gi")
	return corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:      b.GetName() + "-db-pv",
			Namespace: b.GetNamespace(),
			Labels:    labelsForInstance(*b),
		},
		Spec: corev1.PersistentVolumeSpec{
			Capacity: map[corev1.ResourceName]resource.Quantity{
				"storage": storageSize,
			},
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			PersistentVolumeSource: corev1.PersistentVolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/mnt/db-config",
				},
			},
			StorageClassName:              "manual",                             // PoC using Kubernetes in Docker Desktop.
			PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimDelete, // for PoC
		},
	}
}

func (b *BookStack) NewDBPersistentVolumeClaim() corev1.PersistentVolumeClaim {
	manual := "manual"
	return corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      b.GetName() + "-db-pvc",
			Namespace: b.GetNamespace(),
			Labels:    labelsForInstance(*b),
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			Resources: corev1.ResourceRequirements{
				Requests: map[corev1.ResourceName]resource.Quantity{
					"storage": *resource.NewQuantity(2, resource.Format("Gi")),
				},
			},
			StorageClassName: &manual,
		},
	}
}

func (b *BookStack) NewAppPersistentVolumeClaim() corev1.PersistentVolumeClaim {
	manual := "manual"
	return corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      b.GetName() + "-pvc",
			Namespace: b.GetNamespace(),
			Labels:    labelsForInstance(*b),
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			Resources: corev1.ResourceRequirements{
				Requests: map[corev1.ResourceName]resource.Quantity{
					"storage": *resource.NewQuantity(2, resource.Format("Gi")),
				},
			},
			StorageClassName: &manual,
		},
	}
}

//+kubebuilder:object:root=true

// BookStackList contains a list of BookStack
type BookStackList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BookStack `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BookStack{}, &BookStackList{})
}
