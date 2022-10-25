package main

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

var webAddr string

func startWebServer() {
	g := gin.Default()
	g.StaticFS("/", http.Dir("./"))
	g.Run(webAddr)
}
