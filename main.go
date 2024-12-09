package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"time"

	"script_restart/constants" // Import constants
	"script_restart/utils/helper"

	v2 "k8s.io/api/autoscaling/v2"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/metrics/pkg/client/clientset/versioned"

	// "k8s.io/metrics/pkg/client/clientset/versioned"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var podNames []string

func isPodReady(clientset *kubernetes.Clientset, podName, namespace string) bool {
	// Lấy thông tin pod
	pod, err := clientset.CoreV1().Pods(namespace).Get(context.TODO(), podName, metav1.GetOptions{})
	if err != nil {
		log.Printf("Không thể lấy thông tin pod %s: %v", podName, err)
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

func checkForPodReady(clientset *kubernetes.Clientset, namespace string, pollInterval int, labelSelector string) bool {
	var result = false
	maxExistRetries := 3
	existRetries := 0
	// Lấy danh sách Pods với label selector
	for existRetries < maxExistRetries {
		pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{
			LabelSelector: labelSelector,
		})

		if err != nil {
			log.Printf("Lỗi khi lấy danh sách Pods: %v", err)
			return result
		}

		// Sắp xếp danh sách Pods theo thời gian tạo (creation timestamp) giảm dần
		sort.SliceStable(pods.Items, func(i, j int) bool {
			return pods.Items[i].CreationTimestamp.After(pods.Items[j].CreationTimestamp.Time)
		})
		if len(pods.Items) > 0 {
			pod := pods.Items[0]
			podExists := false
			for _, name := range podNames {
				if pod.Name == name {
					podExists = true
					break
				}
			}
			if !podExists {
				podNames = append(podNames, pod.Name)
				maxRetries := 3
				retries := 0
				// Kiểm tra xem Pod đã sẵn sàng chưa, tối đa 3 lần thử
				for retries < maxRetries {
					if isPodReady(clientset, pod.Name, pod.Namespace) {
						log.Printf("Pod %s đã sẵn sàng và có thể nhận request!", pod.Name)
						result = true
						break
					} else {
						log.Printf("Pod %s chưa sẵn sàng, thử lại lần %d", pod.Name, retries+1)
						retries++
						// Nếu chưa sẵn sàng, đợi trước khi thử lại
						time.Sleep(time.Duration(30) * time.Second)
					}
				}

				// Nếu sau 3 lần thử mà Pod vẫn chưa sẵn sàng, ghi lại thông báo lỗi
				if retries == maxRetries {
					log.Printf("Pod %s vẫn chưa sẵn sàng sau %d lần thử.", pod.Name, maxRetries)
				}
				break
			} else {
				// Nếu pod đã tồn tại trong danh sách podNames, đợi 10 giây và thử lại 3 lần mà không kiểm tra lại
				log.Printf("Pod mới chưa được tạo, thử lại sau 10 giây.")
				existRetries++
				time.Sleep(10 * time.Second)
			}
			if existRetries == maxExistRetries {
				log.Printf("Pod mới vẫn chưa được tạo sau %d lần thử.", maxExistRetries)
				return result
			}
		} else {
			log.Printf("Không có pod nào trong namespace %s với label %s.", namespace, labelSelector)
			break
		}
	}
	return result
}

// Hàm trợ giúp để tìm container trong spec của pod
func getContainerFromSpec(pod v1.Pod, containerName string) *v1.Container {
	for _, container := range pod.Spec.Containers {
		if container.Name == containerName {
			return &container
		}
	}
	return nil
}

func checkMemoryUsage(clientset *kubernetes.Clientset, metricsClient *versioned.Clientset, pod v1.Pod, checkBy string) bool {
	// Lấy metrics của pod
	podMetrics, err := metricsClient.MetricsV1beta1().PodMetricses(pod.Namespace).Get(context.TODO(), pod.Name, metav1.GetOptions{})
	if err != nil {
		log.Printf("Không thể lấy metrics cho pod %s: %v", pod.Name, err)
		return false
	}

	log.Printf(helper.PodToJSON(podMetrics))
	// Tìm container 'main'
	var memoryUsage int64
	for _, container := range podMetrics.Containers {
		if container.Name == "main" { // Chỉ kiểm tra container 'main'
			memoryUsageStr := container.Usage.Memory().String() // Lấy giá trị sử dụng bộ nhớ
			memoryUsage = helper.ParseMemory(memoryUsageStr)    // Chuyển đổi về byte
			break
		}
	}

	if memoryUsage == 0 {
		log.Printf("Không tìm thấy container 'main' trong pod %s hoặc không có dữ liệu sử dụng bộ nhớ", pod.Name)
		return false
	}

	// Kiểm tra mức sử dụng bộ nhớ so với request hoặc limit
	for _, container := range pod.Spec.Containers {
		if container.Name == "main" { // Chỉ kiểm tra tài nguyên container 'main'
			resources := container.Resources
			var threshold int64

			if checkBy == "limit" {
				// Kiểm tra theo memory limit
				limit, ok := resources.Limits[v1.ResourceMemory]
				if ok && limit.Value() > 0 {
					threshold = limit.Value()
				} else {
					// Nếu không có memory limit, log thông báo và bỏ qua
					log.Printf("Container '%s' trong Pod '%s' không có memory limit", container.Name, pod.Name)
					return false
				}
			} else if checkBy == "request" {
				// Kiểm tra theo memory request
				request, ok := resources.Requests[v1.ResourceMemory]
				if ok && request.Value() > 0 {
					threshold = request.Value()
				} else {
					// Nếu không có memory request, log thông báo và bỏ qua
					log.Printf("Container '%s' trong Pod '%s' không có memory request", container.Name, pod.Name)
					return false
				}
			}

			// So sánh mức sử dụng bộ nhớ thực tế với ngưỡng
			if threshold > 0 {
				percentageUsage := (float64(memoryUsage) / float64(threshold)) * 100
				percentThreshold, _ := strconv.ParseFloat(os.Getenv("PERCENT_THRESHOLD"), 64)
				// Kiểm tra xem mức sử dụng bộ nhớ đã vượt quá 80% của threshold chưa
				if percentageUsage >= percentThreshold {
					log.Printf("Container '%s' trong pod %s sử dụng bộ nhớ vượt mức %d%% của ngưỡng %d: %d bytes (%.2f%%)", container.Name, pod.Name, percentThreshold, threshold, memoryUsage, percentageUsage)
					return true // Terminate container nếu vượt quá ngưỡng 80%
				}
				log.Printf("Container 'main' trong Pod '%s' không cần bị terminate. Bộ nhớ sử dụng (%d bytes) dưới mức giới hạn (%d bytes).", pod.Name, memoryUsage, threshold)
			}
		}
	}
	return false
}

func scaleHPA(clientset *kubernetes.Clientset, namespace string, labelSelector string) (*v2.HorizontalPodAutoscaler, int32, error) {
	// Lấy HPA hiện tại
	hpaList, err := clientset.AutoscalingV2().HorizontalPodAutoscalers(namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: labelSelector, // Sử dụng label selector để lọc HPA
	})

	if err != nil {
		return nil, 0, fmt.Errorf("không thể lấy HPA: %v", err)
	}
	// Kiểm tra danh sách HPA có phần tử nào không
	if len(hpaList.Items) == 0 {
		return nil, 0, fmt.Errorf("không tìm thấy HPA nào với labelSelector: %s", labelSelector)
	}
	// Lấy phần tử đầu tiên
	var minReplica int32
	hpa := hpaList.Items[0]
	if hpa.Spec.MinReplicas != nil {
		minReplica = *hpa.Spec.MinReplicas
	}
	if hpa.Status.CurrentReplicas == hpa.Spec.MaxReplicas {
		hpa.Spec.MaxReplicas = hpa.Spec.MaxReplicas + 1
	}
	var newMinReplicas = hpa.Status.CurrentReplicas + 1

	// Cập nhật lại HPA với minReplicas mới
	hpa.Spec.MinReplicas = &newMinReplicas
	updatedHPA, err := clientset.AutoscalingV2().HorizontalPodAutoscalers(namespace).Update(context.TODO(), &hpa, metav1.UpdateOptions{})
	if err != nil {
		return nil, minReplica, fmt.Errorf("không thể cập nhật HPA %s: %v", hpa.Name, err)
	}

	log.Printf("Đã cập nhật HPA %s với minReplicas = %d", hpa.Name, newMinReplicas)
	return updatedHPA, minReplica, nil
}

