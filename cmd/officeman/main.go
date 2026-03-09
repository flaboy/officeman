package main

import (
	"log"

	"github.com/github-flaboy/officeman/internal/buildinfo"
)

func main() {
	log.Printf("officeman starting version=%s", buildinfo.Version)
}
