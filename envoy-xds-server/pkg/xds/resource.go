package xds

import (
	"fmt"
	"time"

	"github.com/tellery/saas-xds-server/pkg/constant"
	"github.com/tellery/saas-xds-server/pkg/entity"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"

	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	hcm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
)

const (
	currentClusterName = "xds_cluster"
	routeName          = "local_route"
	listenerName       = "listener_0"
	listenerPort       = 9901
)

func toClusterName(c *entity.UserServer) string {
	return fmt.Sprintf("user-server-%s", c.UserId)
}

func makeCluster(c *entity.UserServer) *cluster.Cluster {
	return &cluster.Cluster{
		Name:           toClusterName(c),
		ConnectTimeout: durationpb.New(250 * time.Millisecond),
		// TODO: polish here
		ClusterDiscoveryType: &cluster.Cluster_Type{Type: cluster.Cluster_STRICT_DNS},
		LbPolicy:             cluster.Cluster_ROUND_ROBIN,
		LoadAssignment:       makeEndpoint(c),
		DnsLookupFamily:      cluster.Cluster_V4_ONLY,
		Http2ProtocolOptions: &core.Http2ProtocolOptions{},
	}
}

func makeEndpoint(c *entity.UserServer) *endpoint.ClusterLoadAssignment {
	return &endpoint.ClusterLoadAssignment{
		ClusterName: toClusterName(c),
		Endpoints: []*endpoint.LocalityLbEndpoints{{
			LbEndpoints: []*endpoint.LbEndpoint{{
				HostIdentifier: &endpoint.LbEndpoint_Endpoint{
					Endpoint: &endpoint.Endpoint{
						Address: &core.Address{
							Address: &core.Address_SocketAddress{
								SocketAddress: &core.SocketAddress{
									Protocol: core.SocketAddress_TCP,
									Address:  c.ServiceName,
									PortSpecifier: &core.SocketAddress_PortValue{
										PortValue: c.ServicePort,
									},
								},
							},
						},
					},
				},
			}},
		}},
	}
}

func makeRoute(c *entity.UserServer) *route.Route {
	return &route.Route{
		Match: &route.RouteMatch{
			PathSpecifier: &route.RouteMatch_Prefix{
				Prefix: "/",
			},
			Headers: []*route.HeaderMatcher{{
				Name: constant.ServiceHeaderMatcher,
				HeaderMatchSpecifier: &route.HeaderMatcher_ExactMatch{
					ExactMatch: c.UserId,
				},
			}},
		},
		Action: &route.Route_Route{
			Route: &route.RouteAction{
				ClusterSpecifier: &route.RouteAction_Cluster{
					Cluster: toClusterName(c),
				},
			},
		},
	}
}

func makeRouteCfg(routes []*route.Route) *route.RouteConfiguration {
	return &route.RouteConfiguration{
		Name: routeName,
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
				RouteConfigName: routeName,
			},
		},
		HttpFilters: []*hcm.HttpFilter{{
			Name: wellknown.Router,
		}},
	}
	pbst, err := anypb.New(manager)
	if err != nil {
		panic(err)
	}

	return &listener.Listener{
		Name: listenerName,
		Address: &core.Address{
			Address: &core.Address_SocketAddress{
				SocketAddress: &core.SocketAddress{
					Protocol: core.SocketAddress_TCP,
					Address:  "0.0.0.0",
					PortSpecifier: &core.SocketAddress_PortValue{
						PortValue: listenerPort,
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
			ApiType:                   core.ApiConfigSource_DELTA_GRPC,
			SetNodeOnFirstMessageOnly: true,
			GrpcServices: []*core.GrpcService{{
				TargetSpecifier: &core.GrpcService_EnvoyGrpc_{
					EnvoyGrpc: &core.GrpcService_EnvoyGrpc{ClusterName: currentClusterName},
				},
			}},
		},
	}
	return source
}

func GenerateSnapshot(userServers []*entity.UserServer, currentVersion string) cache.Snapshot {
	routes := make([]*route.Route, len(userServers))
	clusters := make([]types.Resource, len(userServers))
	for i, c := range userServers {
		routes[i] = makeRoute(c)
		clusters[i] = makeCluster(c)
	}
	return cache.NewSnapshot(
		currentVersion,
		[]types.Resource{},
		clusters,
		[]types.Resource{makeRouteCfg(routes)},
		[]types.Resource{makeHTTPListener()},
		[]types.Resource{},
		[]types.Resource{},
	)
}
