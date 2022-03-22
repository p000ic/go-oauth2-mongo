package mongo

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/go-oauth2/oauth2/v4"
	"github.com/go-oauth2/oauth2/v4/models"
	"github.com/google/uuid"
	"github.com/qiniu/qmgo"
	"github.com/qiniu/qmgo/options"
	"go.mongodb.org/mongo-driver/bson/primitive"
	mongoOpts "go.mongodb.org/mongo-driver/mongo/options"
)

type txn struct {
	Collection string
	Id         string
	Insert     interface{}
}

type basicData struct {
	ID      primitive.ObjectID `bson:"_id,omitempty"`
	Code    string             `bson:"code,omitempty"`
	Data    []byte             `bson:"data,omitempty"`
	Expires *bool              `bson:"expires,omitempty"`
	TTL     time.Time          `bson:"ttl,omitempty"`
}

type tokenData struct {
	ID      primitive.ObjectID `bson:"_id,omitempty"`
	BasicID string             `bson:"basic_id,omitempty"`
	Token   string             `bson:"token,omitempty"`
	Expires *bool              `bson:"expires,omitempty"`
	TTL     time.Time          `bson:"ttl,omitempty"`
}

// TokenConfig token configuration parameters
type TokenConfig struct {
	// store txn collection name(The default is oauth2)
	TxnCName string
	// store token based data collection name(The default is oauth2_basic)
	BasicCName string
	// store access token data collection name(The default is oauth2_access)
	AccessCName string
	// store refresh token data collection name(The default is oauth2_refresh)
	RefreshCName string
}

// TokenStore MongoDB storage for OAuth 2.0
type TokenStore struct {
	tcfg *TokenConfig
	dbase
}

var (
	tRUEts  = true
	fALSEts = false
	storeTS = &TokenStore{}
)

// NewDefaultTokenConfig create a default token configuration
func NewDefaultTokenConfig() *TokenConfig {
	return &TokenConfig{
		TxnCName:     "oauth2_txn",
		BasicCName:   "oauth2_basic",
		AccessCName:  "oauth2_access",
		RefreshCName: "oauth2_refresh",
	}
}

// NewTokenStore create a token store instance based on mongodb
func NewTokenStore(cfg *Config, tcfgs ...*TokenConfig) *TokenStore {
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

	storeTS.ctx = ctx

	storeTS.client, err = qmgo.NewClient(ctx, &dbConfig)
	if err != nil {
		log.Print(err)
	}

	storeTS.dbname = cfg.DB

	storeTS.source = storeTS.client.Database(cfg.DB)

	store := NewTokenStoreWithSession(storeTS, tcfgs...)

	return store
}

// NewTokenStoreWithSession create a token store instance based on mongodb
func NewTokenStoreWithSession(ts *TokenStore, tcfgs ...*TokenConfig) *TokenStore {
	ts.cloneSession()
	defer ts.session.EndSession(ts.ctx)

	ts.tcfg = NewDefaultTokenConfig()
	if len(tcfgs) > 0 {
		ts.tcfg = tcfgs[0]
	}
	exp := int32(1)
	_ = ts.c(ts.tcfg.BasicCName).CreateIndexes(ts.ctx, []options.IndexModel{{
		Key: []string{"ExpiredAt"},
		IndexOptions: &mongoOpts.IndexOptions{
			ExpireAfterSeconds: &exp,
		}},
	})
	_ = ts.c(ts.tcfg.AccessCName).CreateIndexes(ts.ctx, []options.IndexModel{{
		Key: []string{"ExpiredAt"},
		IndexOptions: &mongoOpts.IndexOptions{
			ExpireAfterSeconds: &exp,
		}},
	})

	_ = ts.c(ts.tcfg.RefreshCName).CreateIndexes(ts.ctx, []options.IndexModel{{
		Key: []string{"ExpiredAt"},
		IndexOptions: &mongoOpts.IndexOptions{
			ExpireAfterSeconds: &exp,
		}},
	})
	return ts
}

// Close close the mongo session
func (ts *TokenStore) Close() {
	ts.client.Close(ts.ctx)
}

func (ts *TokenStore) cloneSession() (*TokenStore, error) {
	var err error
	ts.session, err = ts.client.Session()
	if err != nil {
		return ts, err
	}
	return ts, nil
}

func (ts *TokenStore) c(cltn string) *qmgo.Collection {
	return ts.source.Collection(cltn)
}

func (ts *TokenStore) cHandler(cltn string, handler func(c *qmgo.Collection)) {
	ts.client.Session()
	defer ts.session.EndSession(ts.ctx)
	handler(ts.source.Collection(cltn))
}

