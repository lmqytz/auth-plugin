package main

import (
    "crypto/md5"
    "encoding/base64"
    "encoding/json"
    "errors"
    "fmt"
    "strings"
)

type payload map[string]interface{}

type Sign struct {
    Payload    payload
    PayloadStr string
    Signature  string
}

func (sign *Sign) toString() string {
    return strings.Join([]string{base64.StdEncoding.EncodeToString([]byte(sign.PayloadStr)), sign.Signature}, ".")
}

type Codec struct {
    SecretKey string
}

func (c *Codec) Encode(payload payload) *Sign {
    bytes, _ := json.Marshal(payload)

    h := md5.New()
    h.Write(bytes)
    h.Write([]byte(c.SecretKey))

    md5str := fmt.Sprintf("%x", h.Sum(nil))
    return &Sign{
        Payload:    payload,
        PayloadStr: string(bytes),
        Signature:  base64.StdEncoding.EncodeToString([]byte(md5str)),
    }
}

func (c *Codec) Decode(token string) (*Sign, error) {
    sign := &Sign{}

    if token == "" {
        return sign, errors.New("token无效")
    }

    signSlice := strings.Split(token, ".")
    bytes, _ := base64.StdEncoding.DecodeString(signSlice[0])

    var payload payload
    if err := json.Unmarshal(bytes, &payload); err != nil {
        return sign, err
    }
    sign.Payload = payload
    sign.PayloadStr = string(bytes)
    sign.Signature = signSlice[1]

    return sign, nil
}