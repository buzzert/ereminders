package main

import (
	"net/smtp"
	"os"
	"time"
)

// MailJob Represents a mail message to be sent at `date`
type MailJob struct {
	Date     time.Time
	Message  string
	FilePath string
}

func (j MailJob) String() string {
	str := "{\n"
	str += "\t At: " + j.Date.String() + "\n"
	str += "\t Msg: " + j.Message + "\n"
	str += "\t File: " + j.FilePath + "\n"

	str += "}"

	return str
}

// Execute executes the mailjob using sendmail
func (j MailJob) Execute() error {
	auth := smtp.PlainAuth(
		"",             // identity
		"buzzert",      // username
		"***REMOVED***", // password TODO: gpg this or something
		"localhost",
	)

	msgString := "To: buzzert@***REMOVED***\r\n"
	msgString += "From: E-Reminders<ereminders@***REMOVED***>\r\n"
	msgString += "Subject: [E-Reminder] " + j.Message + "\r\n\r\n"
	msgString += j.Message

	err := smtp.SendMail(
		"localhost:25",
		auth,
		"E-Reminders <ereminders@***REMOVED***>",
		[]string{"buzzert@***REMOVED***"},
		[]byte(msgString),
	)

	return err
}

// SelfDestruct deletes the file representation of the mail job
func (j MailJob) SelfDestruct() {
	os.Remove(j.FilePath)
}
