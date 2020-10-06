package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	y "github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	kubeconfig := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		panic(err.Error())
	}
	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	st, err := clientset.AppsV1().StatefulSets("default").Get(context.TODO(), "cassandra-1601974503", metav1.GetOptions{})
	if err != nil {
		panic(err.Error())
	}
	containers := corev1.Container{
		Env: []corev1.EnvVar{
			{
				Name:  "CASSANDRA_EXPORTER_CONFIG_listenport",
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
	//oldData, err := json.Marshal(st)
	//if err != nil {
	//	panic(err.Error())
	//}
	st.Spec.Template.Spec.Containers = append(st.Spec.Template.Spec.Containers, containers)
	//newObj, err := json.Marshal(st)
	//if err != nil {
	//	panic(err.Error())
	//}
	//patchBytes, err := jsonpatch.CreateMergePatch(oldData, newObj)
	//fmt.Println(string(patchBytes))
	//_, err = clientset.AppsV1().StatefulSets("default").Patch(context.TODO(), "cassandra-1601968158", types.MergePatchType, patchBytes, metav1.PatchOptions{}, "")
	//if err != nil {
	//	panic(err.Error())
	//}
	err = doSSA(context.TODO(), config, st)
	if err != nil {
		logrus.Println(err)
	}

	// fmt.Println(sts)
}

var decUnstructured = yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)

func doSSA(ctx context.Context, cfg *rest.Config, stringManifest *v1.StatefulSet) error {
	// 1. Prepare a RESTMapper to find GVR
	dc, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return err
	}
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(dc))
	// 2. Prepare the dynamic client
	dyn, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return err
	}
	// 3. Decode YAML manifest into unstructured.Unstructured
	yml, _ := y.Marshal(stringManifest)
	obj := &unstructured.Unstructured{}
	_, gvk, err := decUnstructured.Decode(yml, nil, obj)
	if err != nil {
		return err
	}
	// 4. Find GVR
	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return err
	}
	logrus.Print(mapping.Scope.Name())
	// 5. Obtain REST interface for the GVR
	var dr dynamic.ResourceInterface
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		// namespaced resources should specify the namespace
		dr = dyn.Resource(mapping.Resource).Namespace(obj.GetNamespace())
	} else {
		// for cluster-wide resources
		dr = dyn.Resource(mapping.Resource)
	}
	// 6. Marshal object into JSON
	data, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	// 7. Create or Update the object with SSA
	//     types.ApplyPatchType indicates SSA.
	//     FieldManager specifies the field owner ID.
	_, err = dr.Patch(ctx, obj.GetName(), types.StrategicMergePatchType, data, metav1.PatchOptions{
		//FieldManager: "backup-agent",
	})
	//_, err = dr.Create(ctx, obj, metav1.CreateOptions{})
	//_, err = dr.Patch()
	//l, err := dr.List(ctx, metav1.ListOptions{})
	//logrus.Print(l)
	//
	return err
}
