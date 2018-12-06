package main

import (
	"flag"
	"fmt"
	"github.com/BurntSushi/toml"
	"log"
	"strconv"
	"strings"
	"time"
)

var (
	port string
)

type QywxAgent struct {
	AgentId string
	Secret  string
}

type AppConfig struct {
	CorpId         string
	DefaultAgentId string
	ContextPath    string
	Port           int
	CookieDomain   string

	Agents map[string]QywxAgent

	EncryptKey  string
	RedirectUri string
	CookieName  string
}

var configFile string
var appConfig AppConfig

type CookieValue struct {
	UserId    string
	Name      string
	Avatar    string
	CsrfToken string
	Expired   time.Time
	Redirect  string
}

func init() {
	flag.StringVar(&configFile, "configFile", "appConfig.toml", "config file path")
	flag.Parse()

	if _, err := toml.DecodeFile(configFile, &appConfig); err != nil {
		log.Panic("config file decode error", err.Error())
	}

	fmt.Println("appConfig:", appConfig)

	if appConfig.ContextPath != "" && !strings.HasPrefix(appConfig.ContextPath, "/") {
		appConfig.ContextPath = "/" + appConfig.ContextPath
	}

	port = strconv.Itoa(appConfig.Port)
}
