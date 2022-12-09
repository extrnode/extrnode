package models

type (
	Endpoint struct {
		Endpoint         string   `json:"endpoint" pg:"endpoint"`
		Version          string   `json:"version"  pg:"version"`
		SupportedMethods []string `json:"supported_methods" pg:"supported_methods"`
		UnscannedMethods []string `json:"unscanned_methods" pg:"unscanned_methods"`
		NodeType         bool     `json:"node_type" pg:"node_type"`
		AsnInfo          AsnInfo  `json:"asn_info" pg:"asn_info"`
		ScanTime         int64    `json:"scan_time" pg:"scan_time"`
	}
	EndpointCsv struct {
		Endpoint string `csv:"endpoint"`
		Version  string `csv:"version"`
		Network  string `csv:"network"`
		Country  string `csv:"country"`
		Isp      string `csv:"isp"`
		NodeType bool   `csv:"node_type"`
	}
	AsnInfo struct {
		Network string  `json:"network" pg:"network"`
		Country Country `json:"country" pg:"country"`
		Isp     string  `json:"isp"  pg:"isp"`
	}
	Country struct {
		Alpha2 string `json:"alpha2" pg:"alpha2"`
		Alpha3 string `json:"alpha3" pg:"alpha3"`
		Name   string `json:"name" pg:"name"`
	}
)
