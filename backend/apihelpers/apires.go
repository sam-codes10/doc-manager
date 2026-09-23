package apihelpers

// ApiRes is a generic API response structure
type ApiRes struct {
	Data    interface{} `json:"data"`
	Status  bool        `json:"status"`
	Message string      `json:"message"`
}

// NewApiRes creates a new ApiRes instance
func NewApiRes(data interface{}, status bool, message string) ApiRes {
	return ApiRes{
		Data:    data,
		Status:  status,
		Message: message,
	}
}
