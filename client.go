package main

import (
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"log"
)

type Accessor interface {
	Ping() error
	Write(uuid string, bsonResource map[string]interface{}) (err error, inserted bool)
	Get(collection, uuid string) (bsonResource map[string]interface{}, found bool, err error)
}

type config struct {
	session *mgo.Session
}

type dbClient struct {
	session *mgo.Session
}

func (c *dbClient) Ping() error {
	newSession := c.session.Copy()
	defer newSession.Close()
	return newSession.Ping()
}

func (c *dbClient) Write(uuid string, bsonResource map[string]interface{}) (error, bool) {
	newSession := c.session.Copy()
	defer newSession.Close()
	db := newSession.DB("")
	res, err := db.C("holiday").Upsert(bson.M{"uuid": uuid}, bsonResource)
	if err != nil {
		log.Fatalf("ERROR - MongoDB insert could not be created: %v\n", err)
	}
	return err, res.Matched == 0
}

func (c *dbClient) Get(collection, uuid string) (bsonResource map[string]interface{}, found bool, err error) {
	newSession := c.session.Copy()
	defer newSession.Close()

	coll := newSession.DB("").C(collection)

	if err = coll.Find(bson.M{"uuid": uuid}).One(&bsonResource); err != nil {
		if err == mgo.ErrNotFound {
			err = nil
			return
		}
		return
	}
	found = true
	return
}
