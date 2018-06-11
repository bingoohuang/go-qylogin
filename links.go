package main

import (
	"github.com/BurntSushi/toml"
	"log"
)

type Link struct {
	LinkTo string
	Name   string
}
type Links struct {
	Links []Link
}

var links Links

func init() {
	if _, err := toml.DecodeFile("links.toml", &links); err != nil {
		log.Fatal(err)
	}

}
