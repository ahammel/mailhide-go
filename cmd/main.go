package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
)

// Environment Variables
var ClientSecret = os.Getenv("RECAPTCHA_SECRET_KEY")
var EmailAddress = os.Getenv("EMAIL_ADDRESS")

// Request/response data
var ResponseHeaders = map[string]string{
	"Content-Type": "text/html; charset=utf-8",
}

// HTML templates

type SiteVerifyResponse struct {
	Success            bool     `json:"success"`
	ChallengeTimestamp string   `json:"challenge_ts"`
	Hostname           string   `json:"hostname"`
	Score              float32  `json:"score"`
	ErrorCodes         []string `json:"error-codes"`
}

func Respond(body string, status int) events.APIGatewayProxyResponse {
	return events.APIGatewayProxyResponse{
		Body: body, StatusCode: status, Headers: ResponseHeaders}
}

func ErrorResponse(err error) string {
	return fmt.Sprintf(
		"<p>mailhide-go has encountered an error. Sorry ¯\\_(ツ)_/¯</p>"+
			"<p>Details:</p>"+
			"<pre>%s</pre>"+
			"<button onclick=\"history.go(-1);\">Back </button>", err)
}

func HandleRequest(
	ctx context.Context,
	request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	form, err := url.ParseQuery(request.Body)

	if err != nil {
		return Respond(ErrorResponse(err), 400), nil
	}

	formFields, ok := form["g-recaptcha-response"]
	reCaptcha := formFields[0]

	if !ok {
		err = errors.New("key 'g-recaptcha-response' absent from request data")
		return Respond(ErrorResponse(err), 400), nil
	}

	fmt.Printf("ReCaptcha response: %s\n", reCaptcha)

	resp, err := http.Get(
		fmt.Sprintf(
			"https://www.recaptcha.net/recaptcha/api/siteverify?secret=%s&response=%s",
			ClientSecret,
			reCaptcha))

	if err != nil {
		return Respond(ErrorResponse(err), 500), nil
	}

	if resp.StatusCode != 200 {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		bodyString := string(bodyBytes)
		err = errors.New(
			fmt.Sprintf(
				"siteverify responded with %d. Body of repsonse: %s\n",
				resp.StatusCode,
				bodyString))

		return Respond(ErrorResponse(err), 500), nil
	}

	decoder := json.NewDecoder(resp.Body)
	var siteVerify SiteVerifyResponse
	err = decoder.Decode(&siteVerify)

	defer resp.Body.Close()

	if err != nil {
		return Respond(ErrorResponse(err), 500), nil
	}

	fmt.Printf("SiteVerify response: %+v\n", siteVerify)

	if !siteVerify.Success {

		return Respond(
			"<p>reCAPTCHA failed</p> <button onclick=\"history.go(-1);\">Back </button>",
			200), nil
	}

	return Respond(
		fmt.Sprintf(
			"<p><a href=\"mailto:%s\">%s</a></p>",
			EmailAddress,
			EmailAddress),
		200), nil
}

func main() {
	lambda.Start(HandleRequest)
}
