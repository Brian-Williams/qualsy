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
	SUCCESS   = "SUCCESS"
	emptyBody = xml.Header + `<ServiceRequest>
</ServiceRequest>`
)

//<?xml version="1.0" encoding="UTF-8"?>
//<ServiceResponse
//xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:noNamespaceSchemaLocation="https://qualysapi.qg2.apps.qualys.com/qps/xsd/2.0/am/asset.xsd">
//<responseCode>INVALID_REQUEST</responseCode>
//<responseErrorDetails>
//<errorMessage>Invalid Request</errorMessage>
//<errorResolution>There were no results corresponding to your request. Please check you request or make sure you have permission for the specified action.</errorResolution>
//</responseErrorDetails>
//</ServiceResponse>
type ResponseBase struct {
	ResponseCode    string `xml:"responseCode"`
	ErrorMessage    string `xml:"responseErrorDetails>errorMessage"`
	ErrorResolution string `xml:"responseErrorDetails>errorResolution"`
	Count           int    `xml:"count"`
}

type Tag struct {
	Id   string `xml:"id"`
	Name string `xml:"name"`
}

type TagAdd struct {
	Name string
	Data struct {
		Tag Tag `xml:"Asset"`
	} `xml:"data"`
}

type Criteria struct {
	Field    string `xml:"field,attr"`
	Operator string `xml:"operator,attr"`
	Criteria string `xml:",chardata"`
}

type CriteriaServiceRequest struct {
	XMLName  xml.Name   `xml:"ServiceRequest"`
	Criteria []Criteria `xml:"filters>Criteria"`
}

type TagInfo struct {
	Name  string `xml:"name"`
	Color string `xml:"color"`
}

//<ServiceRequest>
//<data>
//<Tag>
//<name>EES-smp-testtag-blah</name>
//<color>#FFFFFF</color>
//</Tag>
//</data>
//</ServiceRequest>'
type CreateTag struct {
	XMLName xml.Name `xml:"ServiceRequest"`
	Tag     TagInfo  `xml:"data>Tag"`
}

//<ServiceResponse
//xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:noNamespaceSchemaLocation="https://qualysapi.qg2.apps.qualys.com/qps/xsd/2.0/am/tag.xsd">
//<responseCode>SUCCESS</responseCode>
//<count>1</count>
//<data>
//<Tag>
//<id>25697744</id>
//<name>EES-smp-testtag-blah</name>
//<created>2020-03-24T21:38:24Z</created>
//<modified>2020-03-24T21:38:24Z</modified>
//<color>#FFFFFF</color>
//</Tag>
//</data>
//</ServiceResponse>
type TagResponse struct {
	ResponseBase
	Id string `xml:"data>Tag>id"`
}

//<ServiceRequest>
//<filters>
//	<Criteria field="id" operator="EQUALS">`+c.Data.HostAsset.Id+`</Criteria>
// </filters>
// <data>
//   <Asset>
//	 <tags>
//	   <add>
//		 <TagSimple><id>`+d.Data.Tag.Id+`</id></TagSimple>
//	   </add>
//	 </tags>
//   </Asset>
// </data>
//</ServiceRequest>
type UpdateAsset struct {
	XMLName  xml.Name   `xml:"ServiceRequest"`
	Criteria []Criteria `xml:"filters>Criteria"`
	Id       string     `xml:"data>Asset>tags>add>TagSimple>id"`
}

type Qualys struct {
	user     string
	password string
	api      string
}

func xmlString(v interface{}) (string, error) {
	b, err := xml.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal for xmlString() %+v: %w", v, err)
	}
	return xml.Header + string(b), nil
}

func (c CriteriaServiceRequest) String() string {
	s, err := xmlString(c)
	if err != nil {
		log.Error().Err(err).Msg("couldn't get string")
		return ""
	}
	return s
}

func xmlBytes(v interface{}) ([]byte, error) {
	b, err := xml.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal for xmlBytes(): %w", err)
	}
	x := []byte(xml.Header)
	x = append(x, b...)
	return x, nil
}

func New(user, password, api string) Qualys {
	return Qualys{user, password, api}
}

func (q Qualys) baiscAuth() string {
	return base64.StdEncoding.EncodeToString([]byte(q.user + ":" + q.password))
}

func (q Qualys) newRequest(method, url string, body io.Reader) (*http.Request, error) {
	end := q.api + url

	if method == "POST" && body == nil {
		body = bytes.NewBufferString(emptyBody)
	}

	req, err := http.NewRequest(method, end, body)
	if err != nil {
		return nil, fmt.Errorf("failed to make request for url %v: %w", end, err)
	}

	req.Header.Add("X-requested-with", "Qualys-go")
	req.Header.Add("Content-type", "text/xml")
	req.Header.Add("Authorization", "Basic "+q.baiscAuth())
	req.Header.Add("'cache-control", "no-cache")

	return req, nil
}

