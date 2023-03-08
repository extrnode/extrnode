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

func (p *proxy) getScannedMethods() (scannedMethodList map[string]int, err error) {
	blockchainID, ok := p.blockchainIDs[solanaBlockchain]
	if !ok {
		return scannedMethodList, fmt.Errorf("fail to get blockchainID")
	}

	scannedMethodList, err = p.slStorage.GetRpcMethodsMapByBlockchainID(blockchainID)
	if err != nil {
		return scannedMethodList, fmt.Errorf("GetRpcMethodsMapByBlockchainID: %s", err)
	}

	return scannedMethodList, nil
}

func (p *proxy) getEndpointsURLs(blockchain string) ([]middlewares.UrlWithMethods, error) {
	blockchainID, ok := p.blockchainIDs[blockchain]
	if !ok {
		return nil, fmt.Errorf("fail to get blockchainID")
	}

	endpoints, err := p.slStorage.GetEndpoints(blockchainID, 0, nil, nil, nil, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("GetEndpoints: %s", err)
	}

	// temp sort solution
	// TODO: sort by methods
	sort.Slice(endpoints, func(i, j int) bool {
		return endpoints[i].SupportedMethods.AverageResponseTime() < endpoints[j].SupportedMethods.AverageResponseTime()
	})

	urlsWithMethods := make([]middlewares.UrlWithMethods, 0, len(endpoints))
	for _, e := range endpoints {
		schema := "http://"
		if e.IsSsl {
			schema = "https://"
		}
		parsedUrl, err := url.Parse(fmt.Sprintf("%s%s", schema, e.Endpoint))
		if err != nil {
			return nil, fmt.Errorf("url.Parse: %s", err)
		}

		supportedMethods := make(map[string]struct{}, len(e.SupportedMethods))
		for _, method := range e.SupportedMethods {
			supportedMethods[method.Name] = struct{}{}
		}

		urlsWithMethods = append(urlsWithMethods, middlewares.UrlWithMethods{
			Url:              parsedUrl,
			SupportedMethods: supportedMethods,
		})
	}

	return urlsWithMethods, nil
}

func (p *proxy) updateProxyEndpoints(transport *middlewares.ProxyTransport) {
	for {
		urlsWithMethods, err := p.getEndpointsURLs(solanaBlockchain)
		if err != nil {
			log.Logger.Proxy.Logger.Errorf("Cannot get endpoints from db: %s", err.Error()) // the algorithm will go ahead and clear all targers, therefore continue not needed
		}

		transport.UpdateTargets(urlsWithMethods)
		metrics.ObserveAvailableEndpoints(len(urlsWithMethods))

		time.Sleep(endpointsReloadInterval)
	}
}
