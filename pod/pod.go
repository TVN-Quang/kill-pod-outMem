package pod

import (
	"context"
	"fmt"
	"log"
	"script_restart/config"
	"script_restart/constants"
	"script_restart/utils/helper"
	"sort"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"
	"k8s.io/metrics/pkg/client/clientset/versioned"
)

var cfg, _ = config.GetConfig()

func CheckContainerMemoryUsage(pod *v1.Pod, podMetrics *v1beta1.PodMetrics, checkBy string) bool {
	// Khởi tạo biến lưu mức sử dụng bộ nhớ
	var memoryUsage int64
	containerName := cfg.ContainerName // Lấy tên container từ biến môi trường

	// Duyệt qua các container trong podMetrics để tìm container có tên "main" hoặc tên container từ môi trường
	for _, container := range podMetrics.Containers {
		if container.Name == containerName { // Kiểm tra theo tên container được chỉ định
			// Lấy mức sử dụng bộ nhớ của container
			memoryUsageStr := container.Usage.Memory().String() // Lấy giá trị sử dụng bộ nhớ
			memoryUsage = helper.ParseMemory(memoryUsageStr)    // Chuyển đổi về byte
			break
		}
	}

	if memoryUsage == 0 {
		log.Printf("[%s] Không tìm thấy container '%s' trong pod hoặc không có dữ liệu sử dụng bộ nhớ", pod.Name, containerName)
		return false
	}

	// Kiểm tra mức sử dụng bộ nhớ so với request hoặc limit
	for _, container := range pod.Spec.Containers {
		if container.Name == containerName { // Chỉ kiểm tra tài nguyên container được chỉ định
			resources := container.Resources
			var threshold int64

			if checkBy == "limit" {
				// Kiểm tra theo memory limit
				limit, ok := resources.Limits[v1.ResourceMemory]
				if ok && limit.Value() > 0 {
					threshold = limit.Value()
				} else {
					// Nếu không có memory limit, log thông báo và bỏ qua
					log.Printf("[%s] Container '%s' trong Pod  không có memory limit", pod.Name, container.Name)
					return false
				}
			} else if checkBy == "request" {
				// Kiểm tra theo memory request
				request, ok := resources.Requests[v1.ResourceMemory]
				if ok && request.Value() > 0 {
					threshold = request.Value()
				} else {
					// Nếu không có memory request, log thông báo và bỏ qua
					log.Printf("[%s] Container '%s' trong Pod không có memory request", pod.Name, container.Name)
					return false
				}
			}

			// So sánh mức sử dụng bộ nhớ thực tế với ngưỡng
			if threshold > 0 {
				percentageUsage := (float64(memoryUsage) / float64(threshold)) * 100
				podDeletePercentThreshold := cfg.PodDeletePercentThreshold
				// Kiểm tra xem mức sử dụng bộ nhớ đã vượt quá ngưỡng chưa
				if percentageUsage >= podDeletePercentThreshold {
					log.Printf("[%s] Container '%s' trong pod sử dụng bộ nhớ vượt mức %.2f%% của ngưỡng %d: %d bytes (%.2f%%)", pod.Name, container.Name, podDeletePercentThreshold, threshold, memoryUsage, percentageUsage)

					return true // Terminate container nếu vượt quá ngưỡng
				}
				log.Printf("[%s] Container trong Pod '%s' không cần bị terminate. Bộ nhớ sử dụng (%d bytes) dưới mức giới hạn (%d bytes).", pod.Name, container.Name, memoryUsage, threshold)
			}
		}
	}

	return false
}

