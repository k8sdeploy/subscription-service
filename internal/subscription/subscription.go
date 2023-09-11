package subscription

import (
	"context"
	"github.com/bugfixes/go-bugfixes/logs"
	"github.com/k8sdeploy/subscription-service/internal/config"
	mungo "github.com/keloran/go-config/mongo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type MongoOperations interface {
	GetMongoClient(ctx context.Context, config mungo.Mongo) error
	Disconnect(ctx context.Context) error
	InsertOne(ctx context.Context, document interface{}) (interface{}, error)
	UpdateOne(ctx context.Context, filter interface{}, update interface{}) (interface{}, error)
	FindOne(ctx context.Context, filter interface{}) *mongo.SingleResult
}

type RealMongoOperations struct {
	Client     *mongo.Client
	Collection string
	Database   string
}

func (r *RealMongoOperations) GetMongoClient(ctx context.Context, config mungo.Mongo) error {
	client, err := mungo.GetMongoClient(ctx, config)
	if err != nil {
		return logs.Errorf("error getting mongo client: %v", err)
	}
	r.Client = client
	return nil
}
func (r *RealMongoOperations) Disconnect(ctx context.Context) error {
	return r.Client.Disconnect(ctx)
}
func (r *RealMongoOperations) InsertOne(ctx context.Context, document interface{}) (interface{}, error) {
	return r.Client.Database(r.Database).Collection(r.Collection).InsertOne(ctx, document)
}
func (r *RealMongoOperations) UpdateOne(ctx context.Context, filter interface{}, update interface{}) (interface{}, error) {
	return r.Client.Database(r.Database).Collection(r.Collection).UpdateOne(ctx, filter, update)
}
func (r *RealMongoOperations) FindOne(ctx context.Context, filter interface{}) *mongo.SingleResult {
	return r.Client.Database(r.Database).Collection(r.Collection).FindOne(ctx, filter)
}

type AgentLimits struct {
	CompanyId     string `bson:"company_id"`
	AgentLimit    int    `bson:"limit"`
	UsedAgents    int    `bson:"used"`
	Grandfathered bool   `bson:"grandfathered"`
}

type Service struct {
	config.Config
	context.Context

	MongoOps MongoOperations
}

func NewSubscriptionService(ctx context.Context, cfg config.Config, ops MongoOperations) *Service {
	return &Service{
		Config:   cfg,
		Context:  ctx,
		MongoOps: ops,
	}
}

func (s *Service) GetAgentLimit(companyId string) (*AgentLimits, error) {
	filter := map[string]interface{}{
		"company_id": companyId,
	}

	if err := s.MongoOps.GetMongoClient(s.Context, s.Config.Mongo); err != nil {
		return nil, logs.Errorf("error getting mongo client: %v", err)
	}
	defer func() {
		if err := s.MongoOps.Disconnect(s.Context); err != nil {
			_ = logs.Errorf("error disconnecting mongo: %v", err)
		}
	}()

	result := s.MongoOps.FindOne(s.Context, filter)
	if result.Err() != nil {
		return nil, logs.Errorf("error finding agent limit: %v", result.Err())
	}

	var agentLimit AgentLimits
	err := result.Decode(&agentLimit)
	if err != nil {
		return nil, logs.Errorf("error decoding agent limit: %v", err)
	}

	return &agentLimit, nil
}

func (s *Service) UpdateAgentLimit(companyId string, limit int) (*AgentLimits, error) {
	oldLimits, err := s.GetAgentLimit(companyId)
	if err != nil {
		return nil, logs.Errorf("error getting agent limit: %v", err)
	}

	if err := s.MongoOps.GetMongoClient(s.Context, s.Config.Mongo); err != nil {
		return nil, logs.Errorf("error getting mongo client: %v", err)
	}
	defer func() {
		if err := s.MongoOps.Disconnect(s.Context); err != nil {
			_ = logs.Errorf("error disconnecting mongo: %v", err)
		}
	}()

	newMinLimit := s.getLowLimit(limit, oldLimits.Grandfathered)
	if _, err := s.MongoOps.UpdateOne(s.Context,
		bson.D{{"company_id", companyId}},
		bson.D{{"$set", bson.M{
			"limit": newMinLimit,
		}}}); err != nil {
		return nil, logs.Errorf("error updating agent limit: %v", err)
	}

	return &AgentLimits{
		AgentLimit: newMinLimit,
	}, nil
}

func (s *Service) UpdateUsedAgent(companyId string, used int) (*AgentLimits, error) {
	oldLimits, err := s.GetAgentLimit(companyId)
	if err != nil {
		return nil, logs.Errorf("error getting agent limit: %v", err)
	}

	if err := s.MongoOps.GetMongoClient(s.Context, s.Config.Mongo); err != nil {
		return nil, logs.Errorf("error getting mongo client: %v", err)
	}
	defer func() {
		if err := s.MongoOps.Disconnect(s.Context); err != nil {
			_ = logs.Errorf("error disconnecting mongo: %v", err)
		}
	}()

	if _, err := s.MongoOps.UpdateOne(s.Context,
		bson.D{{"company_id", companyId}},
		bson.D{{"$set", bson.M{
			"used": used,
		}}}); err != nil {
		return nil, logs.Errorf("error updating agent limit: %v", err)
	}

	return &AgentLimits{
		AgentLimit: oldLimits.AgentLimit,
		UsedAgents: used,
	}, nil
}

func (s *Service) getLowLimit(newLimit int, grandFathered bool) int {
	newMinLimit := s.Config.K8sDeploy.MinimumAgents
	if grandFathered {
		newMinLimit = s.Config.K8sDeploy.MinimumGrandfatheredAgents
	}
	if newLimit < newMinLimit {
		newLimit = newMinLimit
	}

	return newLimit
}
