package models

type (
	Endpoint struct {
		Endpoint         string   `json:"endpoint" pg:"endpoint"`
		Version          string   `json:"version"  pg:"version"`
		SupportedMethods []string `json:"supported_methods" pg:"supported_methods"`
		//UnscannedMethods []string `json:"unscanned_methods" pg:"unscanned_methods"`
		IsRpc    bool    `json:"is_rpc" pg:"is_rpc"`
		AsnInfo  AsnInfo `json:"asn_info" pg:"asn_info"`
		ScanTime int64   `json:"scan_time" pg:"scan_time"`
	}
	EndpointCsv struct {
		Endpoint string `csv:"endpoint"`
		Version  string `csv:"version"`
		As       int    `csv:"as"`
		Network  string `csv:"network"`
		Country  string `csv:"country"`
		Isp      string `csv:"isp"`
		IsRpc    bool   `csv:"is_rpc"`
	}
	AsnInfo struct {
		Network string  `json:"network" pg:"network"`
		Isp     string  `json:"isp" pg:"isp"`
		As      int     `json:"ntw_as" pg:"ntw_as"`
		Country Country `json:"country" pg:"country"`
	}
	Country struct {
		Alpha2 string `json:"alpha2" pg:"alpha2"`
		Alpha3 string `json:"alpha3" pg:"alpha3"`
		Name   string `json:"name" pg:"name"`
	}
)