func CheckPodMemoryUsage(pod *v1.Pod, podMetrics *v1beta1.PodMetrics, checkBy string) bool {

	var totalMemoryUsage, totalMemoryLimit, totalMemoryRequest int64

	// Duyệt qua các container trong podMetrics và tính tổng mức sử dụng bộ nhớ
	for _, container := range podMetrics.Containers {
		memoryUsageStr := container.Usage.Memory().String()
		memoryUsage := helper.ParseMemory(memoryUsageStr)
		totalMemoryUsage += memoryUsage // Cộng dồn mức sử dụng bộ nhớ của mỗi container
	}

	if totalMemoryUsage == 0 {
		log.Printf("[%s] Không tìm thấy dữ liệu sử dụng bộ nhớ trong pod", pod.Name)
		return false
	}
	// Kiểm tra mức sử dụng bộ nhớ của pod so với request hoặc limit của container
	for _, container := range pod.Spec.Containers {
		resources := container.Resources

		if checkBy == "limit" {
			// Kiểm tra theo memory limit
			limit, ok := resources.Limits[v1.ResourceMemory]
			if ok && limit.Value() > 0 {
				totalMemoryLimit += limit.Value()
			} else {
				log.Printf("[%s] Container '%s' trong Pod không có memory limit", pod.Name, container.Name)
				return false
			}
		} else if checkBy == "request" {
			// Kiểm tra theo memory request
			request, ok := resources.Requests[v1.ResourceMemory]
			if ok && request.Value() > 0 {
				totalMemoryRequest += request.Value()
			} else {
				log.Printf("[%s] Container '%s' trong Pod không có memory request", pod.Name, container.Name)
				return false
			}
		}
	}
	// Kiểm tra memory limit hoặc memory request
	var threshold int64
	if checkBy == "limit" {
		threshold = totalMemoryLimit
	} else if checkBy == "request" {
		threshold = totalMemoryRequest
	}
	// So sánh mức sử dụng bộ nhớ tổng của pod với ngưỡng
	if threshold > 0 {
		percentageUsage := (float64(totalMemoryUsage) / float64(threshold)) * 100
		podDeletePercentThreshold := cfg.PodDeletePercentThreshold
		if percentageUsage >= podDeletePercentThreshold {
			log.Printf("[%s] Pod sử dụng bộ nhớ vượt mức %d%% của ngưỡng %d: %d bytes (%.2f%%)", pod.Name, podDeletePercentThreshold, threshold, totalMemoryUsage, percentageUsage)
			return true
		}
		log.Printf("[%s] Pod không cần bị terminate. Bộ nhớ sử dụng (%d bytes) dưới mức giới hạn (%d bytes).", pod.Name, totalMemoryUsage, threshold)
	}
	return false
}

func GetPodMetrics(metricsClient *versioned.Clientset, pod v1.Pod) (*v1beta1.PodMetrics, error) {
	// Lấy metrics của pod
	podMetrics, err := metricsClient.MetricsV1beta1().PodMetricses(pod.Namespace).Get(context.TODO(), pod.Name, metav1.GetOptions{})
	if err != nil {
		log.Printf("[%s] Không thể lấy metrics cho pod: %v", pod.Name, err)
		return nil, err
	}
	return podMetrics, nil
}

func CheckMemoryUsage(clientset *kubernetes.Clientset, metricsClient *versioned.Clientset, pod v1.Pod, checkBy string) bool {
	podMetrics, err := GetPodMetrics(metricsClient, pod)
	if err != nil {
		log.Printf("[%s] Không thể lấy metrics cho pod: %v", pod.Name, err)
		return false
	}
	// log.Printf(helper.PodToJSON(podMetrics))
	var result bool
	if cfg.ResourceCheckType == "container" {
		result = CheckContainerMemoryUsage(&pod, podMetrics, checkBy)
	} else if cfg.ResourceCheckType == "pod" {
		result = CheckPodMemoryUsage(&pod, podMetrics, checkBy)
	} else {
		log.Fatal("RESOURCE_CHECK_TYPE phải là 'container' hoặc 'pod'")
	}

	if result {
		log.Printf("[%s] Container/Pod đã vượt quá ngưỡng tài nguyên.", pod.Name)
	}
	return result
}

