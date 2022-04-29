package source

import (
	"context"
	"strings"
	"strconv"
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"sigs.k8s.io/external-dns/endpoint"
)

type dockerContainerLabelSource struct {
}

func NewDockerContainerLabelSource() (Source, error) {
	return &dockerContainerLabelSource{}, nil
}

func (sc *dockerContainerLabelSource) Endpoints(ctx context.Context) ([]*endpoint.Endpoint, error) {
	var endpoints []*endpoint.Endpoint

	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, err
	}

	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		return nil, err
	}

	for _, container := range containers {
		hostnames := container.Labels[hostnameAnnotationKey]

		for _, hostname := range strings.Split(hostnames, ",") {
			
			target := container.Labels[targetAnnotationKey]
			if (target == "") {
				continue
			}

			ttl :=  endpoint.TTL(0)
			ttlString := container.Labels[ttlAnnotationKey]
			if (ttlString != "") {
				if number, err := strconv.Atoi(ttlString); err != nil {
					ttl = endpoint.TTL(number)
				}
			}

			endpoint := endpoint.NewEndpointWithTTL(hostname, endpoint.RecordTypeCNAME, ttl, target)

			endpoints = append(endpoints, endpoint)
		}
	}

	return endpoints, nil
}

func (sc *dockerContainerLabelSource) AddEventHandler(ctx context.Context, handler func()) {
}