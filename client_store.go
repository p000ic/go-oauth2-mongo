package mongo

import (
	"context"
	"log"

	"github.com/go-oauth2/oauth2/v4"
	"github.com/go-oauth2/oauth2/v4/models"
	"github.com/qiniu/qmgo"
	"github.com/qiniu/qmgo/options"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type client struct {
	internalID primitive.ObjectID `bson:"_id"`
	ID         string             `bson:"id"`
	Secret     string             `bson:"secret"`
	Domain     string             `bson:"domain"`
	UserID     string             `bson:"userid"`
}

// ClientConfig client configuration parameters
type ClientConfig struct {
	// store clients data collection name(The default is oauth2_clients)
	ClientsCName string
}

// ClientStore MongoDB storage for OAuth 2.0
type ClientStore struct {
	ccfg *ClientConfig
	dbase
}

var (
	storeCS = &ClientStore{}
)

// NewDefaultClientConfig create a default client configuration
func NewDefaultClientConfig() *ClientConfig {
	return &ClientConfig{
		ClientsCName: "oauth2_clients",
	}
}

// NewClientStore create a client store instance based on mongodb
func NewClientStore(cfg *Config, ccfgs ...*ClientConfig) *ClientStore {
	var err error
	ctx := context.Background()
	dbConfig := qmgo.Config{Uri: cfg.URL}
	if auth {
		dbConfig.Auth = &qmgo.Credential{
			AuthMechanism: cfg.AuthMechanism,
			Username:      cfg.Username,
			Password:      cfg.Password,
			AuthSource:    cfg.DB,
		}
	}

	storeCS.ctx = ctx

	storeCS.client, err = qmgo.NewClient(ctx, &dbConfig)
	if err != nil {
		log.Print(err)
	}

	storeCS.dbname = cfg.DB

	storeCS.source = storeCS.client.Database(cfg.DB)

	store := NewClientStoreWithSession(storeCS, ccfgs...)

	return store
}

// NewClientStoreWithSession create a client store instance based on mongodb
func NewClientStoreWithSession(cs *ClientStore, ccfgs ...*ClientConfig) *ClientStore {
	cs.cloneSession()
	defer cs.session.EndSession(cs.ctx)

	cs.ccfg = NewDefaultClientConfig()
	if len(ccfgs) > 0 {
		cs.ccfg = ccfgs[0]
	}

	_ = cs.c(cs.ccfg.ClientsCName).CreateIndexes(cs.ctx, []options.IndexModel{{
		Key:    []string{"id"},
		Unique: true},
	})

	return cs
}

// Close close the mongo session
func (cs *ClientStore) Close() {
	cs.client.Close(cs.ctx)
}

func (cs *ClientStore) cloneSession() (*ClientStore, error) {
	var err error
	cs.session, err = cs.client.Session()
	if err != nil {
		return cs, err
	}
	return cs, nil
}

func (cs *ClientStore) c(cltn string) *qmgo.Collection {
	return cs.source.Collection(cltn)
}

func (cs *ClientStore) cHandler(cltn string, handler func(c *qmgo.Collection)) {
	cs.client.Session()
	defer cs.session.EndSession(cs.ctx)
	handler(cs.source.Collection(cltn))
}

// Set set client information
func (cs *ClientStore) Set(info oauth2.ClientInfo) (err error) {
	cs.cloneSession()
	cs.cHandler(cs.ccfg.ClientsCName, func(c *qmgo.Collection) {
		entity := &client{
			ID:     info.GetID(),
			Secret: info.GetSecret(),
			Domain: info.GetDomain(),
			UserID: info.GetUserID(),
		}

		if _, cerr := c.InsertOne(cs.ctx, entity); cerr != nil {
			err = cerr
			return
		}
	})
	return
}

// GetByID according to the ID for the client information
func (cs *ClientStore) GetByID(ctx context.Context, id string) (info oauth2.ClientInfo, err error) {
	cs.cloneSession()
	err = nil
	cs.cHandler(cs.ccfg.ClientsCName, func(c *qmgo.Collection) {
		entity := new(client)
		cerr := c.Find(cs.ctx, bson.M{"id": id}).One(entity)
		if cerr != nil {
			err = cerr
			return
		}
		info = &models.Client{
			ID:     entity.ID,
			Secret: entity.Secret,
			Domain: entity.Domain,
			UserID: entity.UserID,
		}
	})
	return
}

// RemoveByID use the client id to delete the client information
func (cs *ClientStore) RemoveByID(ctx context.Context, id string) error {
	cs.cloneSession()
	err := cs.c(cs.ccfg.ClientsCName).Remove(cs.ctx, client{ID: id})
	if err != nil {
		return err
	}
	return nil
}
