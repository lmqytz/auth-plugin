package main

import (
    "encoding/json"
    "github.com/Kong/go-pdk"
    "strconv"
    "time"
)

type Hook struct{}

func New() interface{} {
    return &Hook{}
}

type AuthPlugin struct {
    Redis      *Redis
    HttpClient *HttpClient
    Codec      *Codec
}

type Session struct {
    LastVisit int64  `json:"last_visit"`
    Signature string `json:"signature"`
}

func (s *Session) Parse(data string) {
    _ = json.Unmarshal([]byte(data), &s)
}

type handlerFunc func(kong *pdk.PDK)

var (
    authPlugin AuthPlugin
    handlers   map[string]handlerFunc
)

func init() {
    InitConfig()

    authPlugin = AuthPlugin{
        Redis:      NewRedis(Config.RedisConf),
        HttpClient: NewHttpClient(Config.HttpConf),
        Codec:      &Codec{SecretKey: Config.Token.Secret},
    }

    handlers = make(map[string]handlerFunc)
    handlers[Config.LoginUrl] = authPlugin.Login
    handlers[Config.LoginOutUrl] = authPlugin.LoginOut
}

func (hook *Hook) Access(kong *pdk.PDK) {
    path, _ := kong.Request.GetPath()

    if f, ok := handlers[path]; ok {
        f(kong)
    } else {
        authPlugin.Common(kong)
    }
}

func (authPlugin *AuthPlugin) Login(kong *pdk.PDK) {
    query, err := kong.Request.GetRawQuery()
    if err != nil {
        Response(kong, 0, err.Error(), nil, nil)
        return
    }

    resp, err := authPlugin.HttpClient.Post(Config.HttpConf.LoginUrl, query)
    if err != nil {
        Response(kong, 0, err.Error(), nil, nil)
        return
    }

    if resp.Boolean == 0 {
        Response(kong, 0, resp.Message, nil, nil)
        return
    }

    loginData := payload{
        "uid":  resp.Data["uid"],
        "time": time.Now().Unix(),
    }

    var sign *Sign
    if sign, err = SetLoginStatus(loginData); err != nil {
        Response(kong, 0, err.Error(), nil, nil)
        return
    }

    header := make(map[string]string)
    header[Config.Token.Name] = sign.toString()
    Response(kong, 1, "登录成功", resp.Data, header)
}

func (authPlugin *AuthPlugin) LoginOut(kong *pdk.PDK) {
    authentication, err := kong.Request.GetHeader(Config.Token.Name)
    if authentication == "" {
        Response(kong, 0, "token无效", nil, nil)
        return
    }

    if err != nil {
        Response(kong, 0, err.Error(), nil, nil)
        return
    }

    sign := authPlugin.Codec.Decode(authentication)
    if _, ok := sign.Payload["uid"]; !ok {
        Response(kong, 0, "token信息不完整", nil, nil)
        return
    }

    loginInfo, err := authPlugin.Redis.Do("HGET", Config.Token.Name, sign.Payload["uid"])
    if err != nil {
        Response(kong, 0, err.Error(), nil, nil)
        return
    }

    if loginInfo == nil || loginInfo == "" {
        Response(kong, 0, "尚未登录", nil, nil)
        return
    }

    if _, err := CheckToken(authentication, loginInfo); err != nil {
        Response(kong, 0, err.Error(), nil, nil)
        return
    }

    query, err := kong.Request.GetRawQuery()
    resp, err := authPlugin.HttpClient.Post(Config.HttpConf.LoginOutUrl, query)
    if err != nil {
        Response(kong, 0, err.Error(), nil, nil)
        return
    }

    if resp.Boolean == 0 {
        Response(kong, 0, resp.Message, nil, nil)
        return
    }

    _, err = authPlugin.Redis.Do("HDEL", Config.Token.Name, sign.Payload["uid"])
    if err != nil {
        Response(kong, 0, "退出失败，请重试", nil, nil)
        return
    }

    Response(kong, 1, "OK", nil, nil)
}

func (authPlugin *AuthPlugin) Common(kong *pdk.PDK) {
    authentication, err := kong.Request.GetHeader(Config.Token.Name)
    if authentication == "" {
        Response(kong, 0, "token无效", nil, nil)
        return
    }

    if err != nil {
        Response(kong, 0, err.Error(), nil, nil)
        return
    }

    token := authPlugin.Codec.Decode(authentication)

    uid := int(token.Payload["uid"].(float64))
    loginInfo, _ := authPlugin.Redis.Do("HGET", Config.Token.Name, uid)
    if loginInfo == nil {
        Response(kong, 0, "尚未登录", nil, nil)
        return
    }

    var session Session
    if session, err = CheckToken(authentication, loginInfo); err != nil {
        Response(kong, 0, err.Error(), nil, nil)
        return
    }

    now := time.Now().Unix()
    if now-session.LastVisit > Config.Token.Expire {
        _, err = authPlugin.Redis.Do("HDEL", Config.Token.Name, uid)
        Response(kong, 0, "登录已过期，请重新登录", nil, nil)
        return
    }

    loginData := payload{
        "uid":  uid,
        "time": now,
    }
    if _, err := SetLoginStatus(loginData); err != nil {
        Response(kong, 0, err.Error(), nil, nil)
        return
    }

    if err := kong.ServiceRequest.SetHeader("uid", strconv.Itoa(uid)); err != nil {
        _ = kong.Log.Err(err)
    }
}