// Create create and store the new token information
func (ts *TokenStore) Create(ctx context.Context, info oauth2.TokenInfo) (err error) {
	jsonInfo, err := json.Marshal(info)
	if err != nil {
		return
	}

	id := primitive.NewObjectID()
	if code := info.GetCode(); code != "" {
		ts.cHandler(ts.tcfg.BasicCName, func(c *qmgo.Collection) {
			if _, cerr := c.InsertOne(ts.ctx, basicData{
				ID:      id,
				Code:    code,
				Data:    jsonInfo,
				Expires: &tRUEts,
				TTL:     info.GetCodeCreateAt().Add(info.GetCodeExpiresIn()),
			}); cerr != nil {
				err = cerr
				return
			}
		})
	}
	basicID := uuid.Must(uuid.NewRandom()).String()
	aexp := info.GetAccessCreateAt().Add(info.GetAccessExpiresIn())
	rexp := aexp
	expires := true
	if refresh := info.GetRefresh(); refresh != "" {
		rexp = info.GetRefreshCreateAt().Add(info.GetRefreshExpiresIn())
		if aexp.Second() > rexp.Second() {
			aexp = rexp
		}
		expires = info.GetRefreshExpiresIn() != 0
		ts.cHandler(ts.tcfg.RefreshCName, func(c *qmgo.Collection) {
			if _, cerr := c.InsertOne(ts.ctx, tokenData{
				ID:      id,
				BasicID: basicID,
				Token:   refresh,
				Expires: &expires,
				TTL:     rexp,
			}); cerr != nil {
				err = cerr
				return
			}
		})
	}
	ops := []txn{{
		Collection: ts.tcfg.BasicCName,
		Insert: basicData{
			ID:      id,
			Code:    basicID,
			Data:    jsonInfo,
			Expires: &expires,
			TTL:     rexp,
		},
	}, {
		Collection: ts.tcfg.AccessCName,
		Insert: tokenData{
			ID:      id,
			BasicID: basicID,
			Token:   info.GetAccess(),
			Expires: &expires,
			TTL:     aexp,
		},
	}}
	for _, o := range ops {
		ts.cHandler(o.Collection, func(c *qmgo.Collection) {
			if _, err = c.InsertOne(ts.ctx, o.Insert); err != nil {
				return
			}
		})
	}
	return
}

// RemoveByCode use the authorization code to delete the token information
func (ts *TokenStore) RemoveByCode(ctx context.Context, code string) (err error) {
	ts.cloneSession()
	verr := ts.c(ts.tcfg.BasicCName).Remove(ts.ctx, basicData{Code: code})
	if verr != nil {
		if verr == qmgo.ErrNoSuchDocuments {
			return
		}
		err = verr
	}
	return
}

// RemoveByAccess use the access token to delete the token information
func (ts *TokenStore) RemoveByAccess(ctx context.Context, access string) (err error) {
	ts.cloneSession()
	basicID, err := ts.getByToken(ts.tcfg.AccessCName, access)
	fn := func(sCtx context.Context) (interface{}, error) {
		verr := ts.c(ts.tcfg.AccessCName).Remove(ts.ctx, tokenData{Token: access})
		if verr != nil {
			if verr == qmgo.ErrNoSuchDocuments {
				return nil, nil
			}
			err = verr
		}
		verr = ts.c(ts.tcfg.BasicCName).Remove(ts.ctx, basicData{Code: basicID})
		if verr != nil {
			if verr == qmgo.ErrNoSuchDocuments {
				return nil, nil
			}
			err = verr
		}
		return nil, nil
	}
	_, err = ts.session.StartTransaction(ts.ctx, fn)
	return
}

// RemoveByRefresh use the refresh token to delete the token information
func (ts *TokenStore) RemoveByRefresh(ctx context.Context, refresh string) (err error) {
	ts.cloneSession()
	basicID, err := ts.getByToken(ts.tcfg.RefreshCName, refresh)
	fn := func(sCtx context.Context) (interface{}, error) {
		verr := ts.c(ts.tcfg.RefreshCName).Remove(ts.ctx, tokenData{Token: refresh})
		if verr != nil {
			if verr == qmgo.ErrNoSuchDocuments {
				return nil, nil
			}
			err = verr
		}
		verr = ts.c(ts.tcfg.BasicCName).Remove(ts.ctx, basicData{Code: basicID})
		if verr != nil {
			if verr == qmgo.ErrNoSuchDocuments {
				return nil, nil
			}
			err = verr
		}
		return nil, nil
	}
	_, err = ts.session.StartTransaction(ts.ctx, fn)
	return
}

func (ts *TokenStore) getData(basicID string) (ti oauth2.TokenInfo, err error) {
	ts.cloneSession()
	var bd basicData
	bd.Code = basicID
	verr := ts.c(ts.tcfg.BasicCName).Find(ts.ctx, bd).One(&bd)
	if verr != nil {
		if verr == qmgo.ErrNoSuchDocuments {
			return nil, nil
		}
		err = verr
		return nil, err
	}
	var tm models.Token
	err = json.Unmarshal(bd.Data, &tm)
	if err != nil {
		return nil, err
	}
	ti = &tm
	return
}

func (ts *TokenStore) getByToken(cltn, token string) (basicID string, err error) {
	ts.cloneSession()
	var td tokenData
	err = ts.c(cltn).Find(ts.ctx, tokenData{Token: token}).One(&td)
	if err != nil {
		if err == qmgo.ErrNoSuchDocuments {
			return "", nil
		}
		return
	}
	return td.BasicID, nil
}

// GetByCode use the authorization code for token information data
func (ts *TokenStore) GetByCode(ctx context.Context, code string) (ti oauth2.TokenInfo, err error) {
	ti, err = ts.getData(code)
	return
}

// GetByAccess use the access token for token information data
func (ts *TokenStore) GetByAccess(ctx context.Context, access string) (ti oauth2.TokenInfo, err error) {
	basicID, err := ts.getByToken(ts.tcfg.AccessCName, access)
	if err != nil && basicID == "" {
		return
	}
	ti, err = ts.getData(basicID)
	return
}

// GetByRefresh use the refresh token for token information data
func (ts *TokenStore) GetByRefresh(ctx context.Context, refresh string) (ti oauth2.TokenInfo, err error) {
	basicID, err := ts.getByToken(ts.tcfg.RefreshCName, refresh)
	if err != nil && basicID == "" {
		return
	}
	ti, err = ts.getData(basicID)
	return
}
