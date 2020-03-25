package main

import (
	"encoding/base64"
	"encoding/xml"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

func Get_Credential_Hash(User string, Password string) string {

	return base64.StdEncoding.EncodeToString([]byte(User + ":" + Password))
}

func Get_Command_Line_Args() (string, string, string, string, string) {
	/* Get cmd line paramters */
	UserPtr := flag.String("User", "BOGUS", "Qualys Account User Name")
	PasswordPtr := flag.String("Password", "BOGUS", "Qualys Account password")
	APIURLPtr := flag.String("API URL", "https://qualysapi.qg2.apps.qualys.com/", "Qualys API endpoint")
	IPPtr := flag.String("IP", "0.0.0.0", "IP address to search for")
	SDWUUIDPtr := flag.String("SDWUUID", "kweoeimwomc", "ESS SDW UUID")
	flag.Parse()
	return *UserPtr, *PasswordPtr, *APIURLPtr, *IPPtr, *SDWUUIDPtr
}

type Tag_ID struct {
	ResponseCode string `xml:"responseCode"`
	Data         struct {
		Tag struct {
			Id   string `xml:"id"`
			NAME string `xml:"name"`
		} `xml:"Tag"`
	} `xml:"data"`
}

type Tag_Add struct {
	ResponseCode string `xml:"responseCode"`
	Data         struct {
		Tag struct {
			Id   string `xml:"id"`
			NAME string `xml:"name"`
		} `xml:"Asset"`
	} `xml:"data"`
}

type Hostasset_ID struct {
	ResponseCode string `xml:"responseCode"`
	Data         struct {
		HostAsset struct {
			Id string `xml:"id"`
		} `xml:"HostAsset"`
	} `xml:"data"`
}

func Get_Hostasset_id() string {

	User, Password, APIURL, IP, SDWUUID := Get_Command_Line_Args()
	encodedcred := Get_Credential_Hash(User, Password)

	url := APIURL + "qps/rest/2.0/search/am/hostasset/"
	req, _ := http.NewRequest("POST", url, strings.NewReader(`<ServiceRequest>
 <filters>
   <Criteria field="address" operator="EQUALS">`+IP+`</Criteria>
 </filters>
</ServiceRequest>`))
	req.Header.Add("X-requested-with", "GOLANG")
	req.Header.Add("authorization", "Basic "+encodedcred)
	/* req.Header.Add() */
	res, _ := http.DefaultClient.Do(req)
	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)
	//fmt.Println(res)
	//fmt.Println(string(body))
	var c Hostasset_ID

	xml.Unmarshal(body, &c)
	//fmt.Println(c.ResponseCode)

	if c.ResponseCode == "SUCCESS" {
		if c.Data.HostAsset.Id != "" {
			fmt.Println(SDWUUID)
			url := APIURL + "qps/rest/2.0/create/am/tag"
			req, _ := http.NewRequest("POST", url, strings.NewReader(`<ServiceRequest>
                        <data>
                        <Tag>
                        <name>`+SDWUUID+`</name>
                        <color>#FFFFFF</color>
                        </Tag>
                        </data>
                        </ServiceRequest>`))
			req.Header.Add("X-requested-with", "GOLANG")
			req.Header.Add("authorization", "Basic "+encodedcred)
			/* req.Header.Add() */
			res, _ := http.DefaultClient.Do(req)
			defer res.Body.Close()
			body, _ := ioutil.ReadAll(res.Body)
			//fmt.Println(res)
			//fmt.Println(string(body))
			var d Tag_ID

			xml.Unmarshal(body, &d)
			//fmt.Println(c.ResponseCode)

			fmt.Printf(d.Data.Tag.Id)
			if d.Data.Tag.Id != "" {
				fmt.Println(SDWUUID)
				url := APIURL + "qps/rest/2.0/update/am/asset/" + c.Data.HostAsset.Id
				req, _ := http.NewRequest("POST", url, strings.NewReader(`<ServiceRequest>
                                <filters>
                                    <Criteria field="id" operator="EQUALS">`+c.Data.HostAsset.Id+`</Criteria>
                                 </filters>
                                 <data>
                                   <Asset>
                                     <tags>
                                       <add>
                                         <TagSimple><id>`+d.Data.Tag.Id+`</id></TagSimple>
                                       </add>
                                     </tags> 
                                   </Asset>
                                 </data>
                               </ServiceRequest>`))
				req.Header.Add("X-requested-with", "GOLANG")
				req.Header.Add("authorization", "Basic "+encodedcred)
				/* req.Header.Add() */
				res, _ := http.DefaultClient.Do(req)
				defer res.Body.Close()
				body, _ := ioutil.ReadAll(res.Body)
				//fmt.Println(res)
				//fmt.Println(string(body))
				var a Tag_Add

				xml.Unmarshal(body, &a)
				//fmt.Println(c.ResponseCode)

				fmt.Printf(a.ResponseCode)
			}
		}
		return c.Data.HostAsset.Id
	} else {
		return c.ResponseCode
	}
}

func Create_tag() {

}
func main() {
	var numassets string
	numassets = Get_Hostasset_id()
	fmt.Println("HostAsset ID:", numassets)

}
