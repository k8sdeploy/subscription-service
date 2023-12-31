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

type AccountLimits struct {
	CompanyId     string `bson:"company_id"`
	Grandfathered bool   `bson:"grandfathered"`

	Agents   Agents   `bson:"agents"`
	Projects Projects `bson:"projects"`
}

type Agents struct {
	Limit int `bson:"limit"`
	Used  int `bson:"used"`
}

type Projects struct {
	Limit int `bson:"limit"`
	Used  int `bson:"used"`
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

func (s *Service) GetAgentLimit(companyId string) (*AccountLimits, error) {
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

	var accountLimits AccountLimits
	err := result.Decode(&accountLimits)
	if err != nil {
		return nil, logs.Errorf("error decoding agent limit: %v", err)
	}

	return &accountLimits, nil
}

func (s *Service) UpdateAgentLimit(companyId string, limit int) (*Agents, error) {
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
		bson.D{{"$set", bson.D{
			{"agents.limit", limit},
		}}}); err != nil {
		return nil, logs.Errorf("error updating agent limit: %v", err)
	}

	return &Agents{
		Limit: newMinLimit,
	}, nil
}

func (s *Service) UpdateUsedAgent(companyId string, used int) (*AccountLimits, error) {
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
		bson.D{{"$set", bson.D{
			{"agents.used", used},
		}}}); err != nil {
		return nil, logs.Errorf("error updating agent limit: %v", err)
	}

	return &AccountLimits{
		Agents: Agents{
			Limit: oldLimits.Agents.Limit,
			Used:  used,
		},
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
