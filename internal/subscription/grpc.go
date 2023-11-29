package subscription

import (
	"context"
	"github.com/hashicorp/vault/sdk/helper/pointerutil"
	pb "github.com/k8sdeploy/protobufs/generated/subscription_service/v1"
	"github.com/k8sdeploy/subscription-service/internal/config"
)

type Server struct {
	pb.UnimplementedSubscriptionServiceServer
	config.Config
}

func (s *Server) GetAgentLimit(ctx context.Context, req *pb.GetAgentLimitRequest) (*pb.GetAgentLimitResponse, error) {
	a := NewSubscriptionService(ctx, s.Config, &RealMongoOperations{
		Collection: s.Mongo.Collections["subs"],
		Database:   s.Mongo.Database,
	})
	if err := a.MongoOps.GetMongoClient(ctx, a.Mongo); err != nil {
		return &pb.GetAgentLimitResponse{
			Status: pointerutil.StringPtr(err.Error()),
		}, nil
	}

	ad, err := a.GetAgentLimit(req.GetCompanyId())
	if err != nil {
		return &pb.GetAgentLimitResponse{
			Status: pointerutil.StringPtr(err.Error()),
		}, nil
	}

	return &pb.GetAgentLimitResponse{
		Limit:         int32(ad.Agents.Limit),
		Used:          int32(ad.Agents.Used),
		Grandfathered: &ad.Grandfathered,
	}, nil
}

func (s *Server) UpdateAgentLimit(ctx context.Context, req *pb.UpdateAgentLimitRequest) (*pb.GetAgentLimitResponse, error) {
	a := NewSubscriptionService(ctx, s.Config, &RealMongoOperations{
		Collection: s.Mongo.Collections["subs"],
		Database:   s.Mongo.Database,
	})
	if err := a.MongoOps.GetMongoClient(ctx, a.Mongo); err != nil {
		return &pb.GetAgentLimitResponse{
			Status: pointerutil.StringPtr(err.Error()),
		}, nil
	}

	ad, err := a.UpdateAgentLimit(req.GetCompanyId(), int(req.GetLimit()))
	if err != nil {
		return &pb.GetAgentLimitResponse{
			Status: pointerutil.StringPtr(err.Error()),
		}, nil
	}

	return &pb.GetAgentLimitResponse{
		Limit: int32(ad.Limit),
	}, nil
}

func (s *Server) UpdateUsedAgent(ctx context.Context, req *pb.UpdateUsedAgentRequest) (*pb.GetAgentLimitResponse, error) {
	a := NewSubscriptionService(ctx, s.Config, &RealMongoOperations{
		Collection: s.Mongo.Collections["subs"],
		Database:   s.Mongo.Database,
	})
	if err := a.MongoOps.GetMongoClient(ctx, a.Mongo); err != nil {
		return &pb.GetAgentLimitResponse{
			Status: pointerutil.StringPtr(err.Error()),
		}, nil
	}

	ad, err := a.UpdateUsedAgent(req.GetCompanyId(), int(req.GetUsed()))
	if err != nil {
		return &pb.GetAgentLimitResponse{
			Status: pointerutil.StringPtr(err.Error()),
		}, nil
	}

	return &pb.GetAgentLimitResponse{
		Used:  int32(ad.Agents.Used),
		Limit: int32(ad.Agents.Limit),
	}, nil
}
