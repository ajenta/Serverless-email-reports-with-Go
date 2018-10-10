## Serverless email reports with Go

This project implements an AWS Lambda function in Golang, which retrieves values from DynamoDB and then sends an automated email report based on an HTML template.

The workflow is the following:

1. Retrieve data from DynamoDB using the .
2. Email the data from step (1) using AWS SES to a list of recipients.

### Step 1: Retrieve data from DynamoDB

In our DynamoDB table we use a composite key which comprises a partition and a sort key. The partition key is an Organisation (an ID passed as integer) and a Date (passed as string in DynamoDB). In other words, in our DynamoDB table each item is a document for a specific Organisation at a specific date.

First, we create an anonymous Golang struct and assign the Organisation ID and Date values. In this function the orgID value is passed as an environment variable and the date is always yesteday's date:

```
s := struct {
    Organisation int
    Date         string
}{
    orgID,
    time.Now().AddDate(0, 0, -1).Format("2006-01-02"),
}
```

Then we marshal the anonymous struct for DynamoDB, create the request body and make the GetItemRequest:

```
key, err := dynamodbattribute.MarshalMap(s)
if err != nil {
    panic(err.Error())
}

svc := dynamodb.New(cfg)
req := svc.GetItemRequest(&dynamodb.GetItemInput{
    Key:       key,
    TableName: aws.String(tableName),
})

resp, err := req.Send()
if err != nil {
    panic(err.Error())
}
```

Finally, the result is unmarshalled to a Go struct of type `results`:

```
var out results
dynamodbattribute.UnmarshalMap(resp.Item, &out)
```

### Step 2: Send email with the report data

First, we have to define the date and email subject.

```
yesterday := time.Now().AddDate(0, 0, -1).Format("02/01/2006")
subject := fmt.Sprintf("Concurrency report for %s", yesterday)
```

For the email body we use an HTML template. We parse the HTML file:

```
tmpl, err := template.ParseFiles("./email.html")
if err != nil {
    log.Panic(err.Error())
}
```

Then we pass the variables for report values in the template

```
data := map[string]interface{}{
    "yesterday":                   yesterday,
    "Vidyo":                       res.Vidyo,
    "Pexip":                       res.Pexip,
    ...
    "PexipRTMPMinuteLong":         res.PexipRTMPMinuteLong,
}
```

We have to pass the parsed template with the values to a buffer and we read the email body from there as a string.

```
buf := &bytes.Buffer{}
tmpl.Execute(buf, data)

body := buf.String()
```

Finally, we make a SendEmail request using AWS SES, assuming we have created a domain in SES. The email sender and the recipients list are passed as environment variables:

```
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
    panic(err.Error())
}
```
