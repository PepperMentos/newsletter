package main

import (
	"bytes"
	"encoding/csv"
	"flag"
	"io"
	"log"
	"os"
	"strings"
	"text/template"

	"gopkg.in/gomail.v2"

	"github.com/elsonwu/goutil"
)

var host = flag.String("host", "", "Ex. https://www.thecn.com")
var sourceFile = flag.String("source_file", "", "Ex. users.csv, format:id,username,email")
var sentLogFile = flag.String("sent_log_file", "sent.log", "Ex. sent.log")
var tempFile = flag.String("template", "", "Ex. template.html")
var smtpHost = flag.String("smtp_host", "", "Ex. smtp.sendgrid.net")
var smtpUsername = flag.String("smtp_username", "", "")
var smtpPassword = flag.String("smtp_pwd", "", "")
var smtpPort = flag.Int("smtp_port", 587, "")
var maxMailPerConn = flag.Int("max_mail_per_conn", 1000, "Max email in each smtp connection")
var mailFrom = flag.String("mail_from", "help@thecn.com", "")
var mailSubject = flag.String("mail_subject", "", "")
var unsubcribeSalt = flag.String("unsubcribe_salt", "", "The salt for generating CN unsubcribe URL")

func checkFlag() bool {
	if *host == "" {
		log.Println("host is empty")
		return false
	}

	if *sourceFile == "" {
		log.Println("source_file is empty")
		return false
	}

	if *tempFile == "" {
		log.Println("temp_file is missing")
		return false
	}

	if *smtpHost == "" {
		log.Println("smtp_host is missing")
		return false
	}

	if *smtpUsername == "" {
		log.Println("smtp_username is missing")
		return false
	}

	if *smtpPassword == "" {
		log.Println("smtp_pwd is missing")
		return false
	}

	if *mailSubject == "" {
		log.Println("mail_subject is missing")
		return false
	}

	if *mailFrom == "" {
		log.Println("mail_from is missing")
		return false
	}

	if *unsubcribeSalt == "" {
		log.Println("unsubcribe_salt is missing")
		return false
	}
	return true
}

func main() {

	var err error
	flag.Parse()

	if !checkFlag() {
		flag.Usage()
		return
	}

	f, err := os.Open(*sourceFile)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	d := gomail.NewPlainDialer(*smtpHost, *smtpPort, *smtpUsername, *smtpPassword)
	srv, err := d.Dial()
	if err != nil {
		panic(err)
	}

	tmpl, err := template.ParseFiles(*tempFile)
	if err != nil {
		panic(err)
	}

	msg := gomail.NewMessage()

	sentLog, err := os.OpenFile(*sentLogFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		panic(err)
	}
	defer sentLog.Close()

	reader := csv.NewReader(f)
	num := 0
	for {
		num++

		if num%*maxMailPerConn == 0 {
			srv, err = d.Dial()
			if err != nil {
				panic(err)
			}
		}

		line, err := reader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			log.Printf("Err: %s", err.Error())
			break
		}

		if len(line) != 3 {
			log.Printf("Err, incorrect format, line:%s", num)
		}

		// id,name,email
		id := line[0]
		name := line[1]
		address := line[2]

		var doc bytes.Buffer
		data := struct {
			UnsubscribeUrl string
			Host           string
		}{
			UnsubscribeUrl: generateUnsubscribeUrl(id),
			Host:           *host,
		}

		err = tmpl.Execute(&doc, data)
		if err != nil {
			panic(err)
		}

		msg.SetHeader("From", *mailFrom)
		msg.SetAddressHeader("To", address, name)
		msg.SetHeader("Subject", *mailSubject)
		msg.SetBody("text/html", doc.String())

		if err := gomail.Send(srv, msg); err != nil {
			log.Printf("[Err]Could not send email to %q: %v\n", address, err)
			break
		} else {
			log.Printf("[OK]Sent to %v: %s <%v>\n", id, name, address)
			f.WriteString(strings.Join(line, ","))
		}

		msg.Reset()
	}
}

func generateUnsubscribeUrl(userId string) string {
	requestId := goutil.SubStr(goutil.Md5(userId+*unsubcribeSalt), 4, 16)
	return *host + "/site/unsubscribe-news/" + userId + "/" + requestId
}
