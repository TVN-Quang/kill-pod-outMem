package hpa

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	v2 "k8s.io/api/autoscaling/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

func GetHPA(clientset *kubernetes.Clientset, namespace, labelSelector string) (*v2.HorizontalPodAutoscaler, error) {
	hpaList, err := clientset.AutoscalingV2().HorizontalPodAutoscalers(namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, fmt.Errorf("không thể lấy HPA: %v", err)
	}

	if len(hpaList.Items) == 0 {
		return nil, fmt.Errorf("không tìm thấy HPA nào với labelSelector: %s", labelSelector)
	}

	return &hpaList.Items[0], nil
}

func UpdateHPA(clientset *kubernetes.Clientset, namespace string, hpa *v2.HorizontalPodAutoscaler) (*v2.HorizontalPodAutoscaler, error) {
	updatedHPA, err := clientset.AutoscalingV2().HorizontalPodAutoscalers(namespace).Update(context.TODO(), hpa, metav1.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("không thể cập nhật HPA %s: %v", hpa.Name, err)
	}
	return updatedHPA, nil
}

func ScaleHPA(clientset *kubernetes.Clientset, hpa *v2.HorizontalPodAutoscaler, namespace string, scaleUp bool) (*v2.HorizontalPodAutoscaler, int32, bool, error) {
	var minReplica int32
	var isChangeMax bool = false
	if hpa.Spec.MinReplicas != nil {
		minReplica = *hpa.Spec.MinReplicas
	}

	var newMinReplicas int32
	if scaleUp {
		// Tăng replicas
		if hpa.Status.CurrentReplicas == hpa.Spec.MaxReplicas {
			hpa.Spec.MaxReplicas++
			isChangeMax = true
		}
		newMinReplicas = hpa.Status.CurrentReplicas + 1
	}

	hpa.Spec.MinReplicas = &newMinReplicas
	updatedHPA, err := UpdateHPA(clientset, namespace, hpa)
	if err != nil {
		return nil, minReplica, isChangeMax, err
	}

	log.Printf("Đã cập nhật HPA %s với minReplicas = %d", hpa.Name, newMinReplicas)
	return updatedHPA, minReplica, isChangeMax, nil
}

func ResetHPA(clientset *kubernetes.Clientset, namespace string, hpa *v2.HorizontalPodAutoscaler, oldMinReplica int32, isChangeMax bool) {
	// Cập nhật lại HPA về giá trị ban đầu
	freshHPA, err := clientset.AutoscalingV2().HorizontalPodAutoscalers(namespace).Get(context.TODO(), hpa.Name, metav1.GetOptions{})
	if err != nil {
		fmt.Errorf("không thể lấy HPA: %v", err)
	}
	freshHPA.Spec.MinReplicas = &oldMinReplica
	patch := map[string]interface{}{
		"spec": map[string]interface{}{
			"minReplicas": &oldMinReplica,
		},
	}

	if isChangeMax {
		// Lấy giá trị maxReplicas hiện tại (ví dụ lấy từ một biến đã có sẵn)
		currentMaxReplicas := hpa.Spec.MaxReplicas // Giả sử bạn đã có giá trị của oldMaxReplica
		patch["spec"].(map[string]interface{})["maxReplicas"] = currentMaxReplicas - 1
	}
	patchBytes, err := json.Marshal(patch)
	updatedHPA, err := clientset.AutoscalingV2().HorizontalPodAutoscalers(namespace).Patch(context.TODO(), hpa.Name, types.MergePatchType, patchBytes, metav1.PatchOptions{})
	if err != nil {
		fmt.Errorf("không thể cập nhật lại HPA %s với minReplicas ban đầu: %v", updatedHPA.Name, err)
		return
	}
	log.Printf("Đã cập nhật lại HPA %s với minReplicas ban đầu = %d", updatedHPA.Name, oldMinReplica)
}

