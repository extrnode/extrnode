package asn

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"

	"github.com/biter777/countries"

	"extrnode-be/internal/pkg/log"
	"extrnode-be/internal/scanner/models"
)

const whoisServiceAddress = "whois.cymru.com:43"

var ErrEmptyCountryName = errors.New("empty country name")

func formMsg(addrs []models.NodeInfo) []byte {
	var msg = "begin\nverbose\n"
	for _, addr := range addrs {
		msg = msg + addr.IP.String() + "\n"
	}
	msg = msg + "end\n"

	return []byte(msg)
}

func GetWhoisRecords(addrs []models.NodeInfo) (res []models.NodeInfo, err error) {
	conn, err := net.Dial("tcp", whoisServiceAddress)
	if err != nil {
		return res, fmt.Errorf("Dial: %s", err)
	}
	defer conn.Close()

	_, err = conn.Write(formMsg(addrs))
	if err != nil {
		return res, fmt.Errorf("Write: %s", err)
	}

	read := bufio.NewReader(conn)
	_, err = read.ReadString('\n')
	if err != nil {
		return res, fmt.Errorf("ReadString 1: %s", err)
	}

	for _, a := range addrs {
		output, err := read.ReadString('\n')
		if err != nil && err != io.EOF {
			return res, fmt.Errorf("ReadString 2: %s", err)
		}

		asnInfo, err := parse(output)
		if err != nil {
			if !errors.Is(err, ErrEmptyCountryName) {
				log.Logger.Scanner.Warn("asnScaner.GetWhoisRecords: Parse: ", err)
			}

			continue
		}

		a.AsnInfo = asnInfo
		res = append(res, a)
	}

	return res, nil
}

func parse(input string) (output models.AsnInfo, err error) {
	s := strings.Split(input, "|")
	if len(s) != 7 {
		return output, fmt.Errorf("wrong input %s, can`t parse", input)
	}

	countryName := strings.TrimSpace(s[3])
	if countryName == "" {
		return output, ErrEmptyCountryName
	}
	cc := countries.ByName(countryName)
	if !cc.IsValid() {
		return output, fmt.Errorf("fail to get country by name: %s", countryName)
	}
	output.Alpha2, output.Alpha3, output.Name = cc.Alpha2(), cc.Alpha3(), cc.String()
	output.Isp = strings.TrimSpace(s[6])

	_, output.Network, err = net.ParseCIDR(strings.TrimSpace(s[2]))
	if err != nil {
		return output, err
	}

	output.As, err = strconv.ParseUint(strings.TrimSpace(s[0]), 10, 64)
	if err != nil {
		return output, err
	}

	return output, nil
}
