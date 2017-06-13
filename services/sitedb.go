package services

// DAS service module
// SiteDB module
//
// Copyright (c) 2015-2016 - Valentin Kuznetsov <vkuznet AT gmail dot com>
//

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/dmwm/das2go/dasql"
	"github.com/dmwm/das2go/mongo"
	"github.com/dmwm/das2go/utils"
)

// helper function to load SiteDB data stream
func loadSiteDBData(api string, data []byte) []mongo.DASRecord {
	var out []mongo.DASRecord
	var rec mongo.DASRecord

	// to prevent json.Unmarshal behavior to convert all numbers to float
	// we'll use json decode method with instructions to use numbers as is
	buf := bytes.NewBuffer(data)
	dec := json.NewDecoder(buf)
	dec.UseNumber()
	err := dec.Decode(&rec)

	// original way to decode data
	// err := json.Unmarshal(data, &rec)
	if err != nil {
		msg := fmt.Sprintf("SiteDB unable to unmarshal the data into DAS record, api=%s, data=%s, error=%v", api, string(data), err)
		out = append(out, mongo.DASErrorRecord(msg))
		return out
	}
	desc := rec["desc"].(map[string]interface{})
	headers := desc["columns"].([]interface{})
	values := rec["result"].([]interface{})
	for _, item := range values {
		row := make(mongo.DASRecord)
		val := item.([]interface{})
		for i, h := range headers {
			key := h.(string)
			row[key] = val[i]
			if key == "username" {
				row["name"] = row[key]
			}
		}
		out = append(out, row)
	}
	return out
}

// SiteDBUnmarshal unmarshals SiteDB data stream and return DAS records based on api
func SiteDBUnmarshal(api string, data []byte) []mongo.DASRecord {
	records := loadSiteDBData(api, data)
	return records
}

func getSiteDBData(api string) []mongo.DASRecord {
	furl := fmt.Sprintf("%s/%s", sitedbUrl(), api)
	response := utils.FetchResponse(furl, "")
	if response.Error == nil {
		records := loadSiteDBData(api, response.Data)
		return records
	}
	log.Println(fmt.Sprintf("DAS ERROR, SiteDB API=%s, error=%s", api, response.Error))
	var out []mongo.DASRecord
	return out
}

// L_sitedb2_site_names local API returns site-names
func (LocalAPIs) L_sitedb2_site_names(dasquery dasql.DASQuery) []mongo.DASRecord {
	spec := dasquery.Spec
	var out []mongo.DASRecord
	api := "site-names"
	site := spec["site"].(string)
	sitePattern := ""
	if strings.Contains(site, "*") {
		sitePattern = strings.Replace(site, "*", "", -1)
	}
	records := getSiteDBData(api)
	for _, r := range records {
		siteName := r["site_name"].(string)
		r["name"] = r["site_name"]
		if siteName == site {
			out = append(out, r)
		} else if len(sitePattern) > 0 && strings.Contains(siteName, sitePattern) {
			out = append(out, r)
		}
	}
	return out
}

// L_sitedb2_groups local API returns group names
func (LocalAPIs) L_sitedb2_groups(dasquery dasql.DASQuery) []mongo.DASRecord {
	spec := dasquery.Spec
	var out []mongo.DASRecord
	api := "groups"
	group := spec["group"].(string)
	groupPattern := ""
	if strings.Contains(group, "*") {
		groupPattern = strings.Replace(group, "*", "", -1)
	}
	records := getSiteDBData(api)
	for _, r := range records {
		groupName := r["name"].(string)
		if groupName == group {
			out = append(out, r)
		} else if len(groupPattern) > 0 && strings.Contains(groupName, groupPattern) {
			out = append(out, r)
		}
	}
	return out
}

// L_sitedb2_group_responsibilities return group responsibilities
func (LocalAPIs) L_sitedb2_group_responsibilities(dasquery dasql.DASQuery) []mongo.DASRecord {
	spec := dasquery.Spec
	var out []mongo.DASRecord
	api := "group-responsibilities"
	group := spec["group"].(string)
	groupPattern := ""
	if strings.Contains(group, "*") {
		groupPattern = strings.Replace(group, "*", "", -1)
	}
	records := getSiteDBData(api)
	for _, r := range records {
		val := r["user_name"]
		if val != nil {
			groupName := val.(string)
			r["name"] = r["user_group"]
			if groupName == group {
				out = append(out, r)
			} else if len(groupPattern) > 0 && strings.Contains(groupName, groupPattern) {
				out = append(out, r)
			}
		}
	}
	return out
}

// L_sitedb2_people_via_email returns SiteDB people via email
func (LocalAPIs) L_sitedb2_people_via_email(dasquery dasql.DASQuery) []mongo.DASRecord {
	spec := dasquery.Spec
	var out []mongo.DASRecord
	api := "people"
	user := spec["user"].(string)
	records := getSiteDBData(api)
	for _, r := range records {
		if r["email"].(string) == user {
			out = append(out, r)
		}
	}
	return out
}

// L_sitedb2_people_via_name returns SiteDB people via names
func (LocalAPIs) L_sitedb2_people_via_name(dasquery dasql.DASQuery) []mongo.DASRecord {
	spec := dasquery.Spec
	var out []mongo.DASRecord
	api := "people"
	user := strings.ToLower(spec["user"].(string))
	records := getSiteDBData(api)
	for _, r := range records {
		username := strings.ToLower(r["username"].(string))
		forename := strings.ToLower(r["forename"].(string))
		surname := strings.ToLower(r["surname"].(string))
		email := strings.ToLower(r["email"].(string))
		if username == user || forename == user || surname == user || email == user {
			out = append(out, r)
		}
	}
	return out
}

// L_sitedb2_roles returns SiteDB roles
func (LocalAPIs) L_sitedb2_roles(dasquery dasql.DASQuery) []mongo.DASRecord {
	spec := dasquery.Spec
	var out []mongo.DASRecord
	api := "roles"
	role := spec["role"].(string)
	rolePattern := ""
	if strings.Contains(role, "*") {
		rolePattern = strings.Replace(role, "*", "", -1)
	}
	records := getSiteDBData(api)
	for _, r := range records {
		roleTitle := r["title"].(string)
		if roleTitle == role {
			out = append(out, r)
		} else if len(rolePattern) > 0 && strings.Contains(roleTitle, rolePattern) {
			out = append(out, r)
		}
	}
	return out
}
