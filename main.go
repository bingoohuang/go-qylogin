package main

import (
	"fmt"
	"github.com/bingoohuang/go-utils"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"time"
)

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
		if wxloginCallback(w, r, &cookie) {
			fn(w, r) // 执行被装饰的函数
			return
		}

		err := go_utils.ReadCookie(r, *authParam.EncryptKey, *authParam.CookieName, &cookie)
		log.Println("cookie:", cookie)
		if err == nil && cookie.Name != "" {
			fn(w, r) // 执行被装饰的函数
			return
		}

		csrfToken := go_utils.RandString(10)
		cookie.Redirect = r.FormValue("redirect")
		cookie.CsrfToken = csrfToken
		cookie.Expired = time.Now().Add(time.Duration(8) * time.Hour)
		go_utils.WriteDomainCookie(w, cookieDomain, *authParam.EncryptKey, *authParam.CookieName, &cookie)
		url := go_utils.CreateWxQyLoginUrl(corpId, agentId, *authParam.RedirectUri, csrfToken)
		log.Println("wx login url:", url)

		// 301 redirect: 301 代表永久性转移(Permanently Moved)。
		// 302 redirect: 302 代表暂时性转移(Temporarily Moved )。
		http.Redirect(w, r, url, 302)
	}
}

func wxloginCallback(w http.ResponseWriter, r *http.Request, cookie *CookieValue) bool {
	code := r.FormValue("code")
	state := r.FormValue("state")
	if code == "" {
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

	sendLoginInfo(userInfo, accessToken, state)

	cookie.UserId = userInfo.UserId
	cookie.Name = userInfo.Name
	cookie.Avatar = userInfo.Avatar
	cookie.CsrfToken = ""
	cookie.Expired = time.Now().Add(time.Duration(8) * time.Hour)
	go_utils.WriteDomainCookie(w, cookieDomain, *authParam.EncryptKey, *authParam.CookieName, cookie)
	if cookie.Redirect != "" {
		http.Redirect(w, r, cookie.Redirect, 302)
	}

	return true
}

func sendLoginInfo(info *go_utils.WxUserInfo, accessToken, state string) {
	content := "用户[" + info.Name + "]正在电脑扫码登录。"
	if state == "qylogin" {
		content = "用户[" + info.Name + "]正在企业微信登录。"
	}

	msg := map[string]interface{}{
		"touser": "@all", "toparty": "@all", "totag": "@all", "msgtype": "text", "agentid": agentId, "safe": 0,
		"text": map[string]string{
			"content": content,
		},
	}
	_, err := go_utils.HttpPost("https://qyapi.weixin.qq.com/cgi-bin/message/send?access_token="+accessToken, msg)
	if err != nil {
		log.Println("sendLoginInfo error", err)
	}
}