func SetHpaScaleDown(clientset *kubernetes.Clientset, namespace string, hpaName string, disableScaleDown bool) error {
	var patchData map[string]interface{}

	// Nếu disableScaleDown là true, vô hiệu hóa scale down, ngược lại thì bật lại scale down
	if disableScaleDown {
		disabledPolicy := v2.DisabledPolicySelect
		patchData = map[string]interface{}{
			"spec": map[string]interface{}{
				"behavior": map[string]interface{}{
					"scaleDown": map[string]interface{}{
						"selectPolicy": disabledPolicy, // Vô hiệu hóa scale down
					},
				},
			},
		}
	} else {
		// Cho phép scale down bình thường
		patchData = map[string]interface{}{
			"spec": map[string]interface{}{
				"behavior": map[string]interface{}{
					"scaleDown": map[string]interface{}{
						"selectPolicy": nil,
					},
				},
			},
		}
	}

	// Chuyển patch thành kiểu byte
	patchBytes, err := json.Marshal(patchData)
	if err != nil {
		return fmt.Errorf("chuyển patch thành byte thất bại: %v", err)
	}

	// Thực hiện patch
	_, err = clientset.AutoscalingV2().HorizontalPodAutoscalers(namespace).Patch(context.TODO(), hpaName, types.MergePatchType, patchBytes, metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("Cập nhật HPA thất bại: %v", err)
	}

	return nil
}

func SetHpaScaleUp(clientset *kubernetes.Clientset, namespace string, hpaName string, disableScaleUp bool) error {
	var patchData map[string]interface{}

	// Nếu disableScaleUp là true, vô hiệu hóa scale down, ngược lại thì bật lại scale down
	if disableScaleUp {
		disabledPolicy := v2.DisabledPolicySelect
		patchData = map[string]interface{}{
			"spec": map[string]interface{}{
				"behavior": map[string]interface{}{
					"scaleUp": map[string]interface{}{
						"selectPolicy": disabledPolicy, // Vô hiệu hóa scale down
					},
				},
			},
		}
	} else {
		// Cho phép scale down bình thường
		patchData = map[string]interface{}{
			"spec": map[string]interface{}{
				"behavior": map[string]interface{}{
					"scaleUp": map[string]interface{}{
						"selectPolicy": nil,
					},
				},
			},
		}
	}

	// Chuyển patch thành kiểu byte
	patchBytes, err := json.Marshal(patchData)
	if err != nil {
		return fmt.Errorf("chuyển patch thành byte thất bại: %v", err)
	}

	// Thực hiện patch
	_, err = clientset.AutoscalingV2().HorizontalPodAutoscalers(namespace).Patch(context.TODO(), hpaName, types.MergePatchType, patchBytes, metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("Cập nhật HPA thất bại: %v", err)
	}

	return nil
}

func DecreaseDeploymentReplicas(clientset *kubernetes.Clientset, hpa *v2.HorizontalPodAutoscaler, podName string) {
	deployment, err := clientset.AppsV1().Deployments(hpa.Namespace).Get(context.TODO(), hpa.Spec.ScaleTargetRef.Name, metav1.GetOptions{})
	if err != nil {
		log.Fatalf("[%s] Lỗi khi lấy Deployment: %s", podName, err.Error())
	}

	// Cập nhật số lượng replicas
	// Giảm số lượng replicas (đảm bảo không có lỗi khi con trỏ là nil)
	if deployment.Spec.Replicas != nil && *deployment.Spec.Replicas > *hpa.Spec.MinReplicas {
		// Giảm 1 replica
		newReplicas := *deployment.Spec.Replicas - 1
		deployment.Spec.Replicas = &newReplicas
	} else {
		log.Printf("[Pod: %s] Không thể giảm số lượng replicas nữa, số lượng replicas đã đạt mức tối thiểu", podName)
		return
	}

	// Cập nhật Deployment với replicas mới
	updatedDeployment, err := clientset.AppsV1().Deployments(hpa.Namespace).Update(context.TODO(), deployment, metav1.UpdateOptions{})
	if err != nil {
		log.Fatalf("[%s] Lỗi khi cập nhật số lượng replica Deployment: %s", podName, err.Error())
	}
	log.Printf("[Pod: %s] Đã cập nhật Deployment %s\n", podName, updatedDeployment.Name)
}