// do is an opinionated call
// It considers non-200 status codes an error
func (q Qualys) do(r *http.Request) (*http.Response, error) {
	log.Debug().Msgf("request: %+v", r)
	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		return nil, fmt.Errorf("failed to perform request %v: %w", r.RequestURI, err)
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

func (q Qualys) Post(url string, body io.Reader) (*http.Response, error) {
	req, err := q.newRequest("POST", url, body)
	if err != nil {
		return nil, err
	}
	return q.do(req)
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

func checkResponse(base ResponseBase) error {
	if base.ResponseCode != SUCCESS {
		log.Info().Str("Resolution", base.ErrorResolution).Msg(base.ErrorMessage)
		return fmt.Errorf("non-successful response code: %s", base.ResponseCode)
	}
	return nil
}

func checkCount(base ResponseBase, n int) error {
	err := checkResponse(base)
	if err != nil {
		return err
	}
	if base.Count != n {
		return fmt.Errorf("expected exactly %d count, got %d", n, base.Count)
	}
	return nil
}

func checkResponseBody(body io.ReadCloser) (ResponseBase, error) {
	var r ResponseBase
	err := readUnmarshal(body, &r)
	if err != nil {
		return r, err
	}
	return r, checkResponse(r)
}

// checkCountBody reads a response that has a count and success. Ensures the action was successful with a count of exactly n
func checkCountBody(body io.ReadCloser, n int) error {
	r, err := checkResponseBody(body)
	if err != nil {
		return err
	}
	if r.Count != n {
		return fmt.Errorf("expected exactly %d count, got %d", n, r.Count)
	}
	return nil
}

func (q Qualys) deactivateByID(id string) error {
	r, err := q.Post("qps/rest/2.0/deactivateByID/am/asset/"+id+"?=&module=AGENT_VM%2CAGENT_PC", nil)
	if err != nil {
		return fmt.Errorf("failed to deactivateByID %s: %w", id, err)
	}
	err = checkCountBody(r.Body, 1)
	if err != nil {
		return fmt.Errorf("failed to confirm deactivation %s: %w", id, err)
	}
	return nil
}

func (q Qualys) uninstallByID(id string) error {
	r, err := q.Post("qps/rest/2.0/uninstallByID/am/asset/"+id+"?=", nil)
	if err != nil {
		return fmt.Errorf("failed to uninstallByID %s: %w", id, err)
	}
	err = checkCountBody(r.Body, 1)
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

// EqualBody helps create a post body for an equal name operation on a Criteria value
func EqualBody(criteria string, field string) CriteriaServiceRequest {
	return CriteriaServiceRequest{
		XMLName: xml.Name{Local: "ServiceRequest"},
		Criteria: []Criteria{
			{
				Field:    field,
				Operator: "EQUALS",
				Criteria: criteria,
			},
		},
	}
}

// SearchTagExists searches for one tag
func (q Qualys) SearchTagExists(equalsCriteria string) (string, error) {
	serviceRequest := EqualBody(equalsCriteria, "name")
	b, err := xmlBytes(serviceRequest)
	if err != nil {
		return "", err
	}
	r, err := q.Post("qps/rest/2.0/search/am/tag?=", bytes.NewBuffer(b))
	defer r.Body.Close()
	if err != nil {
		return "", err
	}
	var resp TagResponse
	err = readUnmarshal(r.Body, &resp)
	if err != nil {
		return "", err
	}
	err = checkCount(resp.ResponseBase, 1)
	if err != nil {
		log.Debug().Err(err).Msgf("failed count response: %+v", resp)
		return "", err
	}
	return resp.Id, nil
}

// CreateTag creates a single tag
func (q Qualys) CreateTag(tag CreateTag) (string, error) {
	b, err := xmlBytes(tag)
	if err != nil {
		return "", err
	}
	r, err := q.Post("qps/rest/2.0/create/am/tag?=", bytes.NewBuffer(b))
	defer r.Body.Close()
	if err != nil {
		return "", err
	}
	var resp TagResponse
	err = readUnmarshal(r.Body, &resp)
	if err != nil {
		return "", err
	}
	err = checkCount(resp.ResponseBase, 1)
	if err != nil {
		log.Debug().Err(err).Msgf("failed count response: %+v", resp)
		return "", err
	}
	return resp.Id, nil
}

// UpdateAsset updates a single asset
func (q Qualys) UpdateAsset(update UpdateAsset) error {
	b, err := xmlBytes(update)
	if err != nil {
		return err
	}
	r, err := q.Post("qps/rest/2.0/update/am/asset/", bytes.NewBuffer(b))
	defer r.Body.Close()
	if err != nil {
		return err
	}
	return checkCountBody(r.Body, 1)
}

func PostCritChecker(q Qualys, url string, crit interface{}) error {
	b, err := xmlBytes(crit)
	if err != nil {
		return err
	}
	r, err := q.Post(url, bytes.NewBuffer(b))
	defer r.Body.Close()
	if err != nil {
		return err
	}
	return checkCountBody(r.Body, 1)
}

// general helpers

func (q Qualys) IdFromCriteria(ip CriteriaServiceRequest) (id string, err error) {
	b, err := xmlBytes(ip)
	if err != nil {
		return "", err
	}
	r, err := q.Post("qps/rest/2.0/search/am/hostasset/?=", bytes.NewBuffer(b))
	if err != nil {
		return "", err
	}
	type HostID struct {
		ResponseBase
		Id string `xml:"data>HostAsset>id"`
	}
	var hid HostID
	err = readUnmarshal(r.Body, &hid)
	if err != nil {
		return "", err
	}
	return hid.Id, checkCount(hid.ResponseBase, 1)
}
