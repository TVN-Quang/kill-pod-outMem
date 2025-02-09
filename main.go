package main

import (
	"fmt"
	"log"

	"script_restart/config"
	"script_restart/constants" // Import constants
	hpaObj "script_restart/hpa"
	podObj "script_restart/pod"
	"script_restart/utils/helper"

	v1 "k8s.io/api/core/v1"
	"k8s.io/metrics/pkg/client/clientset/versioned"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var cfg, _ = config.GetConfig()

func processPod(clientset *kubernetes.Clientset, metricsClient *versioned.Clientset, pod *v1.Pod, checkBy string, pollInterval int, podNames map[string]struct{}) {
	// Log thời gian và thông tin pod đang được xử lý
	log.Printf("Xử lý pod: %s", pod.Name)
	namespace := helper.GetEnv(constants.FieldNames.NAMESPACE, constants.NAMESPACE)
	labelKey := cfg.LabelKey

	labelValue := pod.Labels[labelKey]
	if labelValue == "" {
		log.Printf("[%s]Pod không có label phù hợp với key: %s", pod.Name, labelKey)
		return
	}
	labelSelector := fmt.Sprintf("%s=%s", labelKey, labelValue)

	// Kiểm tra mức sử dụng bộ nhớ
	shouldTerminate := podObj.CheckMemoryUsage(clientset, metricsClient, *pod, checkBy)
	if !shouldTerminate {
		return
	}
	// Tạo pod mới thay thế pod cũ bằng cách tăng HPA
	hpa, err := hpaObj.GetHPA(clientset, namespace, labelSelector)
	if err != nil {
		return
	}

	// disableScaleDown := true
	// err = hpaObj.SetHpaScaleDown(clientset, namespace, hpa.Name, disableScaleDown)
	// updatedHPA, oldMinReplica, isChangeMax, err := hpaObj.ScaleHPA(clientset, hpa, namespace, true)
	if err != nil {
		return
	}

	deployment, err := podObj.GetDeployment(clientset, pod)
	podObj.ScaleDeployment(clientset, pod.Namespace, deployment.Name, pod.Name)
	// Chờ pod mới sẵn sàng trước khi xóa pod cũ
	newPod, err := podObj.GetNewPod(clientset, namespace, labelSelector, podNames) //có thời gian đợi và thử lại.
	if err != nil {
		return
	}
	podObj.CheckForPodReady(clientset, namespace, newPod, pod.Name) //có thời gian đợi và thử lại
	disableScaleUp := true
	hpaObj.SetHpaScaleUp(clientset, namespace, hpa.Name, disableScaleUp)
	err = podObj.UpdateLabelValue(clientset, pod, cfg.DeploymentSelectorLabel, constants.SHUTDOWN)
	hpaObj.DecreaseDeploymentReplicas(clientset, hpa, pod.Name)
	hpaObj.SetHpaScaleUp(clientset, namespace, hpa.Name, !disableScaleUp)
	if err != nil {
		log.Fatalf("[%s] Error: %v", pod.Name, err)
	}
	// err = hpaObj.SetHpaScaleDown(clientset, namespace, hpa.Name, !disableScaleDown)
	// if err != nil {
	// 	log.Fatalf("[%s] Error: %v", pod.Name, err)
	// }
	podObj.DeletePodAndWait(clientset, namespace, pod.Name) // Có đợi 5s
}

func main() {
	// Đọc các biến môi trường và các giá trị mặc định

	var k8sConfig *rest.Config

	if cfg.IsLocal {
		k8sConfig, _ = clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	} else {
		k8sConfig, _ = rest.InClusterConfig()
	}

	// Khởi tạo Kubernetes client
	clientset, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		log.Fatalf(constants.ERR_K8S_CLIENT_CREATION, err)
	}

	metricsClient, err := versioned.NewForConfig(k8sConfig)
	if err != nil {
		log.Fatalf(constants.ERR_METRICS_CLIENT, err)
	}

	pods := podObj.GetPods(clientset, cfg.Namespace)

	podNames := make(map[string]struct{})
	for _, pod := range pods.Items {
		if _, exists := podNames[pod.Name]; !exists {
			podNames[pod.Name] = struct{}{}
		}
	}
	// var wg sync.WaitGroup
	// Kiểm tra và xử lý từng pod
	for _, pod := range pods.Items {
		// wg.Add(1)
		// go func(pod v1.Pod) {
		// 	// Đảm bảo giảm số lượng khi goroutine hoàn thành
		// 	defer wg.Done()
		// 	processPod(clientset, metricsClient, &pod, cfg.CheckBy, cfg.PollInterval, podNames)
		// }(pod)
		processPod(clientset, metricsClient, &pod, cfg.CheckBy, cfg.PollInterval, podNames)
	}
	// wg.Wait()
}
