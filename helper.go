package main

import (
    "encoding/json"
    "errors"
    "github.com/Kong/go-pdk"
    "net/http"
    "strings"
)

func CheckToken(authentication string, loginInfo interface{}) (Session, error) {
    session := Session{}
    session.Parse(loginInfo.(string))
    if session.Signature != strings.Split(authentication, ".")[1] {
        return session, errors.New("非法token")
    }

    return session, nil
}

func SetLoginStatus(loginData payload) (*Sign, error) {
    sign := authPlugin.Codec.Encode(loginData)
    refreshLoginInfo := Session{
        LastVisit: loginData["time"].(int64),
        Signature: sign.Signature,
    }

    hashValue, _ := json.Marshal(refreshLoginInfo)
    _, err := authPlugin.Redis.Do("HSET", Config.Token.Name, loginData["uid"], string(hashValue))
    if err != nil {
        return sign, err
    }

    return sign, nil
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
