package hetzner

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"encoding/json"
	"bytes"
	"strings"
	"errors"
	
	"github.com/sirupsen/logrus"

	"sigs.k8s.io/external-dns/endpoint"
	"sigs.k8s.io/external-dns/provider"
	"sigs.k8s.io/external-dns/plan"
)

type HetznerProvider struct {
	provider.BaseProvider
	dryRun bool
	domainFilter endpoint.DomainFilter
	zoneIDFilter provider.ZoneIDFilter
	api *dnsApiV1
}

type dnsApiV1 struct {
	key string
}

type zonesResponseV1 struct {
	Zones []zoneV1 `json:"zones"`
}

type recordsResponseV1 struct {
	Records []recordV1 `json:"records"`
}

type zoneV1 struct {
	ID string `json:"id"`
	Name string `json:"name"`
}

type recordV1 struct {
	Type string `json:"type"`
	ID string `json:"id"`
	Name string `json:"name"`
	Value string `json:"value"`
	Ttl endpoint.TTL `json:"ttl"`
	ZoneId string `json:"zone_id"` 
}

type createRecordsBulkRequestV1 struct {
	Records []*recordV1 `json:"records"`
}

func NewHetznerProvider(domainFilter endpoint.DomainFilter, zoneIDFilter provider.ZoneIDFilter, dryRun bool) (*HetznerProvider, error) {
	//key, ok := os.LookupEnv("DO_TOKEN")
	//if !ok {
	//	return nil, fmt.Errorf("no key found")
	//}

	key := "jEKi10NIJ8eHoZTzw95PazbIYaBbsEtc"

	api := &dnsApiV1{
		key: key,
	}

	p := &HetznerProvider{
		zoneIDFilter: zoneIDFilter,
		domainFilter: domainFilter,
		api: api,
		dryRun: dryRun,
	}

	return p, nil
}

func (api *dnsApiV1) Zones() (zonesResponseV1, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://dns.hetzner.com/api/v1/zones", nil)
	req.Header.Add("Auth-API-Token", api.key)

	resp, err := client.Do(req)
	if err != nil {
		return zonesResponseV1{}, err
	}

	body, _ := ioutil.ReadAll(resp.Body)

	zones := zonesResponseV1{}
    json.Unmarshal([]byte(body), &zones)

	//todo get all pages

	return zones, nil
}

func (api *dnsApiV1) CreateRecords(records []*recordV1) (error) {
	body, err := json.Marshal(createRecordsBulkRequestV1{
		Records: records,
	})
	if err != nil {
		return err
	}

	client := &http.Client{}
	req, err := http.NewRequest("POST", "https://dns.hetzner.com/api/v1/records/bulk", bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Auth-API-Token", api.key)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	if (resp.Status != "200 OK") {
		respBody, _ := ioutil.ReadAll(resp.Body)
		logrus.Error(string(respBody))
	}

	return nil
}

func (api *dnsApiV1) UpdateRecords(records []*recordV1) (error) {
	body, err := json.Marshal(createRecordsBulkRequestV1{
		Records: records,
	})
	if err != nil {
		return err
	}

	client := &http.Client{}
	req, err := http.NewRequest("PUT", "https://dns.hetzner.com/api/v1/records/bulk", bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Auth-API-Token", api.key)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	if (resp.Status != "200 OK") {
		respBody, _ := ioutil.ReadAll(resp.Body)
		logrus.Error(string(respBody))
	}

	return nil
}

func (api *dnsApiV1) DeleteRecord(record *recordV1) (error) {
	client := &http.Client{}
	req, err := http.NewRequest("DELETE", fmt.Sprintf("https://dns.hetzner.com/api/v1/records/%s", record.ID), nil)
	if err != nil {
		return err
	}

	req.Header.Add("Auth-API-Token", api.key)

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Failure : ", err)
		return err
	}

	if resp.Status != "200 OK" {
		respBody, _ := ioutil.ReadAll(resp.Body)
		return errors.New(string(respBody))
	}

	return nil
}

func (api *dnsApiV1) Records(zoneId string) (recordsResponseV1, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", fmt.Sprintf("https://dns.hetzner.com/api/v1/records?zone_id=%s", zoneId), nil)
	req.Header.Add("Auth-API-Token", api.key)

	resp, err := client.Do(req)
	if err != nil {
		return recordsResponseV1{}, err
	}

	body, _ := ioutil.ReadAll(resp.Body)

	records := recordsResponseV1{}
    json.Unmarshal([]byte(body), &records)

	return records, nil
}

func (p *HetznerProvider) zones() ([]zoneV1, error) {
	var zonesResponse, err = p.api.Zones()
	if err != nil {
		return nil, err
	}

	var zones []zoneV1

	for _, zone := range zonesResponse.Zones {
		if (!p.zoneIDFilter.Match(zone.ID)) {
			continue
		}

		if (!p.domainFilter.Match(zone.Name)) {
			continue
		}

		zones = append(zones, zone)
	}

	return zones, nil
}

