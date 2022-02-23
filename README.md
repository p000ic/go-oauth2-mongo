# Mongo Storage for [OAuth 2.0 V4](https://github.com/go-oauth2/oauth2)

golang 1.17 =>

## Install

``` bash
go get -u -v github.com/initial-commit-hq/go-oauth2-mongo
```

## Usage

``` go
package main

import (
 "github.com/initial-commit-hq/go-oauth2-mongo"
 "github.com/go-oauth2/oauth2/v4/manage"
)

func main() {
 manager := manage.NewDefaultManager()

 // use mongodb token store
 manager.MapTokenStorage(
  mongo.NewTokenStore(mongo.NewConfig(
  "mongodb://127.0.0.1:27017", 
  "oauth2", 
  username, 
  password, 
  "oauth2")),
 )
 // ...
}
```

## MIT License

***Copyright (c) 2022 p000ic***
