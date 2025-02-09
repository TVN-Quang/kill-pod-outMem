package constants

type FIELD_NAME struct {
	NAMESPACE     string
	CHECK_BY      string
	POLL_INTERVAL string
}

// Constants chỉ dùng cho các giá trị cố định trong log
const (
	PODPROCESSINGLOG        = "Đang xử lý pod: %s\n"
	MEMORYLIMITLOG          = "Pod %s không vượt ngưỡng %s. Bỏ qua.\n"
	PODCREATEDLOG           = "Pod mới %s đã được tạo.\n"
	PODREADYLOG             = "Pod %s đã sẵn sàng.\n"
	PODNOTREADYLOG          = "Pod %s chưa sẵn sàng. Đợi...\n"
	PODDELETEDLOG           = "Pod cũ %s đã được xóa.\n"
	CANNOTCREATEPODLOG      = "Không thể tạo pod thay thế cho %s: %v\n"
	CANNOTDELETEPODLOG      = "Không thể xóa pod %s: %v\n"
	ERR_K8S_CONFIG_LOAD     = "Không thể tải cấu hình Kubernetes: %v"
	ERR_K8S_CLIENT_CREATION = "Không thể tạo Kubernetes client: %v"
	ERR_METRICS_CLIENT      = "Error creating metrics client: %v"
	ERR_GET_PODS            = "Không thể lấy danh sách pods: %v"
	LIMIT                   = "limit"
	NAMESPACE               = "nginx"
	POLL_INTERVAL           = 10
	SHUTDOWN                = "shutdown"
)

// Khai báo instance của FIELD_NAME với các giá trị cố định
var FieldNames = FIELD_NAME{
	NAMESPACE:     "NAMESPACE",
	CHECK_BY:      "CHECK_BY",
	POLL_INTERVAL: "POLL_INTERVAL",
}
