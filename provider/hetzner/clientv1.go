package main

import (
	//"time"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type Client struct {
	client   *http.Client
	endpoint string
	token    string
}

type ZonesResponse struct {
	Meta  Meta    `json:"meta,omitempty"`
	Zones []*Zone `json:"zones,omitempty"`
}

type Meta struct {
	Pagination Pagination `json:"pagination,omitempty"`
}

type Pagination struct {
	LastPage     int `json:"last_page,omitempty"`
	Page         int `json:"page,omitempty"`
	PerPage      int `json:"per_page,omitempty"`
	TotalEntries int `json:"total_entries,omitempty"`
}

type Zone struct {
	//Created time.Time `json:"created,omitempty"`

	ID             string   `json:"id,omitempty"`
	IsSecondaryDns bool     `json:"is_secondary_dns,omitempty"`
	LegacyDNSHost  string   `json:"legacy_dns_host,omitempty"`
	LegacyNS       []string `json:"legacy_ns,omitempty"`
	//Modified time.Time`json:"modified,omitempty"`
	Name            string          `json:"name,omitempty"`
	NS              []string        `json:"ns,omitempty"`
	Owner           string          `json:"owner,omitempty"`
	Paused          bool            `json:"paused,omitempty"`
	Permission      string          `json:"permission,omitempty"`
	Project         string          `json:"project,omitempty"`
	RecordsCount    uint64          `json:"records_count,omitempty"`
	Registrar       string          `json:"registrar,omitempty"`
	Status          string          `json:"status,omitempty"`
	TTL             uint64          `json:"ttl,omitempty"`
	TXTVerification TXTVerification `json:"txt_verification,omitempty"`
	//Verified time.Time `json:"verified,omitempty"`
}

type TXTVerification struct {
	Name  string `json:"name,omitempty"`
	Token string `json:"token,omitempty"`
}

type ZoneResponse struct {
	Zone Zone `json:"zone,omitempty"`
}

type RecordsResponse struct {
	Records []*Record `json:"records,omitempty"`
}

type Record struct {
	//Created time.Time `json:"created,omitempty"`
	ID string `json:"id,omitempty"`
	//Modified time.Time `json:"modified,omitempty"`
	Name   string `json:"name,omitempty"`
	TTL    uint64 `json:"ttl,omitempty"`
	Type   Type   `json:"type,omitempty"`
	Value  string `json:"value,omitempty"`
	ZoneID string `json:"zone_id,omitempty"`
}

type Type = string

const (
	A               = "A"
	AAAA            = "AAAA"
	PTR             = "PTR"
	NS              = "NS"
	MX              = "MX"
	CNAME           = "CNAME"
	RP              = "RP"
	TXT             = "TXT"
	SOA             = "SOA"
	HINFO           = "HINFO"
	SRV             = "SRV"
	DANE            = "DANE"
	TLSA            = "TLSA"
	DS              = "DS"
	CAA             = "CAA"
	API_ENDPOINT_V1 = "https://dns.hetzner.com/api/v1/"
)

type RecordResponse struct {
	Record Record `json:"record,omitempty"`
}

type BulkRecordsRequest struct {
	Records []*Record `json:"records,omitempty"`
}

type BulkCreateRecordsResponse struct {
	Records        []*Record `json:"records,omitempty"`
	ValidRecords   []*Record `json:"valid_records,omitempty"`
	InvalidRecords []*Record `json:"invalid_records,omitempty"`
}

type BulkUpdateRecordsResponse struct {
	Records       []Record `json:"records,omitempty"`
	FailedRecords []Record `json:"failed_records,omitempty"`
}

func (c *Client) call(method string, target string, requestObject interface{}, responseObject interface{}) error {
	var data []byte
	var err error
	if requestObject != nil {
		data, err = json.Marshal(requestObject)
		if err != nil {
			return err
		}
	}

	url := c.endpoint + target
	req, err := http.NewRequest(method, url, bytes.NewReader(data))
	if err != nil {
		return err
	}

	req.Header.Add("Accept", "application/json")
	req.Header.Add("Auth-API-Token", c.token)

	if requestObject != nil {
		req.Header.Add("Content-Type", "application/json")
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		apiError := &Error{
			resp.StatusCode,
			string(body),
		}

		return apiError
	}

	if len(body) == 0 || responseObject == nil {
		return nil
	}

	return json.Unmarshal(body, responseObject)
}

type Error struct {
	Code    int
	Message string
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s - %s", e.Code, e.Message)
}

func NewClient(token string) *Client {
	return &Client{
		client:   &http.Client{},
		endpoint: API_ENDPOINT_V1,
		token:    token,
	}
}

func (c *Client) GetZones(name string, page int, per_page int, search_name string) (*ZonesResponse, error) {
	var zonesResponse ZonesResponse

	err := c.call("GET", "zones", nil, &zonesResponse)
	if err != nil {
		return nil, err
	}

	return &zonesResponse, nil
}

func main() {
	fmt.Println("staring ....")
	client := NewClient("")

	d, err := client.GetZones()
	if err != nil {
		panic(err)
	}

	fmt.Println(d)
}
