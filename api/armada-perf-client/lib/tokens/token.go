/*******************************************************************************
 * IBM Confidential
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2017, 2020 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets,  * irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package token

import (
	"bytes"
	"encoding/base32"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	config "github.ibm.com/alchemy-containers/armada-performance/api/armada-perf-client/lib/config"
)

var client *http.Client

// executeRequest returns unmarshalled data from the request
func executeRequest(req *http.Request) (map[string]interface{}, error) {
	var bmResp map[string]interface{}
	var err error

	for i := 1; i <= 5; i++ {
		resp, err := client.Do(req)
		if err == nil {
			defer resp.Body.Close()
			body, err := ioutil.ReadAll(resp.Body)
			if err == nil {
				err = json.Unmarshal(body, &bmResp)
				if err == nil && bmResp != nil {
					break
				} else {
					fmt.Printf("Error occurred unmarshalling JSON. Body was: %s", body)
				}
			} else {
				fmt.Println("Error occurred reading body")
			}
		} else {
			fmt.Printf("Error: Http request to get token failed - %s\n", err.Error())
		}

		fmt.Printf("Warning: Failure while trying to get token. Attempt %d of 5\n", i)
		time.Sleep(time.Second * 5)
	}
	return bmResp, err
}

// GetTokens returns the Bluemix UAA and IAM access/refresh tokens for a Bluemix account
func GetTokens(bxConfig *config.BluemixConfig, verbose bool) (string, string, string) {
	var iamToken, refreshToken, uaaToken string
	var escapedUsername, escapedPassword, escapedAPIKey string

	tokenRequestURL := strings.Join([]string{bxConfig.IAMURL, "identity", "token"}, "/")

	escapedUsername = url.QueryEscape(bxConfig.Username)

	decodedPassword, err := base32.StdEncoding.DecodeString(bxConfig.Password)

	if err != nil {
		// Most likely an invalid base32 encoding, we'll fallback to trying an api key instead
		escapedPassword = ""
	} else {
		escapedPassword = url.QueryEscape(string(decodedPassword))
	}

	escapedAPIKey = url.QueryEscape(bxConfig.APIKey)

	escapedResponseType := url.QueryEscape("cloud_iam,uaa")
	escapedAPIGrantType := url.QueryEscape("urn:ibm:params:oauth:grant-type:apikey")

	var reqBody string

	// If a base32 encoded password has been defined, then use it to authenticate
	if len(escapedPassword) > 0 {
		// Use password to authenticate
		if verbose {
			fmt.Println("password")
		}
		reqBody = "grant_type=password&password=" + escapedPassword + "&response_type=" + escapedResponseType + "&uaa_client_id=cf&uaa_client_secret=&username=" + escapedUsername // pragma: allowlist secret
	} else {
		if verbose {
			fmt.Println("api key")
		}
		// otherwise, use a Bluemix API key to authenticate
		reqBody = "apikey=" + escapedAPIKey + "&grant_type=" + escapedAPIGrantType + "&response_type=" + escapedResponseType + "&uaa_client_id=cf&uaa_client_secret=" // pragma: allowlist secret
	}

	req, err := http.NewRequest("POST", tokenRequestURL, bytes.NewBufferString(reqBody))
	if err != nil {
		panic(err)
	}

	authCreds := base64.StdEncoding.EncodeToString([]byte("bx:bx"))
	req.Header.Set("Authorization", "Basic "+authCreds)
	req.Header.Set("accept", "application/json;charset=utf-8")
	req.Header.Set("content-type", "application/x-www-form-urlencoded;charset=utf-8")

	client = &http.Client{}
	bmResp, err := executeRequest(req)

	if err != nil {
		fmt.Println("ERROR: Couldn't get refresh_token")
		panic(err)
	}

	if _, ok := bmResp["refresh_token"]; ok {
		refreshToken = bmResp["refresh_token"].(string)
	} else {
		time.Sleep(time.Second * 10)
		bmResp, err = executeRequest(req)

		if err != nil {
			refreshToken = bmResp["refresh_token"].(string)
		}
	}

	reqBody = "bss_account=" + bxConfig.AccountID + "&grant_type=refresh_token&ims_account=&refresh_token=" + refreshToken + "&response_type=" + escapedResponseType + "&uaa_client_id=cf&uaa_client_secret=" // pragma: allowlist secret

	req, err = http.NewRequest("POST", tokenRequestURL, bytes.NewBufferString(reqBody))
	if err != nil {
		panic(err)
	}
	req.Header.Set("Authorization", "Basic "+authCreds)
	req.Header.Set("accept", "application/json;charset=utf-8")
	req.Header.Set("content-type", "application/x-www-form-urlencoded;charset=utf-8")

	bmResp, err = executeRequest(req)
	if err != nil {
		fmt.Println("ERROR: Couldn't get access_token, refresh_token and uaa_token")
		panic(err)
	}

	if _, ok := bmResp["access_token"]; ok {
		iamToken = bmResp["access_token"].(string)
		refreshToken = bmResp["refresh_token"].(string)
		uaaToken = bmResp["uaa_token"].(string)
	} else {
		time.Sleep(time.Second * 10)
		bmResp, err = executeRequest(req)
		if err != nil {
			iamToken = bmResp["access_token"].(string)
			refreshToken = bmResp["refresh_token"].(string)
			uaaToken = bmResp["uaa_token"].(string)
		}
	}

	return iamToken, refreshToken, uaaToken
}
