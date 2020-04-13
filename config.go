package main

import (
    "gopkg.in/yaml.v2"
    "io/ioutil"
    "log"
    "path"
    "runtime"
)

var (
    Config                          Conf
    DefaultDb                       = 0
    DefaultRedisMaxIdle             = 4
    DefaultRedisMaxActive           = 8
    DefaultRedisTimeout             = 6
    DefaultRedisHealthCheckInterval = 0
    DefaultRedisIdleTimeout         = 300
    DefaultHttpTimeout              = 6
    DefaultTokenName                = "x-custom-token"
)

type Conf struct {
    LoginUrl    string    `yaml:"login_url"`
    LoginOutUrl string    `yaml:"login_out_url"`
    Token       Token     `yaml:"token"`
    RedisConf   RedisConf `yaml:"redis"`
    HttpConf    HttpConf  `yaml:"http"`
}

type Token struct {
    Name   string `yaml:"name"`
    Expire int64  `yaml:"expire"`
    Secret string `yaml:"secret"`
}

type RedisConf struct {
    Address             string `yaml:"address"`
    Password            string `yaml:"password"`
    Db                  int    `yaml:"db"`
    Timeout             int    `yaml:"timeout"`
    MaxIdle             int    `yaml:"max_idle"`
    MaxActive           int    `yaml:"max_active"`
    HealthCheckInterval int    `yaml:"health_check_interval"`
    MaxIdleTime         int    `yaml:"max_idle_timeout"`
}

type HttpConf struct {
    LoginUrl    string `yaml:"login_url"`
    LoginOutUrl string `yaml:"login_out_url"`
    Timeout     int    `yaml:"timeout"`
}

func InitConfig() {
    _, filename, _, _ := runtime.Caller(1)
    yamlFile := path.Join(path.Dir(filename), "./conf.yaml")

    fileBytes, err := ioutil.ReadFile(yamlFile)
    if err != nil {
        log.Fatal(err)
    }

    err = yaml.Unmarshal(fileBytes, &Config)
    if err != nil {
        log.Fatal(err)
    }

    Config.Fix()
}

func (config *Conf) Fix() {
    if config.RedisConf.MaxIdle == 0 {
        config.RedisConf.MaxIdle = DefaultRedisMaxIdle
    }

    if config.RedisConf.MaxActive == 0 {
        config.RedisConf.MaxActive = DefaultRedisMaxActive
    }

    if config.RedisConf.Db == 0 {
        config.RedisConf.Db = DefaultDb
    }

    if config.RedisConf.Timeout == 0 {
        config.RedisConf.Timeout = DefaultRedisTimeout
    }

    if config.RedisConf.HealthCheckInterval == 0 {
        config.RedisConf.HealthCheckInterval = DefaultRedisHealthCheckInterval
    }

    if config.RedisConf.MaxIdleTime == 0 {
        config.RedisConf.MaxIdleTime = DefaultRedisIdleTimeout
    }

    if config.HttpConf.Timeout == 0 {
        config.HttpConf.Timeout = DefaultHttpTimeout
    }

    if config.Token.Name == "" {
        config.Token.Name = DefaultTokenName
    }
}
