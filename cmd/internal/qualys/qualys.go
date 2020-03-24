package qualys

import (
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
)

type Code struct {
	ResponseCode string `xml:"responseCode"`
}

type ResponseBase struct {
	Code
	Count int `xml:"count"`
}

type Hostasset_Delete struct {
	ResponseBase
	Data struct {
		HostAsset struct {
			Id string `xml:"id"`
		} `xml:"HostAsset"`
	} `xml:"data"`
}

type qualys struct {
	user     string
	password string
	api      string
}

func New(user, password, api string) qualys {
	return qualys{user, password, api}
}

func (q qualys) baiscAuth() string {
	return base64.StdEncoding.EncodeToString([]byte(q.user + ":" + q.password))
}

func (q qualys) do(method, url string, body io.Reader) (*http.Response, error) {
	end := q.api + url

	req, err := http.NewRequest(method, end, body)
	if err != nil {
		return nil, fmt.Errorf("failed to make request for url %v: %w", end, err)
	}

	req.Header.Add("X-requested-with", "qualys-go")
	req.Header.Add("Content-type", "text/xml")
	req.Header.Add("Authorization", "Basic "+q.baiscAuth())
	//req.Header.Add("'cache-control", "no-cache")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to perform request %v: %w", req.RequestURI, err)
	}

	if int(resp.StatusCode) < 200 || int(resp.StatusCode) > 299 {
		return resp, fmt.Errorf("\"%v\" returned a non-200 error code: %d(%s)", resp.Request.URL, resp.StatusCode, resp.Status)
	}

	return resp, nil
}

func (q qualys) post(url string, body io.Reader) (*http.Response, error) {
	return q.do("POST", url, body)
}

func readUnmarshal(body io.ReadCloser, s interface{}) error {
	b, err := ioutil.ReadAll(body)
	if err != nil {
		return err
	}
	err = xml.Unmarshal(b, s)
	if err != nil {
		return err
	}
	return nil
}

// reads a response that has a count and success. Ensures the action was successful with a count of exactly one
func checkOne(body io.ReadCloser) error {
	var r ResponseBase
	err := readUnmarshal(body, &r)
	if err != nil {
		return err
	}
	if r.ResponseCode != "SUCCESS" {
		return fmt.Errorf("non-successful response code: %s", r.ResponseCode)
	}
	if r.Count != 1 {
		return fmt.Errorf("expected exactly 1 count, got %d", r.Count)
	}
	return nil
}

func (q qualys) deactivate(id string) error {
	r, err := q.post("qps/rest/2.0/deactivate/am/asset/"+id+"?=&module=AGENT_VM%2CAGENT_PC", nil)
	if err != nil {
		return fmt.Errorf("failed to deactivate %s: %w", id, err)
	}
	err = checkOne(r.Body)
	if err != nil {
		return fmt.Errorf("failed to confirm deactivation %s: %w", id, err)
	}
	return nil
}

func (q qualys) uninstall(id string) error {
	r, err := q.post("qps/rest/2.0/uninstall/am/asset/"+id+"?=", nil)
	if err != nil {
		return fmt.Errorf("failed to uninstall %s: %w", id, err)
	}
	err = checkOne(r.Body)
	if err != nil {
		return fmt.Errorf("failed to confirm uninstall %s: %w", id, err)
	}
	return nil
}

// CleanID will deactivate the id confirm it was removed and then uninstall the id
func (q qualys) CleanID(id string) error {
	err := q.deactivate(id)
	if err != nil {
		return err
	}
	return q.uninstall(id)
}