func isPodReady(clientset *kubernetes.Clientset, podName, namespace string) bool {
	// Lấy thông tin pod
	pod, err := clientset.CoreV1().Pods(namespace).Get(context.TODO(), podName, metav1.GetOptions{})
	if err != nil {
		log.Printf("[%s] Không thể lấy thông tin pod: %v", podName, err)
		return false
	}

	// Kiểm tra điều kiện "Ready"
	for _, cond := range pod.Status.Conditions {
		if cond.Type == "Ready" && cond.Status == "True" {
			return true
		}
	}
	return false
}

func CheckForPodReady(clientset *kubernetes.Clientset, namespace string, pod *v1.Pod, oldPodName string) bool {
	var result = false
	maxRetries := 3
	retries := 0
	// Kiểm tra xem Pod đã sẵn sàng chưa, tối đa 3 lần thử
	for retries < maxRetries {
		if isPodReady(clientset, pod.Name, pod.Namespace) {
			log.Printf("[%s] Pod %s đã sẵn sàng và có thể nhận request!", oldPodName, pod.Name)
			result = true
			break
		} else {
			log.Printf("[%s] Pod %s chưa sẵn sàng, thử lại lần %d", oldPodName, pod.Name, retries+1)
			retries++
			// Nếu chưa sẵn sàng, đợi trước khi thử lại
			time.Sleep(cfg.TimeToWaitPodReady * time.Second)
		}
	}
	// Nếu sau 3 lần thử mà Pod vẫn chưa sẵn sàng, ghi lại thông báo lỗi
	if retries == maxRetries {
		log.Printf("[%s] Pod %s vẫn chưa sẵn sàng sau %d lần thử.", oldPodName, pod.Name, maxRetries)
	}
	return result
}

func GetNewPod(clientset *kubernetes.Clientset, namespace string, labelSelector string, podNames map[string]struct{}) (*v1.Pod, error) {
	// Lấy danh sách Pods với label selector
	var pod *v1.Pod
	sleepDuration := cfg.TimeToWaitPodCreate * time.Second
	for true {
		pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{
			LabelSelector: labelSelector,
		})

		if err != nil {
			log.Printf("GetNewPod [%s] Lỗi khi lấy danh sách Pods: %v", labelSelector, err)
			return nil, err
		}

		// Sắp xếp danh sách Pods theo thời gian tạo (creation timestamp) giảm dần
		sort.SliceStable(pods.Items, func(i, j int) bool {
			return pods.Items[i].CreationTimestamp.After(pods.Items[j].CreationTimestamp.Time)
		})
		if len(pods.Items) > 0 {
			pod = &pods.Items[0]
			if _, exists := podNames[pod.Name]; exists {
				log.Printf("[%s] Pod mới chưa được tạo. Thử lại sau %d", pod.Name, sleepDuration.Seconds())
				time.Sleep(sleepDuration)
				sleepDuration *= 2
			} else {
				podNames[pod.Name] = struct{}{} // Thêm Pod mới
				break
			}
		}
	}
	return pod, nil
}

func GetPods(clientset *kubernetes.Clientset, namespace string) *v1.PodList {
	// Lấy danh sách pods từ namespace
	pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Fatalf(constants.ERR_GET_PODS, err)
	}
	return pods
}

func UpdateLabelValue(clientset *kubernetes.Clientset, pod *v1.Pod, labelKey string, newLabelValue string) error {
	// Check if the pod has the label, and update its value
	if pod.Labels == nil {
		pod.Labels = make(map[string]string)
	}
	pod.Labels[labelKey] = newLabelValue
	log.Printf("[%s] Label %s updated to %s for pod", pod.Name, labelKey, newLabelValue)

	// Update the pod with the new label value
	_, err := clientset.CoreV1().Pods(pod.Namespace).Update(context.TODO(), pod, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("[%s] failed to update pod: %v", pod.Name, err)
	}

	return nil
}

