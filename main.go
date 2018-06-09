package main

import (
	"github.com/gorilla/mux"
	"flag"
	"strconv"
	"strings"
	"net/http"
	"github.com/bingoohuang/go-utils"
	"time"
	"log"
	"github.com/BurntSushi/toml"
	"fmt"
)

var (
	contextPath string
	port        string

	encryptKey  string
	corpId      string
	corpSecret  string
	agentId     string
	redirectUri string

	cookieName string
)

func init() {
	contextPathArg := flag.String("contextPath", "", "context path")
	portArg := flag.Int("port", 10569, "Port to serve.")

	keyArg := flag.String("key", "", "key to encryption or decryption")
	corpIdArg := flag.String("corpId", "", "corpId")
	corpSecretArg := flag.String("corpSecret", "", "cropId")
	agentIdArg := flag.String("agentId", "", "agentId")
	redirectUriArg := flag.String("wxRedirectUri", "", "wxRedirectUri")
	cookieArg := flag.String("cookie", "i-raiyee-cn-auth", "cookie name")

	flag.Parse()

	contextPath = *contextPathArg
	if contextPath != "" && !strings.HasPrefix(contextPath, "/") {
		contextPath = "/" + contextPath
	}

	port = strconv.Itoa(*portArg)

	encryptKey = *keyArg
	corpId = *corpIdArg
	corpSecret = *corpSecretArg
	agentId = *agentIdArg
	redirectUri = *redirectUriArg

	cookieName = *cookieArg
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc(contextPath+"/favicon.png", go_utils.ServeFavicon("res/favicon.png", MustAsset, AssetInfo))
	handleFunc(r, "/", serveHome)

	http.Handle("/", r)

	fmt.Println("start to listen at ", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

type Link struct {
	LinkTo string
	Name   string
}
type Links struct {
	Links []Link
}

func serveHome(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	indexHtml := string(MustAsset("res/index.html"))
	html := go_utils.MinifyHtml(indexHtml, true)
	html = strings.Replace(html, "${contextPath}", contextPath, -1)

	var links Links
	if _, err := toml.DecodeFile("links.toml", &links); err != nil {
		log.Fatal(err)
	}

	linksHtml := ""
	for _, l := range links.Links {
		linksHtml += "<div><a href=\"" + l.LinkTo + "\">" + l.Name + "</a></div>"
	}

	html = strings.Replace(html, "<Links/>", linksHtml, -1)

	w.Write([]byte(html))
}

func handleFunc(r *mux.Router, path string, f func(http.ResponseWriter, *http.Request)) {
	wrap := go_utils.DumpRequest(f)
	r.HandleFunc(contextPath+path, MustAuth(wrap))
}

type CookieValue struct {
	UserId    string
	Name      string
	Avatar    string
	CsrfToken string
	Expired   time.Time
	Redirect  string
}

func (t *CookieValue) ExpiredTime() time.Time {
	return t.Expired
}

func MustAuth(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie := CookieValue{}
		go_utils.ReadCookie(r, encryptKey, cookieName, &cookie)
		if cookie.Name != "" {
			fn(w, r) // 执行被装饰的函数
			return
		}

		if wxloginCallback(w, r, &cookie) {
			fn(w, r) // 执行被装饰的函数
			return
		}

		csrfToken := go_utils.RandString(10)
		cookie.Redirect = r.FormValue("redirect")
		cookie.CsrfToken = csrfToken
		cookie.Expired = time.Now().Add(time.Duration(8) * time.Hour)
		go_utils.WriteCookie(w, encryptKey, cookieName, &cookie)
		url := go_utils.CreateWxQyLoginUrl(corpId, agentId, redirectUri, csrfToken)
		log.Println("wx login url:", url)

		// 301 redirect: 301 代表永久性转移(Permanently Moved)。
		// 302 redirect: 302 代表暂时性转移(Temporarily Moved )。
		http.Redirect(w, r, url, 302)
	}
}

func wxloginCallback(w http.ResponseWriter, r *http.Request, cookie *CookieValue) bool {
	if cookie.CsrfToken == "" {
		return false
	}

	code := r.FormValue("code")
	state := r.FormValue("state")
	if code == "" || state != cookie.CsrfToken {
		return false
	}

	accessToken, err := go_utils.GetAccessToken(corpId, corpSecret)
	if err != nil {
		return false
	}
	userId, err := go_utils.GetLoginUserId(accessToken, code)
	if err != nil {
		return false
	}
	userInfo, err := go_utils.GetUserInfo(accessToken, userId)
	if err != nil {
		return false
	}

	sendLoginInfo(userInfo, accessToken)

	cookie.UserId = userInfo.UserId
	cookie.Name = userInfo.Name
	cookie.Avatar = userInfo.Avatar
	cookie.CsrfToken = ""
	cookie.Expired = time.Now().Add(time.Duration(8) * time.Hour)

	go_utils.WriteCookie(w, encryptKey, cookieName, cookie)

	if cookie.Redirect != "" {
		http.Redirect(w, r, cookie.Redirect, 302)
	}

	return true
}

func sendLoginInfo(info *go_utils.WxUserInfo, accessToken string) {
	msg := map[string]interface{}{
		"touser": "@all", "toparty": "@all", "totag": "@all", "msgtype": "text", "agentid": agentId, "safe": 0,
		"text": map[string]string{
			"content": "用户[" + info.Name + "]正在扫码登录。",
		},
	}
	_, err := go_utils.HttpPost("https://qyapi.weixin.qq.com/cgi-bin/message/send?access_token="+accessToken, msg)
	if err != nil {
		log.Println("sendLoginInfo error", err)
	}
}
