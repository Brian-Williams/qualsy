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

// Code has the "responseCode" which should be present in all Qualys api returns
type Code struct {
	ResponseCode string `xml:"responseCode"`
}

type ResponseBase struct {
	Code
	Count int `xml:"count"`
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
	Tags    TagInfo  `xml:"data>Tag"`
}

// TODO:
type CreateTagResponse struct {
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
	Id       string     `xml:"data>Asset>tags>add>TagSimple>id,chardata"`
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

func xmlBytes(v interface{}) ([]byte, error) {
	b, err := xml.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal for xmlBytes() %+v: %w", v, err)
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
	//req.Header.Add("'cache-control", "no-cache")

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

func checkResponse(body io.ReadCloser) (ResponseBase, error) {
	defer body.Close()
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
	r, err := q.Post("qps/rest/2.0/deactivateByID/am/asset/"+id+"?=&module=AGENT_VM%2CAGENT_PC", nil)
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
	r, err := q.Post("qps/rest/2.0/uninstallByID/am/asset/"+id+"?=", nil)
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

func (c CriteriaServiceRequest) String() string {
	s, err := xmlString(c)
	if err != nil {
		log.Error().Err(err).Msg("couldn't get string")
		return ""
	}
	return s
}

// equalBody helps create a post body for an equal operation on a Criteria value
func equalBody(criteria string) CriteriaServiceRequest {
	return CriteriaServiceRequest{
		XMLName: xml.Name{Local: "ServiceRequest"},
		Criteria: []Criteria{
			{
				Field:    "name",
				Operator: "EQUALS",
				Criteria: criteria,
			},
		},
	}
}

func (q Qualys) CreateTag(tag CreateTag) (interface{}, error) {
	b, err := xmlBytes(tag)
	if err != nil {
		return nil, err
	}
	r, err := q.Post("qps/rest/2.0/create/am/tag?=", bytes.NewBuffer(b))
	defer r.Body.Close()
	if err != nil {
		return nil, err
	}
	// TODO: get response body to get tag from
	return nil, nil
}

func (q Qualys) UpdateOneAsset(update UpdateAsset) (error) {
	b, err := xmlBytes(update)
	if err != nil {
		return err
	}
	// TODO: is the filter enough for this to work or does `HostAsset.Id` need to be a param?
	r, err := q.Post("qps/rest/2.0/update/am/asset/", bytes.NewBuffer(b))
	defer r.Body.Close()
	if err != nil {
		return err
	}
	return checkCount(r.Body, 1)
}
