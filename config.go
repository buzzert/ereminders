package main

import (
    "fmt"
    "github.com/BurntSushi/toml"
)

type Config struct {
    Email  emailInfo `toml:"email"`
    Server smtpInfo  `toml:"smtp"`
}

type emailInfo struct {
    To   string `toml:"to"`
    From string `toml:"from"`
}

type smtpInfo struct {
    Host     string `toml:"host"`
    Port     uint   `toml:"port"`
    Username string `toml:"username"`
    Password string `toml:"password"`
}

func GetDefaultConfigPath() string {
    return "~/.config/ereminders/config.toml"
}

func LoadConfigFromFile(filepath string) Config {
    // TODO: check permissions, should be 0400
    var config Config
    if _, err := toml.DecodeFile(filepath, &config); err != nil {
        fmt.Print("Error decoding config: ")
        fmt.Println(err)
    }

    return config
}

