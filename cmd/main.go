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
	"Content-Type": "application/json",
}

type SiteVerifyResponse struct {
	Success            bool     `json:"success"`
	ChallengeTimestamp string   `json:"challenge_ts"`
	Hostname           string   `json:"hostname"`
	Score              float32  `json:"score"`
	ErrorCodes         []string `json:"error-codes"`
}

type SecretHideResponse struct {
	Success    bool     `json:"success"`
	Secret     *string  `json:"secret"`
}

type SecretHideErrorResponse struct {
	Status string `json:"status"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
}

func RespondSuccess(secret string) events.APIGatewayProxyResponse {
	resp := SecretHideResponse{Success:true, Secret: &secret}
	body, err := json.Marshal(resp)

	if err != nil {
		fmt.Printf("Error serializing success response: %+v. Details: %v \n", resp, err)
	}

	return events.APIGatewayProxyResponse{
		Body: string(body), StatusCode: 200, Headers: ResponseHeaders}
}

func RespondFailure() events.APIGatewayProxyResponse {
	resp := SecretHideResponse{Success:false, Secret: nil}
	body, err := json.Marshal(resp)

	if err != nil {
		fmt.Printf("Error serializing failure response: %+v. Details: %v \n", resp, err)
	}

	return events.APIGatewayProxyResponse{
		Body: string(body), StatusCode: 200, Headers: ResponseHeaders}
}

func RespondError(status int, title string, theError error) events.APIGatewayProxyResponse {
	resp := SecretHideErrorResponse{
		Status: string(status),
		Title: title,
		Detail: fmt.Sprintf("%+v", theError)}

	body, err := json.Marshal(resp)

	if err != nil {
		fmt.Printf("Error serializing error response: %+v. Details: %v \n", resp, err)
	}

	return events.APIGatewayProxyResponse{
		Body: string(body), StatusCode: status, Headers: ResponseHeaders}

}

func HandleRequest(
	ctx context.Context,
	request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	form, err := url.ParseQuery(request.Body)

	if err != nil {
		return RespondError(400, "BadRequest", err), nil
	}

	formFields, ok := form["g-recaptcha-response"]
	reCaptcha := formFields[0]

	if !ok {
		err = errors.New("key 'g-recaptcha-response' absent from request data")
		return RespondError(400, "BadRequest", err), nil
	}

	fmt.Printf("ReCaptcha response: %s\n", reCaptcha)

	resp, err := http.Get(
		fmt.Sprintf(
			"https://www.recaptcha.net/recaptcha/api/siteverify?secret=%s&response=%s",
			ClientSecret,
			reCaptcha))

	if err != nil {
		return RespondError(500, "HttpRequestError", err), nil
	}

	if resp.StatusCode != 200 {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		bodyString := string(bodyBytes)
		err = errors.New(
			fmt.Sprintf(
				"siteverify responded with %d. Body of repsonse: %s\n",
				resp.StatusCode,
				bodyString))

		return RespondError(500, "SiteverifyError", err), nil
	}

	decoder := json.NewDecoder(resp.Body)
	var siteVerify SiteVerifyResponse
	err = decoder.Decode(&siteVerify)

	defer resp.Body.Close()

	if err != nil {
		return RespondError(500, "DeserializationError", err), nil
	}

	fmt.Printf("SiteVerify response: %+v\n", siteVerify)

	if !siteVerify.Success {
		return RespondFailure(), nil
	}

	return RespondSuccess(EmailAddress), nil
}

func main() {
	lambda.Start(HandleRequest)
}
