package internal

import (
	"github.com/golang/protobuf/ptypes"

	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	hcm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
)

const (
	CurrentClusterName = "xds_cluster"
	RouteName          = "local_route"
	ListenerName       = "listener_0"
	ListenerPort       = 9901
)

func makeRoute(routes []*route.Route) *route.RouteConfiguration {
	return &route.RouteConfiguration{
		Name: RouteName,
		VirtualHosts: []*route.VirtualHost{{
			Name:    "local_service",
			Domains: []string{"*"},
			Routes:  routes,
		}},
	}
}

func makeHTTPListener() *listener.Listener {
	// HTTP filter configuration
	manager := &hcm.HttpConnectionManager{
		CodecType:  hcm.HttpConnectionManager_AUTO,
		StatPrefix: "http",
		RouteSpecifier: &hcm.HttpConnectionManager_Rds{
			Rds: &hcm.Rds{
				ConfigSource:    makeConfigSource(),
				RouteConfigName: RouteName,
			},
		},
		HttpFilters: []*hcm.HttpFilter{{
			Name: wellknown.Router,
		}},
	}
	pbst, err := ptypes.MarshalAny(manager)
	if err != nil {
		panic(err)
	}

	return &listener.Listener{
		Name: ListenerName,
		Address: &core.Address{
			Address: &core.Address_SocketAddress{
				SocketAddress: &core.SocketAddress{
					Protocol: core.SocketAddress_TCP,
					Address:  "0.0.0.0",
					PortSpecifier: &core.SocketAddress_PortValue{
						PortValue: ListenerPort,
					},
				},
			},
		},
		FilterChains: []*listener.FilterChain{{
			Filters: []*listener.Filter{{
				Name: wellknown.HTTPConnectionManager,
				ConfigType: &listener.Filter_TypedConfig{
					TypedConfig: pbst,
				},
			}},
		}},
	}
}

func makeConfigSource() *core.ConfigSource {
	source := &core.ConfigSource{}
	source.ResourceApiVersion = resource.DefaultAPIVersion
	source.ConfigSourceSpecifier = &core.ConfigSource_ApiConfigSource{
		ApiConfigSource: &core.ApiConfigSource{
			TransportApiVersion:       resource.DefaultAPIVersion,
			ApiType:                   core.ApiConfigSource_GRPC,
			SetNodeOnFirstMessageOnly: true,
			GrpcServices: []*core.GrpcService{{
				TargetSpecifier: &core.GrpcService_EnvoyGrpc_{
					EnvoyGrpc: &core.GrpcService_EnvoyGrpc{ClusterName: CurrentClusterName},
				},
			}},
		},
	}
	return source
}

func GenerateSnapshot(userServers []*UserServer) cache.Snapshot {
	routes := make([]*route.Route, len(userServers))
	clusters := make([]types.Resource, len(userServers))
	for i, c := range userServers {
		routes[i] = c.makeRoute()
		clusters[i] = c.makeCluster()
	}
	return cache.NewSnapshot(
		"1",
		[]types.Resource{},
		clusters,
		[]types.Resource{makeRoute(routes)},
		[]types.Resource{makeHTTPListener()},
		[]types.Resource{},
		[]types.Resource{},
	)
	// snap, _ := cache.NewSnapshot("1",
	// 	map[resource.Type][]types.Resource{
	// 		resource.ClusterType:  clusters,
	// 		resource.RouteType:    {makeRoute(routes)},
	// 		resource.ListenerType: {makeHTTPListener()},
	// 	})
	// return snap
}
