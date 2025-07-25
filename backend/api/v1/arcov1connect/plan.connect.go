// Code generated by protoc-gen-connect-go. DO NOT EDIT.
//
// Source: api/v1/plan.proto

package arcov1connect

import (
	connect "connectrpc.com/connect"
	context "context"
	errors "errors"
	v1 "github.com/loomi-labs/arco/backend/api/v1"
	http "net/http"
	strings "strings"
)

// This is a compile-time assertion to ensure that this generated file and the connect package are
// compatible. If you get a compiler error that this constant is not defined, this code was
// generated with a version of connect newer than the one compiled into your binary. You can fix the
// problem by either regenerating this code with an older version of connect or updating the connect
// version compiled into your binary.
const _ = connect.IsAtLeastVersion1_13_0

const (
	// PlanServiceName is the fully-qualified name of the PlanService service.
	PlanServiceName = "api.v1.PlanService"
)

// These constants are the fully-qualified names of the RPCs defined in this package. They're
// exposed at runtime as Spec.Procedure and as the final two segments of the HTTP route.
//
// Note that these are different from the fully-qualified method names used by
// google.golang.org/protobuf/reflect/protoreflect. To convert from these constants to
// reflection-formatted method names, remove the leading slash and convert the remaining slash to a
// period.
const (
	// PlanServiceListPlansProcedure is the fully-qualified name of the PlanService's ListPlans RPC.
	PlanServiceListPlansProcedure = "/api.v1.PlanService/ListPlans"
)

// PlanServiceClient is a client for the api.v1.PlanService service.
type PlanServiceClient interface {
	// ListPlans returns all available subscription plans with USD pricing.
	//
	// This endpoint is publicly accessible and returns both Basic and Pro plans
	// with their respective storage limits, feature sets, and USD pricing.
	//
	// Pro plans include overage pricing for storage beyond the base limit,
	// charged in 10GB increments.
	ListPlans(context.Context, *connect.Request[v1.ListPlansRequest]) (*connect.Response[v1.ListPlansResponse], error)
}

// NewPlanServiceClient constructs a client for the api.v1.PlanService service. By default, it uses
// the Connect protocol with the binary Protobuf Codec, asks for gzipped responses, and sends
// uncompressed requests. To use the gRPC or gRPC-Web protocols, supply the connect.WithGRPC() or
// connect.WithGRPCWeb() options.
//
// The URL supplied here should be the base URL for the Connect or gRPC server (for example,
// http://api.acme.com or https://acme.com/grpc).
func NewPlanServiceClient(httpClient connect.HTTPClient, baseURL string, opts ...connect.ClientOption) PlanServiceClient {
	baseURL = strings.TrimRight(baseURL, "/")
	planServiceMethods := v1.File_api_v1_plan_proto.Services().ByName("PlanService").Methods()
	return &planServiceClient{
		listPlans: connect.NewClient[v1.ListPlansRequest, v1.ListPlansResponse](
			httpClient,
			baseURL+PlanServiceListPlansProcedure,
			connect.WithSchema(planServiceMethods.ByName("ListPlans")),
			connect.WithClientOptions(opts...),
		),
	}
}

// planServiceClient implements PlanServiceClient.
type planServiceClient struct {
	listPlans *connect.Client[v1.ListPlansRequest, v1.ListPlansResponse]
}

// ListPlans calls api.v1.PlanService.ListPlans.
func (c *planServiceClient) ListPlans(ctx context.Context, req *connect.Request[v1.ListPlansRequest]) (*connect.Response[v1.ListPlansResponse], error) {
	return c.listPlans.CallUnary(ctx, req)
}

// PlanServiceHandler is an implementation of the api.v1.PlanService service.
type PlanServiceHandler interface {
	// ListPlans returns all available subscription plans with USD pricing.
	//
	// This endpoint is publicly accessible and returns both Basic and Pro plans
	// with their respective storage limits, feature sets, and USD pricing.
	//
	// Pro plans include overage pricing for storage beyond the base limit,
	// charged in 10GB increments.
	ListPlans(context.Context, *connect.Request[v1.ListPlansRequest]) (*connect.Response[v1.ListPlansResponse], error)
}

// NewPlanServiceHandler builds an HTTP handler from the service implementation. It returns the path
// on which to mount the handler and the handler itself.
//
// By default, handlers support the Connect, gRPC, and gRPC-Web protocols with the binary Protobuf
// and JSON codecs. They also support gzip compression.
func NewPlanServiceHandler(svc PlanServiceHandler, opts ...connect.HandlerOption) (string, http.Handler) {
	planServiceMethods := v1.File_api_v1_plan_proto.Services().ByName("PlanService").Methods()
	planServiceListPlansHandler := connect.NewUnaryHandler(
		PlanServiceListPlansProcedure,
		svc.ListPlans,
		connect.WithSchema(planServiceMethods.ByName("ListPlans")),
		connect.WithHandlerOptions(opts...),
	)
	return "/api.v1.PlanService/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case PlanServiceListPlansProcedure:
			planServiceListPlansHandler.ServeHTTP(w, r)
		default:
			http.NotFound(w, r)
		}
	})
}

// UnimplementedPlanServiceHandler returns CodeUnimplemented from all methods.
type UnimplementedPlanServiceHandler struct{}

func (UnimplementedPlanServiceHandler) ListPlans(context.Context, *connect.Request[v1.ListPlansRequest]) (*connect.Response[v1.ListPlansResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("api.v1.PlanService.ListPlans is not implemented"))
}
