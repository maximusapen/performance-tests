/*******************************************************************************
 *
 * OCO Source Materials
 * , 5737-D43
 * (C) Copyright IBM Corp. 2019, 2023 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets, irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package resources

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/IBM-Cloud/ibm-cloud-cli-sdk/bluemix/authentication/iam"
	"github.com/IBM-Cloud/ibm-cloud-cli-sdk/common/rest"
	jwt "github.com/golang-jwt/jwt/v4"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/commands/consts"
)

// Endpoint struct defining config info for APIs
type Endpoint struct {
	AccessToken  string
	BaseURLPath  string
	IAMPath      string
	APIVersion   string
	AccountID    string
	RefreshToken string
	Context      context.Context
	Opts         []EndpointOpt

	restClient *rest.Client
}

// EndpointOpt specifies options for Endpoint
type EndpointOpt func(*Endpoint)

func (endpoint *Endpoint) client() *rest.Client {
	if endpoint.restClient == nil {
		timeout, err := strconv.Atoi(os.Getenv(consts.TimeoutEnvVar))
		if err != nil {
			timeout = 60
		}

		endpoint.restClient = &rest.Client{
			HTTPClient: &http.Client{
				Transport: newTransport(time.Duration(timeout) * time.Second),
			},
		}
	}

	for _, opt := range endpoint.Opts {
		opt(endpoint)
	}
	return endpoint.restClient
}

func newTransport(timeout time.Duration) *http.Transport {
	return &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		Dial: (&net.Dialer{
			Timeout:   timeout,
			KeepAlive: timeout,
		}).Dial,
		TLSHandshakeTimeout:   timeout,
		ResponseHeaderTimeout: timeout,
		ExpectContinueTimeout: timeout,
		MaxIdleConns:          100,
		IdleConnTimeout:       timeout,
	}
}

type claimsExpireVerifier interface {
	jwt.Claims
	VerifyExpiresAt(int64, bool) bool
}

func claimsUpToDate(claims claimsExpireVerifier) error {
	err := claims.Valid()
	if err != nil {
		return err
	}
	const earlyRefreshPeriod = 5 * time.Minute // Recommended check from https://github.ibm.com/ibmcloud-cli/bluemix-cli/issues/3646
	if !claims.VerifyExpiresAt(time.Now().Add(earlyRefreshPeriod).Unix(), true) {
		return fmt.Errorf("Token will expire in less than %s", earlyRefreshPeriod)
	}
	return nil
}

func (endpoint *Endpoint) refreshTokensIfNeeded() error {
	return endpoint.refreshIAMTokenIfNeeded()
}

// Validate the IAM token locally
func (endpoint *Endpoint) refreshIAMTokenIfNeeded() error {
	claims, err := parseJWTToken(endpoint.AccessToken)
	if err != nil {
		fmt.Printf("The IAM token failed to parse: %v\n", err)
		return nil // could not attempt a refresh, so don't indicate an error
	}

	err = claimsUpToDate(claims)
	if err != nil {
		auth := iam.NewClient(iam.DefaultConfig(endpoint.getIAMPath()), rest.NewClient())
		if token, err := auth.GetToken(iam.RefreshTokenRequest(endpoint.RefreshToken)); err == nil {
			endpoint.AccessToken = fmt.Sprintf("%s %s", token.TokenType, token.AccessToken)
			endpoint.RefreshToken = token.RefreshToken
			time.Sleep(1 * time.Second) // Sleep for a momnet to allow token issue time to become valid
		} else {
			fmt.Printf("Failed to refresh IAM token. %v\n", err)
			return errors.New("Access token is expired")
		}
	}

	return nil
}

func (endpoint *Endpoint) getIAMPath() string {
	s := endpoint.IAMPath
	return s
}

func (endpoint *Endpoint) getBaseURL() string {
	s := endpoint.BaseURLPath
	if endpoint.APIVersion != "" {
		if u, err := url.Parse(s); err == nil {
			u.Path = path.Join(u.Path, endpoint.APIVersion)
			s = u.String()
		}
	}
	return s
}
