package main

import (
	"bytes"
	"github.com/liquidgecka/gorc2"
	"encoding/json"
	"github.com/hoisie/web"
	"log"
	"os"
	"strconv"
)

var (
	orc = gorc2.NewClient(os.Getenv("ORC_KEY"))
	host = os.Getenv("ORC_HOST")
)

type Result struct {
	Value json.RawMessage `json:"value"`
}

type Results struct {
	Results []Result `json:"results"`
	Count int `json:"count"`
}

func main() {
	web.Config.StaticDir = "static"
	port := os.Getenv("PORT")
	log.Printf("Listening on port %v ...", port)
	web.Get("/api/([^/]+/?)", search)
	web.Run(":" + port)
}

func search(ctx *web.Context, collection string) {
	if (host != "") {
		orc.APIHost = host
	}

	ctx.ContentType("json")
	ctx.SetHeader("Access-Control-Allow-Origin", "*", true)

	query := ctx.Params["query"]

	var limit, offset int64
	var err error

	if limit, err = strconv.ParseInt(ctx.Params["limit"], 10, 32); err != nil {
		limit = 10
		err = nil
	}
	if offset, err = strconv.ParseInt(ctx.Params["offset"], 10, 32); err != nil {
		offset = 0
		err = nil
	}

	c := orc.Collection(collection)

	searchParms := &gorc2.SearchQuery{
		Limit: int(limit),
		Offset: offset,
		Sort: ctx.Params["sort"],
	}

	it := c.Search(query, searchParms)

	results := Results{}

	for i := 0; it.Next(); i++ {
		if it.Error != nil {
			err = it.Error
			break
		}

		result := Result{}

	  if _, err := it.Get(&result.Value); err != nil {
			log.Println(err)
			break
		}

		results.Results = append(results.Results, result)
	}

	results.Count = len(results.Results)

	buf := new(bytes.Buffer)
	encoder := json.NewEncoder(buf)

	if err != nil {
		encoder.Encode(err)
		log.Println(err)
	} else {
		encoder.Encode(&results)
	}


	ctx.Write(buf.Bytes())
}
