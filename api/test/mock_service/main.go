
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"
)

var verbose bool

func returnCode200(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("☄ Dummy Bluemix Authentication returned authorised."))

	if verbose {
		fmt.Println("returnCode200\n\thttp://" + r.Host + r.URL.String())
	}
}

func returnCode201(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("☄ Dummy Bluemix User Policy returned created."))

	if verbose {
		fmt.Println("returnCode201\n\thttp://" + r.Host + r.URL.String())
	}
}

func returnAPIKey(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	responseBody := "{\"metadata\":{\"uuid\":\"ApiKey-cb21db30-d2bb-4156-8271-059e0a3a6c15\",\"crn\":\"crn:v1:staging:public:iam::a/a1b2c3d45f9a123bc456dde012345e67F::apikey:ApiKey-cb21db30-d2bb-4156-8271-059e0a3a6c15\",\"version\":\"1-5bf31abe55c5c0883b0fc58137b4e4ad\",\"createdAt\":\"2017-05-09T14:07+0000\",\"modifiedAt\":\"2017-05-09T14:07+0000\"},\"entity\":{\"boundTo\":\"crn:v1:staging:public:iam::a/a1b2c3d45f9a123bc456dde012345e67F:IBMid:user:110000A0AA\",\"name\":\"ap_test\",\"format\":\"APIKEY\",\"apiKey\":\"Aa11A1AAa1aaaA11AaAAaAA11AA11AAAAa-1AAa1A1aa\"}}" // pragma: allowlist secret
	w.Write([]byte(responseBody))

	if verbose {
		fmt.Println("returnAPIKey\n\thttp://" + r.Host + r.URL.String() + " : " + responseBody)
	}
}

func returnToken(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	responseBody := "{\"Token\":\"eyJraWQiOiIyMDE3MDEwMS0wMDowMDowMCIsImFsZyI6IlJTMjU2In0.eyJyZWFsbWlkIjoiaW50ZXJuYWwiLCJpZGVudGlmaWVyIjoiU2VydmljZUlkLWIzMzU0ODc4LTczMGMtNDQ4OC05NmU2LTdlZDI2ZjQyY2ZlMCIsInN1YiI6IlNlcnZpY2VJZC1iMzM1NDg3OC03MzBjLTQ0ODgtOTZlNi03ZWQyNmY0MmNmZTAiLCJzdWJfdHlwZSI6IlNlcnZpY2VJZCIsImFjY291bnQiOnsiYnNzIjoiYmVmNzFlYTgzZjNkMGMxOGViNzAxMjU2Y2Y2MDM0MWMifSwibWZhIjp7fSwiaWF0IjoxNDg2MzkyMzI3LCJleHAiOjE0ODYzOTU5MjcsImlzcyI6Imh0dHBzOi8vaWFtLnN0YWdlMS5uZy5ibHVlbWl4Lm5ldC9vaWRjL3Rva2VuIiwiZ3JhbnRfdHlwZSI6InVybjppYm06cGFyYW1zOm9hdXRoOmdyYW50LXR5cGU6YXBpa2V5Iiwic2NvcGUiOiJvcGVuaWQiLCJjbGllbnRfaWQiOiJkZWZhdWx0In0.TJPLDSxBN637P6TvmS7QgLSijfz-0ar4XVFledBbcZiPyGEFS-7EEcMCrQqQHncfMBzcgeF7RaDwrZG-F-1waFXMrOLGCY-qZQ5TkrdypLrOND_pccTGFzwRQbve1_ysqnrvE3JfNvfcwcewxXF2hR-vt94XUvtE-_cpTHet6zQj-DBACRyw1nBNcWLCH4kyl9gizS5p9oKpxaO_hz7uFc85Bh18xqpLQcBqOZNUXx2YLv_nkRQDets5dXR1ZAc_ILp8nuSTJWX8SSOSVhYS3QVkacmRfBAnBccRq-pEZyO-JVuo7fpV1jA2E89F1DK9ALomuIJKs2rZ-r5az87mEA\"}" // pragma: allowlist secret
	w.Write([]byte(responseBody))

	if verbose {
		fmt.Println("returnToken\n\thttp://" + r.Host + r.URL.String() + " : " + responseBody)
	}
}