func (p *HetznerProvider) records() ([]recordV1, error) {
	zones, err := p.zones()
	if err != nil {
		return nil, fmt.Errorf("could not fetch zones: %s", err) 
	}

	var records []recordV1

	for _, zone := range zones {
		logrus.Debugf("fetch records from zone '%s'", zone.Name)

		recordsResponse, err := p.api.Records(zone.ID)
		if err != nil { // todo  !isNotFoundError(err) {
			return nil, fmt.Errorf("could not fetch records from zone '%s': %s", zone.Name, err)
		}

		for _, record := range recordsResponse.Records {
			if provider.SupportedRecordType(record.Type) {
				if record.Name == "@" {
					record.Name = zone.Name
				}
				
				if ! strings.HasSuffix(record.Name, ".") {
					record.Name += "."
				}
				
				records = append(records, record)
			}
		}
	}

	return records, nil
}

func (p *HetznerProvider) Records(ctx context.Context) ([]*endpoint.Endpoint, error) {
	records, err := p.records()
	if err != nil {
		return nil, err
	}

	var endpoints []*endpoint.Endpoint

	for _, record := range records {
		endpoints = append(endpoints, endpoint.NewEndpointWithTTL(record.Name[:len(record.Name)-1], record.Type, record.Ttl, record.Value))
	}

	return endpoints, nil
}

func (p *HetznerProvider) ApplyChanges(ctx context.Context, changes *plan.Changes) error {
	zoneIdFinder := make(provider.ZoneIDName)
	zones, err := p.zones()
	if err != nil {
		return err
	}
	for _, zone := range zones {
		zoneIdFinder.Add(zone.ID, zone.Name)
	}

	currentRecords, err := p.records()
	if err != nil {
		return err
	}

	if err:= p.createRecords(changes.Create, zoneIdFinder); err != nil {
		return err
	}

	if err:= p.updateRecords(changes.UpdateNew, zoneIdFinder, currentRecords); err != nil {
		return err
	}

	if err:= p.deleteRecords(changes.Delete, currentRecords); err != nil {
		return err
	}

	return nil
}

func endpointToRecordV1(endpoint *endpoint.Endpoint) *recordV1 {
	return &recordV1{
		Type: endpoint.RecordType,
		Name: endpoint.DNSName + ".",
		Value: endpoint.Targets[0], //todo handel multiple tags
		Ttl: endpoint.RecordTTL,
	}
}

func findIdForRecord(record *recordV1, currentRecords []recordV1) string {
	for _, r := range currentRecords {
		if (r.Name == record.Name && r.Type == record.Type) {
			return r.ID
		}
	}

	return ""
}

func (p *HetznerProvider) createRecords(endpoints []*endpoint.Endpoint, zoneIdFinder provider.ZoneIDName) error {
	var records []*recordV1
	for _, endpoint := range endpoints {
		zoneId, _ := zoneIdFinder.FindZone(endpoint.DNSName)
		if zoneId == "" {
			logrus.Debugf("Skipping record %s because no hosted zone matching record DNS Name was detected", endpoint.DNSName)
			continue
		}

		record := endpointToRecordV1(endpoint)
		record.ZoneId = zoneId
		records = append(records, record)

		if (p.dryRun) {
			logrus.Info("create: ", record)
		}

		
	}

	if (p.dryRun) {
		return nil
	}

	p.api.CreateRecords(records)//todo handel error

	return nil
}

func (p *HetznerProvider) updateRecords(endpoints []*endpoint.Endpoint, zoneIdFinder provider.ZoneIDName, currentRecords []recordV1) error {
	var records []*recordV1
	for _, endpoint := range endpoints {
		zoneId, _ := zoneIdFinder.FindZone(endpoint.DNSName)
		if zoneId == "" {
			logrus.Debugf("Skipping creating of record %s because no hosted zone matching record DNS Name was detected", endpoint.DNSName)
			continue
		}

		record := endpointToRecordV1(endpoint)
		record.ZoneId = zoneId
		record.ID = findIdForRecord(record, currentRecords)
		if record.ID == "" {
			logrus.Debugf("Skipping update for record %s because its id was not detected", endpoint.DNSName)
			continue
		}

		records = append(records, record)

		if (p.dryRun) {
			logrus.Info("update: ", record)
		}
	}

	if (p.dryRun) {
		return nil
	}

	p.api.UpdateRecords(records) //todo handel error

	return nil
}

func (p *HetznerProvider) deleteRecords(endpoints []*endpoint.Endpoint, currentRecords []recordV1) error {
	var records []*recordV1
	for _, endpoint := range endpoints {
		record := endpointToRecordV1(endpoint)
		record.ID = findIdForRecord(record, currentRecords)

		records = append(records, record)

		if (p.dryRun) {
			logrus.Info("delete: ", record)
		}
	}

	if (p.dryRun) {
		return nil
	}

	for _, record := range records {
		err := p.api.DeleteRecord(record)
		if err != nil {
			logrus.Infof("failed to record %s ", record)
		}
	}

	return nil
}