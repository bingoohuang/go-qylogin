package main

import (
	"fmt"
	"github.com/bingoohuang/go-utils"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func main() {
	r := mux.NewRouter()
	r.HandleFunc(appConfig.ContextPath+"/favicon.png", go_utils.ServeFavicon("res/favicon.png", MustAsset, AssetInfo))
	handleFunc(r, "/", serveHome)

	http.Handle("/", r)

	fmt.Println("start to listen at ", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

func handleFunc(r *mux.Router, path string, f func(http.ResponseWriter, *http.Request)) {
	wrap := go_utils.DumpRequest(f)
	r.HandleFunc(appConfig.ContextPath+path, MustAuth(wrap))
}

func (t *CookieValue) ExpiredTime() time.Time {
	return t.Expired
}

func MustAuth(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie := CookieValue{}
		cookieName := findCookieName(r)
		log.Println("MustAuth cookieName:", cookieName)

		err := go_utils.ReadCookie(r, appConfig.EncryptKey, cookieName, &cookie)
		log.Println("MustAuth cookie:", cookie)
		if wxloginCallback(w, r, &cookie) {
			fn(w, r) // 执行被装饰的函数
			return
		}

		if err == nil && cookie.Name != "" {
			fn(w, r) // 执行被装饰的函数
			return
		}

		agentId := go_utils.EmptyThen(r.FormValue("agentId"), appConfig.DefaultAgentId)
		csrfToken := agentId + "," + cookieName + "," + go_utils.RandString(10)
		cookie.Redirect = r.FormValue("redirect")
		cookie.CsrfToken = csrfToken
		cookie.Expired = time.Now().Add(time.Duration(8) * time.Hour)
		_ = go_utils.WriteDomainCookie(w, appConfig.CookieDomain, appConfig.EncryptKey, cookieName, &cookie)

		urlCreate := func(cropId, agentId, redirectUri, csrfToken string) string {
			return "https://open.work.weixin.qq.com/wwopen/sso/qrConnect?appid=" +
				cropId + "&agentid=" + agentId + "&redirect_uri=" + url.QueryEscape(redirectUri) + "&state=" + csrfToken
		}

		url := urlCreate(appConfig.CorpId, agentId, appConfig.RedirectUri, csrfToken)
		log.Println("wx login url:", url)

		// 301 redirect: 301 代表永久性转移(Permanently Moved)。
		// 302 redirect: 302 代表暂时性转移(Temporarily Moved )。
		http.Redirect(w, r, url, 302)
	}
}

func findCookieName(r *http.Request) string {
	cookieName := ""
	state := r.FormValue("state")
	if state != "" {
		log.Println("findCookieName state:", state)
		parts := strings.Split(state, ",")
		if len(parts) == 3 {
			cookieName = parts[1]
		}
	}
	if cookieName != "" {
		return cookieName
	}

	cookieName = r.FormValue("cookie")
	if cookieName != "" {
		return cookieName
	}

	return appConfig.CookieName
}

func wxloginCallback(w http.ResponseWriter, r *http.Request, cookie *CookieValue) bool {
	code := r.FormValue("code")
	state := r.FormValue("state")
	if code == "" {
		return false
	}

	fmt.Println("code:", code)
	fmt.Println("state:", state)

	stateInfo := strings.Split(state, ",")
	fmt.Println("stateInfo:", stateInfo)

	agentId := appConfig.DefaultAgentId
	randomStr := state
	if len(stateInfo) == 3 {
		agentId = stateInfo[0]
		randomStr = stateInfo[2]
	}
	fmt.Println("agentId:", agentId)

	secret := appConfig.Agents[agentId].Secret
	fmt.Println("secret:", secret)

	accessToken, err := go_utils.GetAccessToken(appConfig.CorpId, secret)
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

	sendLoginInfo(userInfo, randomStr, agentId, secret)

	cookie.UserId = userInfo.UserId
	cookie.Name = userInfo.Name
	cookie.Avatar = userInfo.Avatar
	cookie.CsrfToken = ""
	cookie.Expired = time.Now().Add(time.Duration(8) * time.Hour)

	cookieName := appConfig.CookieName
	if len(stateInfo) == 3 {
		cookieName = stateInfo[1]
	}

	_ = go_utils.WriteDomainCookie(w, appConfig.CookieDomain, appConfig.EncryptKey, cookieName, cookie)
	if cookie.Redirect != "" {
		http.Redirect(w, r, cookie.Redirect, 302)
	}

	return true
}

func sendLoginInfo(info *go_utils.WxUserInfo, randomStr, agentId, secret string) string {
	content := "用户[" + info.Name + "]正在扫码登录。"
	if randomStr == "qylogin" {
		content = "用户[" + info.Name + "]正在企业微信登录。"
	}

	accessToken, err := go_utils.SendWxQyMsg(appConfig.CorpId, secret, agentId, content)
	if err != nil {
		log.Println("sendLoginInfo error", err)
	}

	return accessToken
}