func returnTokenOIDC(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	responseBody := "{\"access_token\":\"eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJpZCI6ImlhbS1TZXJ2aWNlSWQtNjdhY2ViNDUtODI1Mi00YzAzLWE3NTItYTdmYjgxNTk4YTBhIiwiaWFtX2lkIjoiaWFtLVNlcnZpY2VJZC02N2FjZWI0NS04MjUyLTRjMDMtYTc1Mi1hN2ZiODE1OThhMGEiLCJyZWFsbWlkIjoiaWFtIiwiaWRlbnRpZmllciI6IlNlcnZpY2VJZC02N2FjZWI0NS04MjUyLTRjMDMtYTc1Mi1hN2ZiODE1OThhMGEiLCJzdWIiOiJTZXJ2aWNlSWQtNjdhY2ViNDUtODI1Mi00YzAzLWE3NTItYTdmYjgxNTk4YTBhIiwic3ViX3R5cGUiOiJTZXJ2aWNlSWQiLCJhY2NvdW50Ijp7ImJzcyI6ImExYjJjM2Q0NWY5YTEyM2JjNDU2ZGRlMDEyMzQ1ZTY3In0sImlhdCI6MTQ5NTIxMTE2NywiZXhwIjoxNDk1NDY5NTA3LCJpc3MiOiJodHRwczovL2lhbS5zdGFnZTEubmcuYmx1ZW1peC5uZXQvb2lkYy90b2tlbiIsImdyYW50X3R5cGUiOiJ1cm46aWJtOnBhcmFtczpvYXV0aDpncmFudC10eXBlOmFwaWtleSIsInNjb3BlIjoib3BlbmlkIiwiY2xpZW50X2lkIjoiZGVmYXVsdCIsImp0aSI6Ijc3ZTI4NjE5LTVhNGItNDZjZi1iYWVjLWYzMzc5OWQyNTk1MiJ9.Ae9a0cAszp3ZDf-WAdAMpUlNOAmfCXarDvx7cQSpuIY\",\"token_type\":\"Bearer\",\"expires_in\":360000,\"expiration\":9999999999}" // pragma: allowlist secret
	w.Write([]byte(responseBody))

	if verbose {
		fmt.Println("returnToken\n\thttp://" + r.Host + r.URL.String() + " : " + responseBody)
	}
}

func returnIAMPermit(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	responseBody := "{\"Decision\":\"Permit\"}"
	w.Write([]byte(responseBody))

	if verbose {
		fmt.Println("returnIAMPermit\n\thttp://" + r.Host + r.URL.String() + " : " + responseBody)
	}
}

func returnOrg(w http.ResponseWriter, r *http.Request) {
	urlComponents := strings.Split(r.URL.String(), "/")
	orgID := urlComponents[len(urlComponents)-1]
	w.WriteHeader(http.StatusOK)
	responseBody := "{ \"metadata\": {\"guid\": \"a1b2cdef-6a7f-1569-9f29-999999999999\"},\"entity\": {\"organization_guid\":\"" + orgID + "\"}}"
	w.Write([]byte(responseBody))

	if verbose {
		fmt.Println("returnOrg\n\thttp://" + r.Host + r.URL.String() + " : " + responseBody)
	}
}

