package proxy

import (
	"fmt"
	"net/url"
	"sort"
	"time"

	"extrnode-be/internal/pkg/log"
	"extrnode-be/internal/pkg/metrics"
	"extrnode-be/internal/proxy/middlewares"
)

func (p *proxy) getEndpointsURLs(blockchain string) ([]*url.URL, error) {
	blockchainID, ok := p.blockchainIDs[blockchain]
	if !ok {
		return nil, fmt.Errorf("fail to get blockchainID")
	}
	isRpc := true

	endpoints, err := p.pgStorage.GetEndpoints(blockchainID, 0, &isRpc, nil, nil, nil, nil)
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

func (p *proxy) updateProxyEndpoints(transport *middlewares.ProxyTransport) {
	for {
		urls, err := p.getEndpointsURLs(solanaBlockchain)
		if err != nil {
			log.Logger.Proxy.Logger.Errorf("Cannot get endpoints from db: %s", err.Error()) // the algorithm will go ahead and clear all targers, therefore continue not needed
		}
		transport.UpdateTargets(urls)
		metrics.ObserveAvailableEndpoints(len(urls))

		time.Sleep(endpointsReloadInterval)
	}
}
