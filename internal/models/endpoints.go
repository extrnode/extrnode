package models

type (
	Endpoint struct {
		Endpoint         string           `json:"endpoint"`
		Version          string           `json:"version" `
		SupportedMethods SupportedMethods `json:"supported_methods"`
		IsRpc            bool             `json:"is_rpc"`
		IsValidator      bool             `json:"is_validator"`
		IsSsl            bool             `json:"is_ssl"`
		AsnInfo          AsnInfo          `json:"asn_info"`
	}
	EndpointCsv struct {
		Endpoint    string `csv:"endpoint"`
		Version     string `csv:"version"`
		As          int    `csv:"as"`
		Network     string `csv:"network"`
		Country     string `csv:"country"`
		Isp         string `csv:"isp"`
		IsRpc       bool   `csv:"is_rpc"`
		IsValidator bool   `csv:"is_validator"`
	}
	AsnInfo struct {
		Network string  `json:"network"`
		Isp     string  `json:"isp"`
		As      int     `json:"ntw_as"`
		Country Country `json:"country"`
	}
	Country struct {
		Alpha2 string `json:"alpha2"`
		Alpha3 string `json:"alpha3"`
		Name   string `json:"name"`
	}
	Stat struct {
		Total     int `json:"total"`
		Alive     int `json:"alive"`
		Rpc       int `json:"rpc"`
		Validator int `json:"validator"`
	}
)

// helpers
type (
	SupportedMethods []struct {
		Name         string `json:"name"`
		ResponseTime int64  `json:"response_time"`
	}
)

func (s SupportedMethods) AverageResponseTime() (total int64) {
	if len(s) == 0 {
		return total
	}

	for _, v := range s {
		total += v.ResponseTime
	}

	return total / int64(len(s))
}
