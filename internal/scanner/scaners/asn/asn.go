package asn

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/biter777/countries"

	"extrnode-be/internal/pkg/log"
	"extrnode-be/internal/scanner/models"
)

const whoisServiceAddress = "whois.cymru.com:43"

var ErrEmptyCountryName = errors.New("empty country name")

func formMsg(addrs []models.NodeInfo, isGetCountryByAS bool) []byte {
	var msg string
	for _, addr := range addrs {
		if isGetCountryByAS {
			msg = msg + "AS" + strconv.FormatUint(addr.As, 10) + "\n"
		} else {
			msg = msg + addr.IP.String() + "\n"
		}
	}

	return []byte(msg)
}

func formResponseArray(addrs []models.NodeInfo, read *bufio.Reader, isAddCountryInfo bool) (res []models.NodeInfo, additionalASN []models.NodeInfo, err error) {
	for _, a := range addrs {
		output, err := read.ReadString('\n')
		if err != nil {
			return res, additionalASN, fmt.Errorf("ReadString: %s", err)
		}
		if output == "" {
			continue
		}

		if isAddCountryInfo {
			country, err := parseCountry(output)
			if err != nil {
				log.Logger.Scanner.Warn("asnScaner.GetWhoisRecords: Parse: ", err)
				continue
			}

			a.Alpha2, a.Alpha3, a.Name = country.Alpha2(), country.Alpha3(), country.String()
		} else {
			a.AsnInfo, err = parse(output)
			if err != nil {
				if errors.Is(err, ErrEmptyCountryName) {
					additionalASN = append(additionalASN, a)
				} else {
					log.Logger.Scanner.Warn("asnScaner.GetWhoisRecords: Parse: ", err)
				}
				continue
			}
		}
		res = append(res, a)
	}

	return res, additionalASN, nil
}

func GetWhoisRecords(addrs []models.NodeInfo) (res []models.NodeInfo, err error) {
	if len(addrs) == 0 {
		return res, nil
	}

	conn, err := net.Dial("tcp", whoisServiceAddress)
	if err != nil {
		return res, fmt.Errorf("Dial: %s", err)
	}
	defer conn.Close()

	_, err = conn.Write([]byte("begin\nverbose\n"))
	if err != nil {
		return res, fmt.Errorf("Write 1: %s", err)
	}

	_, err = conn.Write(formMsg(addrs, false))
	if err != nil {
		return res, fmt.Errorf("Write 2: %s", err)
	}
	defer conn.Write([]byte("end\n"))

	var additionalASN, resAdd []models.NodeInfo

	read := bufio.NewReader(conn)
	_, err = read.ReadString('\n')
	if err != nil {
		return res, fmt.Errorf("ReadString: %s", err)
	}

	resAdd, additionalASN, err = formResponseArray(addrs, read, false)
	if err != nil {
		return res, fmt.Errorf("formResponseArray 1: %s", err)
	}

	res = append(res, resAdd...)
	if len(additionalASN) > 0 {
		_, err = conn.Write(formMsg(additionalASN, true))
		if err != nil {
			return res, fmt.Errorf("Write 3: %s", err)
		}

		resAdd, _, err = formResponseArray(additionalASN, read, true)
		if err != nil {
			return res, fmt.Errorf("formResponseArray 2: %s", err)
		}

		res = append(res, resAdd...)
	}

	return res, nil
}

func parse(input string) (output models.AsnInfo, err error) {
	s := strings.Split(input, "|")
	if len(s) != 7 {
		return output, fmt.Errorf("wrong input %s, can`t parse", input)
	}

	output.As, err = strconv.ParseUint(strings.TrimSpace(s[0]), 10, 64)
	if err != nil {
		return output, err
	}

	output.Isp = strings.TrimSpace(s[6])
	_, output.Network, err = net.ParseCIDR(strings.TrimSpace(s[2]))
	if err != nil {
		return output, err
	}

	countryName := strings.TrimSpace(s[3])
	if countryName == "" {
		return output, ErrEmptyCountryName
	}

	cc := countries.ByName(countryName)
	if !cc.IsValid() {
		return output, fmt.Errorf("fail to get country by name: %s", s)
	}

	output.Alpha2, output.Alpha3, output.Name = cc.Alpha2(), cc.Alpha3(), cc.String()

	return output, nil
}

func parseCountry(input string) (country countries.CountryCode, err error) {
	s := strings.Split(input, "|")
	if len(s) != 5 {
		return country, fmt.Errorf("wrong input %s, can`t parse", input)
	}

	country = countries.ByName(strings.TrimSpace(s[1]))
	if !country.IsValid() {
		return country, fmt.Errorf("fail to get country by name: %s", s)
	}

	return country, nil
}
