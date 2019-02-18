package main

import (
    "fmt"
    "net/smtp"
    "os"
    "time"
)

// RepeatType defines how often the job is repeated. Defaults to None
type RepeatType uint

// RepeatType options
const (
    RepeatNone RepeatType = iota
    RepeatDaily
    RepeatWeekly
    RepeatMonthly
)

func (r RepeatType) String() string {
    switch r {
    case RepeatNone:
        return "None"
    case RepeatDaily:
        return "Daily"
    case RepeatWeekly:
        return "Weekly"
    case RepeatMonthly:
        return "Monthly"
    }

    return "?"
}

// MailJob Represents a mail message to be sent at `date`
type MailJob struct {
    Date     time.Time
    Message  string
    FilePath string
    Repeat   RepeatType
}

func (j MailJob) String() string {
    str := "{\n"
    str += "\t At: " + j.Date.String() + "\n"
    str += "\t Msg: " + j.Message + "\n"
    str += "\t File: " + j.FilePath + "\n"
    str += "\t Repeats: " + j.Repeat.String() + "\n"

    str += "}"

    return str
}

// Execute executes the mailjob using sendmail
func (j MailJob) ExecuteWithConfig(config Config) error {
    auth := smtp.PlainAuth(
        "",                          // identity
        config.Server.Username,      // username
        config.Server.Password,      // password
        config.Server.Host,          // host
    )

    msgString := "To: " + config.Email.To + "\r\n"
    msgString += "From: " + config.Email.From + "\r\n"
    msgString += "Subject: [E-Reminder] " + j.Message + "\r\n\r\n"
    msgString += j.Message

    err := smtp.SendMail(
        fmt.Sprintf("%s:%d", config.Server.Host, config.Server.Port),
        auth,
        config.Email.From,
        []string{config.Email.To},
        []byte(msgString),
    )

    return err
}

// SelfDestruct deletes the file representation of the mail job
func (j MailJob) SelfDestruct() {
    os.Remove(j.FilePath)
}