func returnCoe(w http.ResponseWriter, r *http.Request) {
	urlComponents := strings.Split(r.URL.String(), "/")
	accountID := urlComponents[len(urlComponents)-1]
	w.WriteHeader(http.StatusOK)

	responseBody := "{\"metadata\":{\"guid\":\"" + accountID + "\",\"url\":\"/coe/v2/accounts/" + accountID + "\",\"created_at\":\"2017-02-21T16:14:46.661Z\",\"updated_at\":\"2017-05-22T10:40:40.459Z\"},\"entity\":{\"name\":\"IBM\",\"type\":\"PAYG\",\"state\":\"ACTIVE\",\"owner\":\"111aa11a-a111-11a1-a111-1111111aa1aa\",\"owner_userid\":\"dummy_armadaemail@uk.ibm.com\",\"owner_unique_id\":\"110000A0AA\",\"customer_id\":\"" + accountID + "\",\"country_code\":\"USA\",\"currency_code\":\"USD\",\"billing_country_code\":\"USA\",\"terms_and_conditions\":{\"accepted\":true,\"timestamp\":\"2017-02-21T16:15:08.291Z\"},\"tags\":[],\"team_directory_enabled\":true,\"linkages\":[{\"origin\":\"IMS\",\"state\":\"LINKABLE\"}],\"bluemix_subscriptions\":[{\"type\":\"PAYG\",\"state\":\"ACTIVE\",\"payment_method\":{\"type\":\"TRIAL_CREDIT\",\"started\":\"2017-02-21T16:14:46.660Z\",\"ended\":\"07/15/2018\"},\"subscription_id\":\"AA1A-11AAAA1AAAAAAAA\",\"part_number\":\"COE-Trial-federated\",\"subscriptionTags\":[],\"history\":[]}],\"subscription_id\":\"AA1A-11AAAA1AAAAAAAA\",\"configuration_id\":\"\",\"onboarded\":0}}"
	w.Write([]byte(responseBody))

	if verbose {
		fmt.Println("returnCoe\n\thttp://" + r.Host + r.URL.String() + " : " + responseBody)
	}
}

func returnAccounts(w http.ResponseWriter, r *http.Request) {
	accountID := "a1b2c3d45f9a123bc456dde012345e67" // pragma: allowlist secret
	w.WriteHeader(http.StatusOK)
	responseBody := "{\"total_results\":1,\"total_pages\":1,\"resources\":[{\"metadata\":{\"guid\":\"" + accountID + "\"}}]}"
	w.Write([]byte(responseBody))

	if verbose {
		fmt.Println("returnAccounts\n\thttp://" + r.Host + r.URL.String() + " : " + responseBody)
	}
}

func returnSoftlayer(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	responseBody := "{\"Auth\":\"True\"}"
	w.Write([]byte(responseBody))

	if verbose {
		fmt.Println("returnSoftlayer\n\thttp://" + r.Host + r.URL.String() + " : " + responseBody)
	}
}

func main() {
	const PORT = 80

	flag.BoolVar(&verbose, "verbose", false, "verbose logging output")
	flag.Parse()

	// Use rooted subtree pattern to match all requests
	http.HandleFunc("/", returnCode200)
	http.HandleFunc("/apikeys/", returnAPIKey)
	http.HandleFunc("/apikeys", returnAPIKey)
	http.HandleFunc("/v1/events", returnCode200)    // BSS Billing Event
	http.HandleFunc("/v1/accounts", returnAccounts) // Account verification
	http.HandleFunc("/v2/spaces/", returnOrg)
	http.HandleFunc("/coe/v2/accounts/", returnCoe)
	http.HandleFunc("/v1/authz", returnIAMPermit) // IAM CRN Authorization
	http.HandleFunc("/acms/v1/scopes/", returnCode201)
	http.HandleFunc("/oidc/token", returnTokenOIDC)
	http.HandleFunc("/oauth/token", returnToken) // BSS Token
	http.HandleFunc("/SoftLayer_Account.json", returnSoftlayer)

	addr := fmt.Sprintf("%s%d", ":", PORT)
	fmt.Printf("Mock Bluemix Authentication Service listening on %s\n", addr)

	log.Fatal(http.ListenAndServe(addr, nil))
}
