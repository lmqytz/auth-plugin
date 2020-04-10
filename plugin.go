package main

import (
	"encoding/json"
	"github.com/Kong/go-pdk"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
)

type Hook struct{}

func New() interface{} {
	return &Hook{}
}

type responseModel struct {
	Boolean int    `json:"boolean"`
	Message string `json:"message"`
	Data    struct {
		Uid      int64  `json:"uid"`
		Username string `json:"username"`
	} `json:"data"`
}

type Handler struct {
	Redis      *Redis
	HttpClient *HttpClient
	Codec      *Codec
}

type SessionModel struct {
	LastVisit int64 `json:"last_visit"`
}

var (
	responseFunc func(kong *pdk.PDK, status int, message string, data map[string]interface{})
	handler      Handler
	response     responseModel
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

	responseFunc = func(kong *pdk.PDK, status int, message string, data map[string]interface{}) {
		var response struct {
			Boolean int                    `json:"boolean"`
			Message string                 `json:"message"`
			Data    map[string]interface{} `json:"data"`
		}
		response.Boolean = status
		response.Message = message
		response.Data = data

		header := make(http.Header)
		header.Set("Content-Type", "application/json")
		body, _ := json.Marshal(response)

		kong.Response.Exit(
			200,
			string(body),
			header,
		)
	}
}

func (hook *Hook) Access(kong *pdk.PDK) {
	path, _ := kong.Request.GetPath()
	if path == Config.LoginUrl {
		handler.Login(kong)
	} else if path == Config.LoginOutUrl {
		handler.LoginOut(kong)
	} else {
		handler.Common(kong)
	}
}

func (h *Handler) Login(kong *pdk.PDK) {
	query, _ := kong.Request.GetRawQuery()
	resp, err := h.HttpClient.Post(Config.HttpConf.LoginUrl, query)
	if err != nil {
		_ = kong.Log.Err(err.Error())
	}

	res, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		_ = kong.Log.Err(err.Error())
	}

	err = json.Unmarshal(res, &response)
	if err != nil {
		_ = kong.Log.Err(err.Error())
	}

	if response.Boolean == 1 {
		now := time.Now().Unix()
		payload := map[string]interface{}{
			"uid":  response.Data.Uid,
			"time": now,
		}

		token, _ := h.Codec.createToken(payload)
		hashValue, _ := json.Marshal(SessionModel{LastVisit: now})
		_, err := h.Redis.Do("HSET", Config.Token.Name, response.Data.Uid, string(hashValue))
		if err != nil {
			_ = kong.Log.Err(err.Error())
		}

		resp.Header.Set(Config.Token.Name, token)
	}

	kong.Response.Exit(
		resp.StatusCode,
		string(res),
		resp.Header,
	)
}

func (h *Handler) LoginOut(kong *pdk.PDK) {
	jwtToken, err := kong.Request.GetHeader(Config.Token.Name)
	if jwtToken == "" {
		responseFunc(kong, 0, "token无效", nil)
		return
	}

	if err != nil {
		responseFunc(kong, 0, err.Error(), nil)
		return
	}

	payload, err := h.Codec.parseToken(jwtToken)
	if err != nil {
		responseFunc(kong, 0, err.Error(), nil)
		return
	}

	userData, err := h.Redis.Do("HGET", Config.Token.Name, payload["uid"])
	if err != nil {
		_ = kong.Log.Err(err.Error())
		responseFunc(kong, 0, err.Error(), nil)
		return
	}

	if userData == nil {
		responseFunc(kong, 0, "尚未登录", nil)
		return
	}

	//执行退出操作
	_, err = h.Redis.Do("HDEL", Config.Token.Name, payload["uid"])
	if err != nil {
		_ = kong.Log.Err(err.Error())
		responseFunc(kong, 0, "退出失败，请重试", nil)
		return
	}

	responseFunc(kong, 1, "OK", nil)
}

func (h *Handler) Common(kong *pdk.PDK) {
	jwtToken, err := kong.Request.GetHeader(Config.Token.Name)
	if jwtToken == "" {
		responseFunc(kong, 0, "token无效", nil)
		return
	}

	if err != nil {
		responseFunc(kong, 0, err.Error(), nil)
		return
	}

	payload, err := h.Codec.parseToken(jwtToken)
	if err != nil {
		responseFunc(kong, 0, err.Error(), nil)
		return
	}

	uid := int(payload["uid"].(float64))
	session, _ := h.Redis.Do("HGET", Config.Token.Name, uid)
	if session == nil {
		responseFunc(kong, 0, "尚未登录", nil)
		return
	}

	var data SessionModel
	if err := json.Unmarshal(session.([]byte), &data); err != nil {
		_ = kong.Log.Err(err)
	}

	now := time.Now().Unix()
	if now-data.LastVisit > Config.Token.Expire {
		_, _ = h.Redis.Do("HDEL", Config.Token.Name, uid)
		responseFunc(kong, 0, "登录已过期，请重新登录", nil)
		return
	}

	if err := kong.ServiceRequest.SetHeader("uid", strconv.Itoa(int(uid))); err != nil {
		_ = kong.Log.Err(err)
	}

	data.LastVisit = now
	hashValue, _ := json.Marshal(SessionModel{LastVisit: now})
	_, err = h.Redis.Do("HSET", Config.Token.Name, uid, hashValue)
	if err != nil {
		_ = kong.Log.Err(err)
	}
}
