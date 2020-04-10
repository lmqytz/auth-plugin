package main

import (
	"encoding/json"
	"github.com/Kong/go-pdk"
	"net/http"
	"strconv"
	"time"
)

type Hook struct{}

func New() interface{} {
	return &Hook{}
}

type Handler struct {
	Redis      *Redis
	HttpClient *HttpClient
	Codec      *Codec
}

type SessionData struct {
	LastVisit int64 `json:"last_visit"`
}

type handlerFunc func(kong *pdk.PDK)

var (
	responseFunc func(kong *pdk.PDK, status int, message string, data map[string]interface{}, header map[string]string)
	handler      Handler
	handlers     map[string]handlerFunc
)

func init() {
	InitConfig()
	redisHandler := NewRedis(Config.RedisConf)
	httpClient := NewHttpClient(Config.HttpConf)
	codec := &Codec{Secret: Config.Token.Secret}

	handler = Handler{
		Redis:      redisHandler,
		HttpClient: httpClient,
		Codec:      codec,
	}

	//注册handler
	handlers[Config.LoginUrl] = handler.Login
	handlers[Config.LoginOutUrl] = handler.LoginOut

	responseFunc = func(kong *pdk.PDK, status int, message string, data map[string]interface{}, header map[string]string) {
		var response ResponseData
		response.Boolean = status
		response.Message = message
		response.Data = data

		h := make(http.Header)
		h.Set("Content-Type", "application/json")
		if len(header) > 0 {
			for k, v := range header {
				h.Set(k, v)
			}
		}

		body, _ := json.Marshal(response)

		if status == 0 {
			_ = kong.Log.Err(message)
		}

		kong.Response.Exit(200, string(body), h)
	}
}

func (hook *Hook) Access(kong *pdk.PDK) {
	path, _ := kong.Request.GetPath()

	if f, ok := handlers[path]; ok {
		f(kong)
	} else {
		handler.Common(kong)
	}
}

func (h *Handler) Login(kong *pdk.PDK) {
	query, err := kong.Request.GetRawQuery()
	if err != nil {
		responseFunc(kong, 0, err.Error(), nil, nil)
		return
	}

	resp, err := h.HttpClient.Post(Config.HttpConf.LoginUrl, query)
	if err != nil {
		responseFunc(kong, 0, err.Error(), nil, nil)
		return
	}

	if resp.Boolean == 0 {
		responseFunc(kong, 0, resp.Message, nil, nil)
		return
	}

	now := time.Now().Unix()
	payload := map[string]interface{}{
		"uid":  resp.Data["uid"],
		"time": now,
	}

	token, _ := h.Codec.createToken(payload)
	hashValue, _ := json.Marshal(SessionData{LastVisit: now})
	_, err = h.Redis.Do("HSET", Config.Token.Name, resp.Data["uid"], string(hashValue))
	if err != nil {
		responseFunc(kong, 0, resp.Message, nil, nil)
		return
	}

	header := make(map[string]string)
	header[Config.Token.Name] = token
	responseFunc(kong, 0, "登录成功", resp.Data, header)
}

func (h *Handler) LoginOut(kong *pdk.PDK) {
	jwtToken, err := kong.Request.GetHeader(Config.Token.Name)
	if jwtToken == "" {
		responseFunc(kong, 0, "token无效", nil, nil)
		return
	}

	if err != nil {
		responseFunc(kong, 0, err.Error(), nil, nil)
		return
	}

	payload, err := h.Codec.parseToken(jwtToken)
	if err != nil {
		responseFunc(kong, 0, err.Error(), nil, nil)
		return
	}

	userData, err := h.Redis.Do("HGET", Config.Token.Name, payload["uid"])
	if err != nil {
		responseFunc(kong, 0, err.Error(), nil, nil)
		return
	}

	if userData == nil {
		responseFunc(kong, 0, "尚未登录", nil, nil)
		return
	}

	query, err := kong.Request.GetRawQuery()
	resp, err := h.HttpClient.Post(Config.HttpConf.LoginOutUrl, query)
	if err != nil {
		responseFunc(kong, 0, err.Error(), nil, nil)
		return
	}

	if resp.Boolean == 0 {
		responseFunc(kong, 0, resp.Message, nil, nil)
		return
	}

	//网关退出
	_, err = h.Redis.Do("HDEL", Config.Token.Name, payload["uid"])
	if err != nil {
		responseFunc(kong, 0, "退出失败，请重试", nil, nil)
		return
	}

	responseFunc(kong, 1, "OK", nil, nil)
}

func (h *Handler) Common(kong *pdk.PDK) {
	jwtToken, err := kong.Request.GetHeader(Config.Token.Name)
	if jwtToken == "" {
		responseFunc(kong, 0, "token无效", nil, nil)
		return
	}

	if err != nil {
		responseFunc(kong, 0, err.Error(), nil, nil)
		return
	}

	payload, err := h.Codec.parseToken(jwtToken)
	if err != nil {
		responseFunc(kong, 0, err.Error(), nil, nil)
		return
	}

	uid := int(payload["uid"].(float64))
	session, _ := h.Redis.Do("HGET", Config.Token.Name, uid)
	if session == nil {
		responseFunc(kong, 0, "尚未登录", nil, nil)
		return
	}

	var data SessionData
	_ = json.Unmarshal(session.([]byte), &data)

	now := time.Now().Unix()
	if now-data.LastVisit > Config.Token.Expire {
		_, _ = h.Redis.Do("HDEL", Config.Token.Name, uid)
		responseFunc(kong, 0, "登录已过期，请重新登录", nil, nil)
		return
	}

	if err := kong.ServiceRequest.SetHeader("uid", strconv.Itoa(uid)); err != nil {
		_ = kong.Log.Err(err)
	}

	data.LastVisit = now
	hashValue, _ := json.Marshal(SessionData{LastVisit: now})
	_, err = h.Redis.Do("HSET", Config.Token.Name, uid, hashValue)
	if err != nil {
		_ = kong.Log.Err(err)
	}
}
