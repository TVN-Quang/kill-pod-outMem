package config

import (
	"fmt"
	"log"
	"os"
	"script_restart/constants"
	"script_restart/utils/helper"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	IsLocal                   bool
	Namespace                 string
	CheckBy                   string
	PollInterval              int
	PodDeleteWaitTime         int
	PodDeletePercentThreshold float64
	LabelKey                  string
	ResourceCheckType         string
	ContainerName             string
	TimeToWaitPodCreate       time.Duration
	TimeToWaitPodReady        time.Duration
	DeploymentSelectorLabel   string
}

func GetConfig() (*Config, error) {
	err := godotenv.Load(".env")
	if err != nil {
		fmt.Println("Error loading .env file:", err)
	}
	// Lấy giá trị từ biến môi trường hoặc giá trị mặc định
	isLocal, err := strconv.ParseBool(os.Getenv("LOCAL"))
	if err != nil {
		log.Printf("Cảnh báo: Không thể chuyển đổi giá trị LOCAL thành bool. Trả về giá trị mặc định: false")
		isLocal = false
	}

	namespace := helper.GetEnv(constants.FieldNames.NAMESPACE, constants.NAMESPACE)
	checkBy := helper.GetEnv(constants.FieldNames.CHECK_BY, constants.LIMIT)
	pollInterval := helper.GetEnvAsInt(constants.FieldNames.POLL_INTERVAL, constants.POLL_INTERVAL)
	podDeleteWaitTime := helper.GetEnvAsInt("POD_DELETE_WAIT_TIME", 5)                      // Giá trị mặc định là 5
	podDeletePercentThreshold := helper.GetEnvAsFloat("POD_DELETE_PERCENT_THRESHOLD", 50.0) // Giá trị mặc định là 50.0
	labelKey := helper.GetEnv("LABEL_SELECTOR_HPA", "app")
	resourceCheckType := helper.GetEnv("RESOURCE_CHECK_TYPE", "container")
	containerName := helper.GetEnv("CONTAINER_NAME", "main")
	timeToWaitPodCreate := time.Duration(helper.GetEnvAsInt("TIME_TO_WAIT_POD_CREATE", 10))
	timeToWaitPodReady := time.Duration(helper.GetEnvAsInt("TIME_TO_WAIT_POD_READY", 10))
	deploymentSelectorLabel := helper.GetEnv("DEPLOYMENT_SELECTOR_LABEL", "app.kubernetes.io/name")
	// Trả về struct chứa các giá trị đã lấy
	config := &Config{
		IsLocal:                   isLocal,
		Namespace:                 namespace,
		CheckBy:                   checkBy,
		PollInterval:              pollInterval,
		PodDeleteWaitTime:         podDeleteWaitTime,
		PodDeletePercentThreshold: podDeletePercentThreshold,
		LabelKey:                  labelKey,
		ResourceCheckType:         resourceCheckType,
		ContainerName:             containerName,
		TimeToWaitPodCreate:       timeToWaitPodCreate,
		TimeToWaitPodReady:        timeToWaitPodReady,
		DeploymentSelectorLabel:   deploymentSelectorLabel,
	}

	return config, nil
}