func DeletePodAndWait(clientset *kubernetes.Clientset, podNamespace, podName string) {
	err := clientset.CoreV1().Pods(podNamespace).Delete(context.TODO(), podName, metav1.DeleteOptions{})
	if err != nil {
		log.Printf("[%s] Không thể xóa pod cũ: %v", podName, err)
		return
	}
	log.Printf("[%s] Đã gửi yêu cầu xóa pod", podName)

	// Vòng lặp để kiểm tra trạng thái của pod
	for {
		_, err := clientset.CoreV1().Pods(podNamespace).Get(context.TODO(), podName, metav1.GetOptions{})
		if err != nil {
			// Nếu pod không tồn tại nữa, coi như đã bị terminate
			log.Printf("[%s] Pod đã bị terminate.", podName)
			break
		}
		log.Printf("[%s] Pod vẫn đang tồn tại. Chờ...", podName)
		podDeleteWaitTime := helper.GetEnvAsInt("POD_DELETE_WAIT_TIME", 5)
		time.Sleep(time.Duration(podDeleteWaitTime) * time.Second) // Chờ một khoảng thời gian trước khi kiểm tra lại
	}
}

func GetDeployment(clientset *kubernetes.Clientset, pod *v1.Pod) (*appsv1.Deployment, error) {
	// Kiểm tra owner references của pod
	for _, owner := range pod.OwnerReferences {
		// Kiểm tra xem Owner có phải là một Deployment không
		if owner.Kind == "ReplicaSet" {
			// Lấy tên ReplicaSet từ owner
			replicaSetName := owner.Name

			// Xóa phần hash cuối của tên ReplicaSet để lấy tên Deployment
			lastDashIndex := strings.LastIndex(replicaSetName, "-")
			if lastDashIndex == -1 {
				return nil, fmt.Errorf("Không tìm thấy dấu '-' trong tên ReplicaSet")
			}
			
			// Lấy tên Deployment bằng cách cắt chuỗi từ đầu đến dấu '-' cuối
			deploymentName := replicaSetName[:lastDashIndex]

			// Lấy Deployment từ API server
			deploymentClient := clientset.AppsV1().Deployments(pod.Namespace)
			deployment, err := deploymentClient.Get(context.TODO(), deploymentName, metav1.GetOptions{})
			if err != nil {
				return nil, fmt.Errorf("không thể lấy Deployment từ ReplicaSet: %v", err)
			}

			// Trả về Deployment
			return deployment, nil
		}
	}
	return nil, fmt.Errorf("[%s] Pod không thuộc về bất kỳ Deployment nào", pod.Name)
}

func ScaleDeployment(clientset *kubernetes.Clientset, namespace string, deploymentName string, podName string) error {
	// Lấy Deployment hiện tại
	deploymentClient := clientset.AppsV1().Deployments(namespace)
	deployment, err := deploymentClient.Get(context.Background(), deploymentName, metav1.GetOptions{})
	if err != nil {
		log.Printf("[%s] Không thể lấy deployment: %v", podName, err)
		return fmt.Errorf("không thể lấy deployment: %v", err)
	}

	// Tăng số lượng replicas lên 1
	newReplicas := *deployment.Spec.Replicas + 1

	// Tạo bản vá (patch) cho số lượng replicas
	patch := []byte(fmt.Sprintf(`{"spec": {"replicas": %d}}`, newReplicas))

	// Sử dụng PATCH để cập nhật Deployment
	_, err = deploymentClient.Patch(context.Background(), deploymentName, types.StrategicMergePatchType, patch, metav1.PatchOptions{})
	if err != nil {
		log.Printf("[%s] Không thể PATCH deployment: %v", podName, err)
		return fmt.Errorf("không thể PATCH deployment: %v", err)
	}

	log.Printf("[%s] Deployment %s đã được scale lên %d replicas", podName, deploymentName, newReplicas)
	return nil
}
