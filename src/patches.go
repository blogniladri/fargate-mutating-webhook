package main

import (
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	_ "k8s.io/client-go/applyconfigurations/core/v1"
	"strconv"
	"strings"
	"time"
)

func createPatch(pod corev1.Pod) ([]patchOperation, error) { //, sidecarConfig *Config

	m.Lock()

	logger.Info("inside createPatch >>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>> ")
	time.Sleep(1 * time.Second)

	var patches []patchOperation

	labels := pod.ObjectMeta.Labels
	annotations := pod.ObjectMeta.Annotations
	schedulerName := "default-scheduler"

	nodePodCnt := getNodePodCount(pod)
	customHpaStrategy, fargate_profile_name := getCustomHpaStrategy(pod)

	logger.Info("isFargateTarget() nodePodCnt ", nodePodCnt)
	logger.Info("isFargateTarget() node-pod-max-count ", customHpaStrategy)

	logger.Info("isFargateTarget() returning ", (nodePodCnt >= customHpaStrategy))

	if nodePodCnt >= customHpaStrategy { //is fargate
		labels["custom-hpa-enabled"] = "1"
		annotations["controller.kubernetes.io/pod-deletion-cost"] = "0"
		schedulerName = "fargate-scheduler"

		fargate_profile_labels := pod.ObjectMeta.Labels
		fargate_profile_labels["eks.amazonaws.com/fargate-profile"] = fargate_profile_name

		patches = append(patches, patchOperation{
			Op:    "add",
			Path:  "/metadata/labels",
			Value: fargate_profile_labels,
		})

	} else { //Node
		labels["custom-hpa-enabled"] = "0"
		annotations["controller.kubernetes.io/pod-deletion-cost"] = "100"
	}

	patches = append(patches, patchOperation{
		Op:    "add",
		Path:  "/metadata/labels",
		Value: labels,
	})

	patches = append(patches, patchOperation{
		Op:    "add",
		Path:  "/metadata/annotations",
		Value: annotations,
	})

	patches = append(patches, patchOperation{
		Op:    "replace",
		Path:  "/spec/schedulerName",
		Value: schedulerName,
	})

	logger.Info("existing createPatch<<<<<<<<<<<<<<<<<<<<")
	m.Unlock()

	return patches, nil
}

// func isFargateTarget(pod corev1.Pod) bool {

// 	nodePodCnt := getNodePodCount(pod)
// 	customHpaStrategy := getCustomHpaStrategy(pod)

// 	logger.Info("isFargateTarget() nodePodCnt ", nodePodCnt)
// 	logger.Info("isFargateTarget() node-pod-max-count ", customHpaStrategy)

// 	logger.Info("isFargateTarget() returning ", (nodePodCnt >= customHpaStrategy))

// 	return (nodePodCnt >= customHpaStrategy)

// }

// //TODO
// func getFargateProfileName(pod corev1.Pod) string {
// 	return "test"
// }

//TODO
func getNodePodCount(pod corev1.Pod) int {

	logger.Info("inside getNodePodCount ")
	logger.Info("inside getNodePodCount : pod.Namespace ", pod.Namespace)

	pod_template_hash := pod.Labels["pod-template-hash"]

	logger.Info("inside getNodePodCount : pod_template_hash ", pod_template_hash)

	//podNameSplitList := strings.Split(pod.GenerateName, pod.Labels["pod-template-hash"])
	//deploymentName := strings.Trim(podNameSplitList[0], "-")

	//****pod-template-hash
	labelsSelector := labels.NewSelector()
	req1, err := labels.NewRequirement(
		"pod-template-hash",
		selection.Equals,
		[]string{pod_template_hash},
	)
	labelsSelector = labelsSelector.Add(*req1)
	ls := labelsSelector.String()

	//****spec.schedulerName=fargate-scheduler
	fselector := fields.OneTermNotEqualSelector(
		"spec.schedulerName",
		"fargate-scheduler",
	)
	fs := fselector.String()

	pods, err := k8sClientSet.CoreV1().Pods(pod.Namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: ls, FieldSelector: fs,
	})

	logger.Info("pod.Namespace: ", pod.Namespace)
	logger.Info("Total pod running in namespace: ", len(pods.Items))

	if err != nil {
		logger.Panic("error getting pod count: ", err.Error())
	}

	return len(pods.Items)
}

func getCustomHpaStrategy(pod corev1.Pod) (int, string) {

	podNameSplitList := strings.Split(pod.GenerateName, pod.Labels["pod-template-hash"])
	deploymentName := strings.Trim(podNameSplitList[0], "-")
	deploymentsClient := k8sClientSet.AppsV1().Deployments(pod.Namespace)
	deploymentData, getErr := deploymentsClient.Get(context.TODO(), deploymentName, metav1.GetOptions{})
	if getErr != nil {
		panic(fmt.Errorf("Failed to get latest version of Deployment: %v", getErr))
	}
	logger.Info("getCustomHpaStrategy >>>>>>>>> pod.Labels[custom-hpa-strategy]=%s", deploymentData.Annotations["custom-hpa-strategy"])

	entries := strings.Split(deploymentData.Annotations["custom-hpa-strategy"], ",")

	node_pod_max_count := 0
	var fargate_name string
	var e []string

	for i := 0; i < len(entries); i++ {

		e = strings.Split(entries[i], "=")
		if e[0] == "node-pod-max-count" {
			node_pod_max_count, _ = strconv.Atoi(e[1])
		} else if e[0] == "fargate-profile-name" {
			fargate_name = e[1]
		}
	}

	logger.Info("node_pod_max_count %s ", node_pod_max_count)
	logger.Info("fargate_name %s ", fargate_name)

	return node_pod_max_count, fargate_name
}