func deletePodAndWait(clientset *kubernetes.Clientset, podNamespace, podName string) {
	err := clientset.CoreV1().Pods(podNamespace).Delete(context.TODO(), podName, metav1.DeleteOptions{})
	if err != nil {
		log.Printf("Không thể xóa pod cũ %s: %v", podName, err)
		return
	}
	log.Printf("Đã gửi yêu cầu xóa pod: %s", podName)

	// Vòng lặp để kiểm tra trạng thái của pod
	for {
		_, err := clientset.CoreV1().Pods(podNamespace).Get(context.TODO(), podName, metav1.GetOptions{})
		if err != nil {
			// Nếu pod không tồn tại nữa, coi như đã bị terminate
			log.Printf("Pod %s đã bị terminate.", podName)
			break
		}
		log.Printf("Pod %s vẫn đang tồn tại. Chờ...", podName)
		waitTimeSeconds := helper.GetEnvAsInt("TIME_DELETE_WAITING", 5)
		time.Sleep(time.Duration(waitTimeSeconds) * time.Second) // Chờ một khoảng thời gian trước khi kiểm tra lại
	}
}

func processPod(clientset *kubernetes.Clientset, metricsClient *versioned.Clientset, pod *v1.Pod, checkBy string, pollInterval int) {
	// Log thời gian và thông tin pod đang được xử lý
	log.Printf("Xử lý pod: %s", pod.Name)
	namespace := helper.GetEnv(constants.FieldNames.NAMESPACE, constants.NAMESPACE)
	// Kiểm tra mức sử dụng bộ nhớ
	shouldTerminate := checkMemoryUsage(clientset, metricsClient, *pod, checkBy)
	if !shouldTerminate {
		return
	}
	// Tạo pod mới thay thế pod cũ
	updatedHPA, minReplica, err := scaleHPA(clientset, namespace, fmt.Sprintf("app=%s", pod.Labels["app"]))
	// Chờ pod mới sẵn sàng trước khi xóa pod cũ
	checkForPodReady(clientset, namespace, pollInterval, fmt.Sprintf("app=%s", pod.Labels["app"]))

	hpa, err := clientset.AutoscalingV2().HorizontalPodAutoscalers(namespace).Get(context.TODO(), updatedHPA.Name, metav1.GetOptions{})
	hpa.Spec.MinReplicas = &minReplica
	// Cập nhật lại HPA về giá trị ban đầu
	_, err = clientset.AutoscalingV2().HorizontalPodAutoscalers(namespace).Update(context.TODO(), hpa, metav1.UpdateOptions{})
	if err != nil {
		fmt.Errorf("không thể cập nhật lại HPA %s với minReplicas ban đầu: %v", updatedHPA.Name, err)
		return
	}
	log.Printf("Đã cập nhật lại HPA %s với minReplicas ban đầu = %d", updatedHPA.Name, minReplica)

	// Xóa pod cũ
	deletePodAndWait(clientset, pod.Namespace, pod.Name)
}

