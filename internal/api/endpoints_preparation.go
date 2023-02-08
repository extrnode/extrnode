package api

import (
	"fmt"
	"net/url"
	"sort"
	"time"

	"extrnode-be/internal/api/middlewares/proxy"
	"extrnode-be/internal/pkg/log"
	"extrnode-be/internal/pkg/metrics"
)

func (a *api) getEndpointsURLs(blockchain string) ([]*url.URL, error) {
	blockchainID, ok := a.blockchainIDs[blockchain]
	if !ok {
		return nil, fmt.Errorf("fail to get blockchainID")
	}
	isRpc := true

	endpoints, err := a.pgStorage.GetEndpoints(blockchainID, 0, &isRpc, nil, nil, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("GetEndpoints: %s", err)
	}

	// temp sort solution
	// TODO: sort by methods
	sort.Slice(endpoints, func(i, j int) bool {
		return endpoints[i].SupportedMethods.AverageResponseTime() < endpoints[j].SupportedMethods.AverageResponseTime()
	})

	urls := make([]*url.URL, 0, len(endpoints))
	for _, e := range endpoints {
		schema := "http://"
		if e.IsSsl {
			schema = "https://"
		}
		parsedUrl, err := url.Parse(fmt.Sprintf("%s%s", schema, e.Endpoint))
		if err != nil {
			return nil, fmt.Errorf("url.Parse: %s", err)
		}

		urls = append(urls, parsedUrl)
	}

	return urls, nil
}

func (a *api) updateProxyEndpoints(transport *proxy.ProxyTransport) {
	for {
		urls, err := a.getEndpointsURLs(solanaBlockchain)
		if err != nil {
			log.Logger.Api.Logger.Fatalf("Cannot get endpoints from db: %s", err.Error())
		}

		transport.UpdateTargets(urls)
		metrics.ObserveAvailableEndpoints(len(urls))

		time.Sleep(endpointsReloadInterval)
	}
}
