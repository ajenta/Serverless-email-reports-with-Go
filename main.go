package main

import (
	"aws-lambda-go/lambda"
	"bytes"
	"fmt"
	"html/template"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go-v2/service/ses"
)

type results struct {
	Organisation                     int
	Date                             string
	Vidyo                            int
	VidyoTotalParticipants           int
	VidyoMinuteLong                  int
	VidyoParticipantsMinuteLong      int
	Pexip                            int
	PexipTotalParticipants           int
	PexipWebRTC                      int
	PexipH323                        int
	PexipSIP                         int
	PexipS4B                         int
	PexipRTMP                        int
	PexipMinuteLong                  int
	PexipTotalParticipantsMinuteLong int
	PexipWebRTCMinuteLong            int
	PexipH323MinuteLong              int
	PexipSIPMinuteLong               int
	PexipS4BMinuteLong               int
	PexipRTMPMinuteLong              int
}

var (
	cfg        aws.Config
	err        error
	orgID      int
	tableName  string
	sender     string
	recipients []string
)

func init() {
	cfg, err = external.LoadDefaultAWSConfig()
	if err != nil {
		log.Panic("unable to load SDK config, " + err.Error())
	}

	orgID, err = strconv.Atoi(os.Getenv("ORG_ID"))
	if err != nil {
		log.Panic(err.Error())
	}

	tableName = os.Getenv("DYNAMODB_TABLE")
	recipients = strings.Split(os.Getenv("RECIPIENTS"), ",")
	sender = os.Getenv("SENDER")
}

func getConcurrencyNumbers() results {
	s := struct {
		Organisation int
		Date         string
	}{
		orgID,
		time.Now().AddDate(0, 0, -1).Format("2006-01-02"),
	}

	key, err := dynamodbattribute.MarshalMap(s)
	if err != nil {
		log.Panic(err.Error())
	}

	svc := dynamodb.New(cfg)
	req := svc.GetItemRequest(&dynamodb.GetItemInput{
		Key:       key,
		TableName: aws.String(tableName),
	})

	resp, err := req.Send()
	if err != nil {
		log.Panic(err.Error())
	}

	var out results
	dynamodbattribute.UnmarshalMap(resp.Item, &out)

	return out
}

func sendEmail(res results) {
	yesterday := time.Now().AddDate(0, 0, -1).Format("02/01/2006")
	subject := fmt.Sprintf("Vscene report for %s", yesterday)

	// Parse the email.html template
	tmpl, err := template.ParseFiles("./email.html")
	if err != nil {
		log.Panic(err.Error())
	}

	// Pass the variables in the template
	data := map[string]interface{}{
		"yesterday":                   yesterday,
		"Vidyo":                       res.Vidyo,
		"VidyoTotalParticipants":      res.VidyoTotalParticipants,
		"VidyoMinuteLong":             res.VidyoMinuteLong,
		"VidyoParticipantsMinuteLong": res.VidyoParticipantsMinuteLong,
		"Pexip":                            res.Pexip,
		"PexipTotalParticipants":           res.PexipTotalParticipants,
		"PexipWebRTC":                      res.PexipWebRTC,
		"PexipH323":                        res.PexipH323,
		"PexipSIP":                         res.PexipSIP,
		"PexipS4B":                         res.PexipS4B,
		"PexipRTMP":                        res.PexipRTMP,
		"PexipMinuteLong":                  res.PexipMinuteLong,
		"PexipTotalParticipantsMinuteLong": res.PexipTotalParticipantsMinuteLong,
		"PexipWebRTCMinuteLong":            res.PexipWebRTCMinuteLong,
		"PexipH323MinuteLong":              res.PexipH323MinuteLong,
		"PexipSIPMinuteLong":               res.PexipSIPMinuteLong,
		"PexipS4BMinuteLong":               res.PexipS4BMinuteLong,
		"PexipRTMPMinuteLong":              res.PexipRTMPMinuteLong,
	}

	// Write the filled template to a buffer
	buf := &bytes.Buffer{}
	tmpl.Execute(buf, data)

	// Read the email body from the buffer
	body := buf.String()

	// Create and make the request to AWS SES
	svc := ses.New(cfg)
	req := svc.SendEmailRequest(&ses.SendEmailInput{
		Destination: &ses.Destination{
			ToAddresses: recipients,
		},
		Message: &ses.Message{
			Subject: &ses.Content{
				Data: aws.String(subject),
			},
			Body: &ses.Body{
				Html: &ses.Content{
					Data: aws.String(body),
				},
			},
		},
		Source: aws.String(sender),
	})

	_, err = req.Send()
	if err != nil {
		log.Panic(err.Error())
	}
}

func concurrencyEmailReports() {
	results := getConcurrencyNumbers()
	sendEmail(results)
}

func main() {
	lambda.Start(concurrencyEmailReports)
}