func main() {
	// Đọc các biến môi trường và các giá trị mặc định
	// err := godotenv.Load(".env")
	// if err != nil {
	// 	fmt.Println("Error loading .env file:", err)
	// 	return
	// }

	// Lấy giá trị biến môi trường
	isLocal, err := strconv.ParseBool(os.Getenv("LOCAL"))

	namespace := helper.GetEnv(constants.FieldNames.NAMESPACE, constants.NAMESPACE)
	checkBy := helper.GetEnv(constants.FieldNames.CHECK_BY, constants.LIMIT)
	pollInterval := helper.GetEnvAsInt(constants.FieldNames.POLL_INTERVAL, constants.POLL_INTERVAL)

	var config *rest.Config

	if isLocal {
		config, _ = clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	} else {
		config, _ = rest.InClusterConfig()
	}

	if err != nil {
		log.Fatalf("Không thể tải cấu hình Kubernetes: %v", err)
	}

	// Khởi tạo Kubernetes client
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Không thể tạo Kubernetes client: %v", err)
	}
	metricsClient, err := versioned.NewForConfig(config)
	if err != nil {
		log.Fatalf("Error creating metrics client: %v", err)
	}

	// Thời gian lặp lại
	timeInterval := helper.GetEnvAsInt("TIME_INTERVAL", 30)
	ticker := time.NewTicker(time.Duration(timeInterval) * time.Second)
	defer ticker.Stop()

	// Lấy danh sách pods từ namespace
	pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Fatalf("Không thể lấy danh sách pods: %v", err)
	}
	// Duyệt qua các pod và lấy tên pod, để kiểm tra pod mới đã ready chưa khi tăng hpa
	for _, pod := range pods.Items {
		podNames = append(podNames, pod.Name)
	}

	// Kiểm tra và xử lý từng pod
	for _, pod := range pods.Items {
		processPod(clientset, metricsClient, &pod, checkBy, pollInterval)
	}
}
