package nmap

import (
	"context"
	"fmt"
	"time"

	"github.com/Ullaakut/nmap/v2"

	"extrnode-be/internal/pkg/log"
	"extrnode-be/internal/pkg/storage"
)

const portOpenState = "open"

func ScanAndInsertPorts(ctx context.Context, s storage.PgStorage, peer storage.PeerWithIpAndBlockchain) error {
	ports, err := NmapScan(ctx, peer.Address.String())
	if err != nil {
		return err
	}

	for _, port := range ports {
		if peer.Port == int(port) {
			continue
		}
		// TODO: handle diff blockchains
		_, err = s.GetOrCreatePeer(peer.BlockchainID, peer.IpID, int(port), "", false, false, false, false, false, "")
		if err != nil {
			return fmt.Errorf("GetOrCreatePeer: %s; blcId %d ipId %d port %d", err, peer.BlockchainID, peer.IpID, port)
		}

	}

	return nil
}

func NmapScan(ctx context.Context, address string) (ports []uint16, err error) {
	scanner, err := nmap.NewScanner(
		nmap.WithTargets(address),
		nmap.WithSkipHostDiscovery(),
		nmap.WithDisabledDNSResolution(),
		nmap.WithHostTimeout(time.Second),
		nmap.WithMaxRTTTimeout(200*time.Millisecond),
		nmap.WithMaxScanDelay(0),
		nmap.WithPorts("80,82,88,90,443,1234,1922,3000,7778,7999,8008,8080,8090,8099,8443,8545,8666,8799,8819,8888,8899,9000,9024,9857,14000,21611,30000,38899"),
		nmap.WithContext(ctx),
	)
	if err != nil {
		return ports, err
	}

	result, warnings, err := scanner.Run()

	if err != nil || len(result.Hosts) == 0 || len(result.Hosts[0].Ports) == 0 || len(result.Hosts[0].Addresses) == 0 {
		return ports, err
	}

	if warnings != nil {
		log.Logger.Scanner.Warn(err)
	}

	for _, port := range result.Hosts[0].Ports {
		if port.State.String() == portOpenState {
			ports = append(ports, port.ID)
		}
	}

	return ports, err
}
