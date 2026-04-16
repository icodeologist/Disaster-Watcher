package models

type EmailBody struct {
	Title      string
	Location   string
	Precaution string
}

type EmailModel struct {
	Email     string
	EmailBody EmailBody
}
