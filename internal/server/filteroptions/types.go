package filteroptions

type DateRange struct {
	Oldest string `json:"oldest"`
	Newest string `json:"newest"`
}

type Option struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Detail string `json:"detail"`
	Count  int64  `json:"count"`
}

type Result struct {
	DateRange    DateRange `json:"dateRange"`
	Devices      []Option  `json:"devices"`
	Repositories []Option  `json:"repositories"`
	Projects     []Option  `json:"projects"`
	Models       []Option  `json:"models"`
	Branches     []Option  `json:"branches"`
}
