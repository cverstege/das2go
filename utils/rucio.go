package utils

// DAS RucioAuth module
//
// Copyright (c) 2018 - Valentin Kuznetsov <vkuznet AT gmail dot com>

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"os/exec"
	"strings"
	"time"
)

// RucioValidity
var RucioValidity int64

// RucioTokenCurl
var RucioTokenCurl bool

// RucioAuth represents instance of rucio authentication module
var RucioAuth RucioAuthModule

// RucioAuthModule structure holds all information about Rucio authentication
type RucioAuthModule struct {
	account string
	agent   string
	token   string
	url     string
	ts      int64
}

// String provides string representation of RucioAuthModule
func (r *RucioAuthModule) String() string {
	s := fmt.Sprintf("<RucioAuth account=%s agent=%s url=%s token=%s expire=%v>", r.account, r.agent, r.url, r.token, r.ts)
	return s
}

// Token returns Rucio authentication token
func (r *RucioAuthModule) Token() (string, error) {
	t := time.Now().Unix()
	if r.token != "" && t < r.ts {
		if VERBOSE > 1 {
			log.Println("use cached token", r.String(), "current time", t)
		}
		return r.token, nil
	}
	if VERBOSE > 1 {
		log.Println("get new token", r.String())
	}
	var token string
	var expire int64
	var err error
	if RucioTokenCurl {
		token, expire, err = FetchRucioTokenViaCurl(r.Url())
	} else {
		token, expire, err = FetchRucioToken(r.Url())
	}
	if err != nil {
		return "", err
	}
	r.ts = expire
	r.token = token
	return r.token, nil
}

// Account returns Rucio authentication account
func (r *RucioAuthModule) Account() string {
	if r.account == "" {
		r.account = "das"
		v := GetEnv("RUCIO_ACCOUNT")
		if v != "" {
			r.account = v
		}
	}
	return r.account
}

// Agent returns Rucio authentication agent
func (r *RucioAuthModule) Agent() string {
	if r.agent == "" {
		r.agent = "dasgoserver"
	}
	return r.agent
}

// Url returns Rucio authentication url
func (r *RucioAuthModule) Url() string {
	if r.url == "" {
		v := GetEnv("RUCIO_AUTH_URL")
		if v != "" {
			r.url = fmt.Sprintf("%s/auth/x509", v)
		} else {
			r.url = "https://cms-rucio-auth.cern.ch/auth/x509"
		}
	}
	return r.url
}

// run go-routine to periodically obtain rucio token
// FetchRucioToken request new Rucio token
func FetchRucioToken(rurl string) (string, int64, error) {
	// I need to replace expire with time provided by Rucio auth server
	expire := time.Now().Add(time.Minute * 5).Unix()
	req, _ := http.NewRequest("GET", rurl, nil)
	req.Header.Add("Accept-Encoding", "identity")
	racc := GetEnv("RUCIO_ACCOUNT")
	if WEBSERVER > 0 {
		if racc != "" {
			req.Header.Add("X-Rucio-Account", racc)
		} else {
			req.Header.Add("X-Rucio-Account", RucioAuth.Account())
		}
		req.Header.Add("User-Agent", RucioAuth.Agent())
	} else {
		if racc != "" {
			req.Header.Add("X-Rucio-Account", racc)
		}
		req.Header.Add("User-Agent", "dasgoclient")
	}
	req.Header.Add("Connection", "keep-alive")
	if VERBOSE > 1 {
		dump, err := httputil.DumpRequestOut(req, true)
		log.Printf("http request %+v, rurl %v, dump %v, error %v\n", req, rurl, string(dump), err)
	}
	client := HttpClient()
	resp, err := client.Do(req)
	if err != nil {
		if VERBOSE > 0 {
			log.Println("ERROR: unable to perform request", err)
		}
		return "", 0, err
	}
	defer resp.Body.Close()
	if VERBOSE > 1 {
		dump, err := httputil.DumpResponse(resp, true)
		log.Printf("http response rurl %v, dump %v, error %v\n", rurl, string(dump), err)
	}
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		if VERBOSE > 0 {
			log.Println("ERROR: unable to read response body", err)
		}
		return "", 0, err
	}
	if v, ok := resp.Header["X-Rucio-Auth-Token"]; ok {
		return v[0], expire, nil
	}
	return "", 0, err
}

// FetchRucioTokenViaCurl is a helper function to get Rucio token by using curl command
func FetchRucioTokenViaCurl(rurl string) (string, int64, error) {
	// I need to replace expire with time provided by Rucio auth server
	expire := time.Now().Add(time.Minute * 5).Unix()
	proxy := os.Getenv("X509_USER_PROXY")
	account := GetEnv("RUCIO_ACCOUNT")
	if account == "" {
		account = fmt.Sprintf("X-Rucio-Account: %s", RucioAuth.Account())
	}
	agent := RucioAuth.Agent()
	cmd := fmt.Sprintf("curl -q -I --key %s --cert %s -H \"%s\" -A %s %s", proxy, proxy, account, agent, rurl)
	if WEBSERVER == 0 {
		cmd = fmt.Sprintf("curl -q -I --key %s --cert %s -A %s %s", proxy, proxy, agent, rurl)
	}
	fmt.Println(cmd)
	out, err := exec.Command("sh", "-c", cmd).CombinedOutput()
	if err != nil {
		log.Println("ERROR: unable to execute command", cmd, "error", err)
		return "", 0, err
	}
	var token string
	for _, v := range strings.Split(string(out), "\n") {
		if strings.Contains(strings.ToLower(v), "x-rucio-auth-token:") {
			arr := strings.Split(v, "X-Rucio-Auth-Token: ")
			token = strings.Replace(arr[len(arr)-1], "\n", "", -1)
			token = strings.Replace(token, "\r", "", -1)
			return token, expire, nil
		}
	}
	return token, expire, nil
}
