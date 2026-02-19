package contracts

type ApiResponse struct {
	Data  any       `json:"data,omitempty"`
	Error *ApiError `json:"error,omitempty"`
}

type ApiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
