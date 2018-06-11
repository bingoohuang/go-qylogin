package main

import (
	"github.com/bingoohuang/go-utils"
	"flag"
	"strings"
	"strconv"
)

var (
	contextPath string
	port        string

	corpId     string
	corpSecret string
	agentId    string

	cookieDomain string

	authParam go_utils.MustAuthParam
)

func init() {
	contextPathArg := flag.String("contextPath", "", "context path")
	portArg := flag.Int("port", 10569, "Port to serve.")

	corpIdArg := flag.String("corpId", "", "corpId")
	corpSecretArg := flag.String("corpSecret", "", "cropId")
	agentIdArg := flag.String("agentId", "", "agentId")
	cookieDomainArg := flag.String("cookieDomain", "raiyee.cn", "cookie domain")

	go_utils.PrepareMustAuthFlag(&authParam)

	flag.Parse()

	contextPath = *contextPathArg
	if contextPath != "" && !strings.HasPrefix(contextPath, "/") {
		contextPath = "/" + contextPath
	}

	port = strconv.Itoa(*portArg)

	corpId = *corpIdArg
	corpSecret = *corpSecretArg
	agentId = *agentIdArg
	cookieDomain = *cookieDomainArg
}
