package internal

import (
	"fmt"
	"time"

	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	"github.com/golang/protobuf/ptypes"
)

type UserServer struct {
	UserId      string
	Deployment  string
	ServiceName string
	ServicePort uint32
}

func (c *UserServer) getName() string {
	return fmt.Sprintf("server%s", c.UserId)
}

func (c *UserServer) makeCluster() *cluster.Cluster {
	return &cluster.Cluster{
		Name:           c.getName(),
		ConnectTimeout: ptypes.DurationProto(250 * time.Millisecond),
		// TODO: polish here
		ClusterDiscoveryType: &cluster.Cluster_Type{Type: cluster.Cluster_STRICT_DNS},
		LbPolicy:             cluster.Cluster_ROUND_ROBIN,
		LoadAssignment:       c.makeEndpoint(),
		// any difference ?
		DnsLookupFamily:      cluster.Cluster_V4_ONLY,
		Http2ProtocolOptions: &core.Http2ProtocolOptions{},
	}
}

func (c *UserServer) makeEndpoint() *endpoint.ClusterLoadAssignment {
	return &endpoint.ClusterLoadAssignment{
		ClusterName: c.getName(),
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

func (c *UserServer) makeRoute() *route.Route {
	return &route.Route{
		Match: &route.RouteMatch{
			PathSpecifier: &route.RouteMatch_Prefix{
				Prefix: "/",
			},
			Headers: []*route.HeaderMatcher{{
				Name: "User-Id",
				HeaderMatchSpecifier: &route.HeaderMatcher_ExactMatch{
					ExactMatch: c.UserId,
				},
			}},
		},
		Action: &route.Route_Route{
			Route: &route.RouteAction{
				ClusterSpecifier: &route.RouteAction_Cluster{
					Cluster: c.getName(),
				},
			},
		},
	}
}
