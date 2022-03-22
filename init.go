package mongo

import (
	"context"
	"log"
	"strconv"

	"github.com/qiniu/qmgo"
)

var (
	auth       bool
	username   string
	password   string
	host       string
	port       int
	source     string
	authsource string
	mongo      = &dbase{}
)

type dbase struct {
	ctx        context.Context
	session    *qmgo.Session
	source     *qmgo.Database
	client     *qmgo.Client
	authDBName string
	dbname     string
}

// testConfig - initialize Mongo for tests
func testConfig() {
	var err error
	ctx := context.Background()
	connStr := "mongodb://" + host + ":" + strconv.Itoa(port)
	dbConfig := qmgo.Config{
		Uri: connStr,
	}
	if auth {
		dbConfig.Auth = &qmgo.Credential{
			AuthMechanism: "SCRAM-SHA-1",
			Username:      username,
			Password:      password,
			AuthSource:    authsource,
		}
	}

	mongo.ctx = ctx

	mongo.client, err = qmgo.NewClient(ctx, &dbConfig)
	if err != nil {
		log.Print(err)
	}

	mongo.dbname = source

	mongo.source = mongo.client.Database(source)
}

// cloneSession - cloneSession to Database
func cloneSession() (*dbase, error) {
	var err error
	mongo.session, err = mongo.client.Session()
	if err != nil {
		return mongo, err
	}
	return mongo, nil
}
