package main

import (
    "testing"
)

func TestConfig(t *testing.T) {
    config := LoadConfigFromFile("sample_config.toml")

    if config.Email.From != "E-Reminders <ereminders@example.com>" {
        t.Log("config.Email.From")
        t.Fail()
    }

    if config.Email.To != "Dade Murphy <dademurphy@example.com>" {
        t.Log("config.Email.To")
        t.Fail()
    }

    if config.Server.Host != "example.com" {
        t.Log("config.Server.Host")
        t.Fail()
    }

    if config.Server.Port != 25 {
        t.Log("config.Server.Port")
        t.Fail()
    }

    if config.Server.Username != "dademurphy" {
        t.Log("config.Server.Username")
        t.Fail()
    }

    if config.Server.Password != "iwas0cool" {
        t.Log("config.Server.Password")
        t.Fail()
    }

    t.Log(config)
}

