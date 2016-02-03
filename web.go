package main

import (
	"bytes"
	"github.com/hoisie/web"
	"github.com/liquidgecka/gorc2"
	"encoding/json"
	"log"
	"os"
)

var (
	orc  = gorc2.NewClient(os.Getenv("ORC_KEY"))
	host = "api.orchestrate.io"
)

type Result struct {
	Value json.RawMessage `json:"value"`
}

type Results struct {
	Results []Result `json:"results"`
	Count   int      `json:"count"`
}

func main() {
	web.Config.StaticDir = "static"
	port := os.Getenv("PORT")
	web.Get("/api/([^/]+/?)", search)
	web.Run(":" + port)
}

func search(ctx *web.Context, collection string) {
	if host != "" {
		orc.APIHost = host
	}

	ctx.ContentType("json")
	ctx.SetHeader("Access-Control-Allow-Origin", "*", true)

	query := ctx.Params["query"]

	var err error

	c := orc.Collection(collection)

	searchParms := &gorc2.SearchQuery{
		Limit: int(100),
		Sort:  ctx.Params["sort"],
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
