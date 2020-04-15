package main

import (
    "encoding/json"
    "errors"
    "fmt"
    "github.com/Kong/go-pdk"
    "net/http"
    "strconv"
    "strings"
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
    defer func(kong *pdk.PDK) {
        if err := recover(); err != nil {
            _ = kong.Log.Err(fmt.Sprint(err))
        }
    }(kong)

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

    uid, err := ConventToInt64(resp.Data["uid"])
    if err != nil {
        Response(kong, 0, resp.Message, nil, nil)
        return
    }

    loginData := payload{
        "uid":  uid,
        "time": time.Now().Unix(),
    }

    sign := authPlugin.Codec.Encode(loginData)
    refreshLoginInfo := Session{
        LastVisit: loginData["time"].(int64),
        Signature: sign.Signature,
    }

    if err = SetLoginStatus(uid, refreshLoginInfo); err != nil {
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

    sign, err := authPlugin.Codec.Decode(authentication)
    if err != nil {
        Response(kong, 0, err.Error(), nil, nil)
        return
    }

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

    token, err := authPlugin.Codec.Decode(authentication)
    if err != nil {
        Response(kong, 0, err.Error(), nil, nil)
        return
    }

    if _, ok := token.Payload["uid"]; !ok {
        Response(kong, 0, "token无效", nil, nil)
        return
    }

    uid, err := ConventToInt64(token.Payload["uid"])
    if err != nil {
        Response(kong, 0, err.Error(), nil, nil)
        return
    }

    loginInfo, _ := authPlugin.Redis.Do("HGET", Config.Token.Name, uid)
    if loginInfo == nil || loginInfo == 0 {
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
        Response(kong, 0, "尚未登录", nil, nil)
        return
    }

    loginData := payload{
        "uid":  uid,
        "time": now,
    }

    refreshLoginInfo := Session{
        LastVisit: loginData["time"].(int64),
        Signature: session.Signature,
    }

    if err := SetLoginStatus(uid, refreshLoginInfo); err != nil {
        Response(kong, 0, err.Error(), nil, nil)
        return
    }

    if err := kong.ServiceRequest.SetHeader("uid", strconv.Itoa(int(uid))); err != nil {
        _ = kong.Log.Err(err)
    }
}

func CheckToken(authentication string, loginInfo interface{}) (Session, error) {
    session := Session{}
    session.Parse(string(loginInfo.([]byte)))
    if session.Signature != strings.Split(authentication, ".")[1] {
        return session, errors.New("尚未登录")
    }

    return session, nil
}

func SetLoginStatus(uid int64, session Session) error {
    hashValue, _ := json.Marshal(session)
    _, err := authPlugin.Redis.Do("HSET", Config.Token.Name, uid, string(hashValue))
    if err != nil {
        return err
    }

    return nil
}

func Response(kong *pdk.PDK, status int, message string, data map[string]interface{}, header map[string]string) {
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
