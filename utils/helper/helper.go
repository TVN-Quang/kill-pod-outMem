package helper

import (
	"encoding/json"
	"log"
	"os"
	"strconv"
	"strings"

	v1 "k8s.io/api/core/v1"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"
)

// Hàm lấy giá trị từ biến môi trường hoặc trả về giá trị mặc định nếu không có biến môi trường
func GetEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// Hàm lấy giá trị từ biến môi trường dưới dạng số nguyên, trả về giá trị mặc định nếu không có biến môi trường hoặc có lỗi
func GetEnvAsInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	intValue, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return intValue
}

func GetEnvAsFloat(key string, defaultValue float64) float64 {
	// Lấy giá trị của biến môi trường
	envValue := os.Getenv(key)
	if envValue == "" {
		// Nếu không có giá trị trong môi trường, trả về giá trị mặc định
		return defaultValue
	}

	// Chuyển đổi giá trị từ chuỗi thành float64
	value, err := strconv.ParseFloat(envValue, 64)
	if err != nil {
		// Nếu không thể chuyển đổi, ghi log và trả về giá trị mặc định
		log.Printf("Cảnh báo: Không thể chuyển đổi giá trị '%s' từ biến môi trường %s thành float64. Trả về giá trị mặc định: %v", envValue, key, defaultValue)
		return defaultValue
	}

	return value
}

// parseMemory chuyển đổi giá trị memory từ dạng string (e.g., "123456Ki") sang byte
func ParseMemory(memoryStr string) int64 {
	// Xử lý chuỗi trống
	if memoryStr == "" {
		log.Println("Memory string is empty")
		return 0
	}

	// Đơn vị mặc định
	multiplier := int64(1)

	// Xử lý hậu tố
	switch {
	case strings.HasSuffix(memoryStr, "Ki"):
		multiplier = 1024
		memoryStr = strings.TrimSuffix(memoryStr, "Ki")
	case strings.HasSuffix(memoryStr, "Mi"):
		multiplier = 1024 * 1024
		memoryStr = strings.TrimSuffix(memoryStr, "Mi")
	case strings.HasSuffix(memoryStr, "Gi"):
		multiplier = 1024 * 1024 * 1024
		memoryStr = strings.TrimSuffix(memoryStr, "Gi")
	case strings.HasSuffix(memoryStr, "K"):
		multiplier = 1000
		memoryStr = strings.TrimSuffix(memoryStr, "K")
	case strings.HasSuffix(memoryStr, "M"):
		multiplier = 1000 * 1000
		memoryStr = strings.TrimSuffix(memoryStr, "M")
	case strings.HasSuffix(memoryStr, "G"):
		multiplier = 1000 * 1000 * 1000
		memoryStr = strings.TrimSuffix(memoryStr, "G")
	}

	// Chuyển đổi giá trị số
	value, err := strconv.ParseInt(memoryStr, 10, 64)
	if err != nil {
		log.Printf("Không thể parse giá trị memory '%s': %v", memoryStr, err)
		return 0
	}

	return value * multiplier
}

func PodToJSON(podMetrics *v1beta1.PodMetrics) string {
	podJSON, err := json.Marshal(*podMetrics)
	if err != nil {
		log.Fatalf("Error marshaling pod: %v", err)
	}
	return string(podJSON)
}

func PodInfo(pod *v1.Pod) string {
	podJSON, err := json.Marshal(pod)
	if err != nil {
		log.Fatalf("Error marshaling pod: %v", err)
	}
	return string(podJSON)
}
