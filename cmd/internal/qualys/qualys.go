package qualys

import (
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"github.com/rs/zerolog/log"
	"io"
	"io/ioutil"
	"net/http"
)

const (
	SUCCESS = "SUCCESS"
)

// Code has the "responseCode" which should be present in all Qualys api returns
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


type Tag struct {
	Id   string `xml:"id"`
	Name string `xml:"name"`
}

type TagAdd struct {
	Data         struct {
		Tag Tag `xml:"Asset"`
	} `xml:"data"`
}

type Qualys struct {
	user     string
	password string
	api      string
}

func New(user, password, api string) Qualys {
	return Qualys{user, password, api}
}

func (q Qualys) baiscAuth() string {
	return base64.StdEncoding.EncodeToString([]byte(q.user + ":" + q.password))
}

// do is an opinionated call
// It handles headers, client, and considers non-200 status codes an error
func (q Qualys) do(method, url string, body io.Reader) (*http.Response, error) {
	end := q.api + url

	req, err := http.NewRequest(method, end, body)
	if err != nil {
		return nil, fmt.Errorf("failed to make request for url %v: %w", end, err)
	}

	req.Header.Add("X-requested-with", "Qualys-go")
	req.Header.Add("Content-type", "text/xml")
	req.Header.Add("Authorization", "Basic "+q.baiscAuth())
	//req.Header.Add("'cache-control", "no-cache")

	log.Debug().Msgf("request: %+v", req)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to perform request %v: %w", req.RequestURI, err)
	}

	if int(resp.StatusCode) < 200 || int(resp.StatusCode) > 299 {
		err := fmt.Errorf("\"%v\" returned a non-200 error code: %s", resp.Request.URL, resp.Status)
		log.Error().
			Err(err).
			Int("StatusCode", resp.StatusCode).
			Msgf("non-200 status code")
		return resp, err
	}

	return resp, nil
}

// post is an opinionated post to Qualys.
// It will try to make an invalid call valid before sending
func (q Qualys) post(url string, body io.Reader) (*http.Response, error) {
	if body == nil {
		log.Info().Msg("adding filler body for no body post call")
		body = bytes.NewBufferString(`<?xml version="1.0" encoding="UTF-8" ?> <ServiceRequest>
</ServiceRequest>`)
	}
	return q.do("POST", url, body)
}

func readUnmarshal(body io.ReadCloser, s interface{}) error {
	b, err := func() ([]byte, error) {
		defer body.Close()
		b, err := ioutil.ReadAll(body)
		return b, err
	}()
	if err != nil {
		return err
	}

	err = xml.Unmarshal(b, s)
	if err != nil {
		return err
	}
	return nil
}

func checkResponse(body io.ReadCloser) (ResponseBase, error) {
	var r ResponseBase
	err := readUnmarshal(body, &r)
	if err != nil {
		return r, err
	}
	if r.ResponseCode != SUCCESS {
		return r, fmt.Errorf("non-successful response code: %s", r.ResponseCode)
	}
	return r, nil
}

// checkCount reads a response that has a count and success. Ensures the action was successful with a count of exactly one
func checkCount(body io.ReadCloser, n int) error {
	r, err := checkResponse(body)
	if err != nil {
		return err
	}
	if r.Count != n {
		return fmt.Errorf("expected exactly %d count, got %d", n, r.Count)
	}
	return nil
}

func (q Qualys) deactivateByID(id string) error {
	r, err := q.post("qps/rest/2.0/deactivateByID/am/asset/"+id+"?=&module=AGENT_VM%2CAGENT_PC", nil)
	if err != nil {
		return fmt.Errorf("failed to deactivateByID %s: %w", id, err)
	}
	err = checkCount(r.Body, 1)
	if err != nil {
		return fmt.Errorf("failed to confirm deactivation %s: %w", id, err)
	}
	return nil
}

func (q Qualys) uninstallByID(id string) error {
	r, err := q.post("qps/rest/2.0/uninstallByID/am/asset/"+id+"?=", nil)
	if err != nil {
		return fmt.Errorf("failed to uninstallByID %s: %w", id, err)
	}
	err = checkCount(r.Body, 1)
	if err != nil {
		return fmt.Errorf("failed to confirm uninstallByID %s: %w", id, err)
	}
	return nil
}

// CleanID will deactivateByID the id confirm it was removed and then uninstallByID the id
func (q Qualys) CleanID(id string) error {
	err := q.deactivateByID(id)
	if err != nil {
		return err
	}
	return q.uninstallByID(id)
}

// TAG BASED ACTIONS

//curl --request POST \
//--url 'https://qualysapi.qg2.apps.qualys.com/qps/rest/2.0/create/am/tag?=' \
//--header 'authorization: Basic ZWFzdGMyYWUxOjhKZW50MzBOZC03' \
//--header 'cache-control: no-cache' \
//--header 'content-type: text/xml' \
//--header 'x-requested-with: Insomnia-Testing' \
//--cookie JSESSIONID=02871775DAB22E7D1E3B16578444A193 \
//--data '<?xml version="1.0" encoding="UTF-8" ?>
//<ServiceRequest>
//<data>
//<Tag>
//<name>EES-smp-testtag-blah</name>
//<color>#FFFFFF</color>
//</Tag>
//</data>
//</ServiceRequest>'
func UpdateAsset() {}
