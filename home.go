package main

import (
	"net/http"
	"github.com/bingoohuang/go-utils"
	"strings"
)

func serveHome(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	indexHtml := string(MustAsset("res/index.html"))
	html := go_utils.MinifyHtml(indexHtml, true)
	html = strings.Replace(html, "${contextPath}", contextPath, -1)

	linksHtml := ""
	for _, l := range links.Links {
		linksHtml += "<div><a href=\"" + l.LinkTo + "\">" + l.Name + "</a></div>"
	}

	html = strings.Replace(html, "<Links/>", linksHtml, -1)

	w.Write([]byte(html))
}
