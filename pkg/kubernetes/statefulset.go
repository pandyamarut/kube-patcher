package kubernetes

import (
	"context"
	"encoding/json"

	jsonpatch "github.com/evanphx/json-patch"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// GetStatefulset returns the stateful set.
func GetStatefulset(name, namespace string) (*v1.StatefulSet, error) {
	statefulset, err := ClientSet.AppsV1().StatefulSets(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return statefulset, nil
}

// GetContainerPatch will return patch
func GetContainerPatch() corev1.Container {
	return corev1.Container{
		Env: []corev1.EnvVar{
			{
				Name:  "CASSANDRA_EXPORTER_CONFIG_listenPort",
				Value: "5556",
			},
			{
				Name: "JVM_OPTS",
			},
		},
		Image:           "criteord/cassandra_exporter:2.0.2",
		ImagePullPolicy: "IfNotPresent",
		LivenessProbe: &corev1.Probe{
			FailureThreshold: 3,
			PeriodSeconds:    10,
			SuccessThreshold: 1,
			Handler: corev1.Handler{
				TCPSocket: &corev1.TCPSocketAction{
					Port: intstr.FromInt(5556),
				},
			},
			TimeoutSeconds: 1,
		},
		Name: "casssandra-exporter",
		Ports: []corev1.ContainerPort{
			{
				ContainerPort: 5556,
				Name:          "metrics",
				Protocol:      "TCP",
			},
			{
				ContainerPort: 5555,
				Name:          "jmx",
				Protocol:      "TCP",
			},
		},
		ReadinessProbe: &corev1.Probe{
			FailureThreshold: 10,
			Handler: corev1.Handler{
				HTTPGet: &corev1.HTTPGetAction{
					Path:   "/metrics",
					Port:   intstr.FromInt(5556),
					Scheme: "HTTP",
				},
			},
			InitialDelaySeconds: 20,
			PeriodSeconds:       10,
			SuccessThreshold:    1,
			TimeoutSeconds:      45,
		},
		TerminationMessagePath:   "/dev/termination-log",
		TerminationMessagePolicy: "File",
	}
}

// StatefulsetPatch will patch the resource.
func StatefulsetPatch(envMap map[string]string) error {
	resource, err := GetStatefulset(envMap["resourceName"], envMap["namespace"])
	if err != nil {
		return err
	}
	oldResourceManifestByte, err := json.Marshal(resource)
	if err != nil {
		return err
	}
	resource.Spec.Template.Spec.Containers = append(resource.Spec.Template.Spec.Containers, GetContainerPatch())
	newResourceManifestByte, err := json.Marshal(resource)
	if err != nil {
		return err
	}
	patchBytes, err := jsonpatch.CreateMergePatch(oldResourceManifestByte, newResourceManifestByte)
	//fmt.Println(string(patchBytes))
	_, err = ClientSet.AppsV1().StatefulSets(envMap["namespace"]).Patch(context.TODO(), envMap["resourceName"], types.StrategicMergePatchType, patchBytes, metav1.PatchOptions{}, "")
	if err != nil {
		return err
	}
	return nil
}
