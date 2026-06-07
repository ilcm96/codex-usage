package httpapi

type ErrorResponse struct {
	Error string `json:"error"`
}

type StatusResponse struct {
	Status string `json:"status"`
}

type HealthResponse struct {
	Status string `json:"status"`
}

type MeResponse struct {
	Role string `json:"role"`
}
