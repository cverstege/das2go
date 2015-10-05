/*
 *
 * Author     : Valentin Kuznetsov <vkuznet AT gmail dot com>
 * Description: DAS web server, it handles all DAS reuqests
 * Created    : Fri Jun 26 14:25:01 EDT 2015
 *
 * Some links: http://www.alexedwards.net/blog/golang-response-snippets
 * http://blog.golang.org/json-and-go
 * http://golang.org/pkg/html/template/
 * https://labix.org/mgo
 */
package utils

import (
	"bytes"
	"crypto/tls"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"time"
	"x509proxy"
)

/*
 * Return array of certificates
 */
func Certs() (tls_certs []tls.Certificate) {
	uproxy := os.Getenv("X509_USER_PROXY")
	uckey := os.Getenv("X509_USER_KEY")
	ucert := os.Getenv("X509_USER_CERT")
	log.Println("X509_USER_PROXY", uproxy)
	log.Println("X509_USER_KEY", uckey)
	log.Println("X509_USER_CERT", ucert)
	if len(uproxy) > 0 {
		// use local implementation of LoadX409KeyPair instead of tls one
		x509cert, err := x509proxy.LoadX509Proxy(uproxy)
		if err != nil {
			log.Println("Fail to parser proxy X509 certificate", err)
			return
		}
		tls_certs = []tls.Certificate{x509cert}
	} else if len(uckey) > 0 {
		x509cert, err := tls.LoadX509KeyPair(ucert, uckey)
		if err != nil {
			log.Println("Fail to parser user X509 certificate", err)
			return
		}
		tls_certs = []tls.Certificate{x509cert}
	} else {
		return
	}
	return
}

/*
 * HTTP client for urlfetch server
 */
func HttpClient() (client *http.Client) {
	// create HTTP client
	certs := Certs()
	log.Println("Number of certificates", len(certs))
	if len(certs) == 0 {
		client = &http.Client{}
		return
	}
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{Certificates: certs,
			InsecureSkipVerify: true},
	}
	log.Println("Create TLSClientConfig")
	client = &http.Client{Transport: tr}
	return
}

// create global HTTP client and re-use it through the code
var client = HttpClient()

// ResponseType structure is what we expect to get for our URL call.
// It contains a request URL, the data chunk and possible error from remote
type ResponseType struct {
	Url   string
	Data  []byte
	Error error
}

// A URL fetch Worker. It has three channels: in channel for incoming requests
// (in a form of URL strings), out channel for outgoing responses in a form of
// ResponseType structure and quit channel
func Worker(in <-chan string, out chan<- ResponseType, quit <-chan bool) {
	for {
		select {
		case url := <-in:
			//            log.Println("Receive", url)
			go Fetch(url, out)
		case <-quit:
			//            log.Println("Quit Worker")
			return
		default:
			time.Sleep(time.Duration(100) * time.Millisecond)
			//            log.Println("Waiting for request")
		}
	}
}

// Fetch data for provided URL
func FetchResponse(url string) ResponseType {
	var response ResponseType
	response.Url = url
	response.Data = []byte{}
	if validate_url(url) == false {
		response.Error = errors.New("Invalid URL")
		return response
	}
	//     resp, err := client.Get(url)
	req, err := http.NewRequest("GET", url, nil)
	req.Header.Add("Accept-Encoding", "identity")
	resp, err := client.Do(req)
	if err != nil {
		response.Error = err
		return response
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		response.Error = err
		return response
	}
	response.Data = body
	return response
}

// Fetch data for provided URL and redirect results to given channel
func Fetch(url string, ch chan<- ResponseType) {
	//    log.Println("Receive", url)
	var resp, r ResponseType
	retry := 3 // how many times we'll retry given url
	startTime := time.Now()
	resp = FetchResponse(url)
	if resp.Error != nil {
		log.Println("DAS WARNING, fail to fetch data", url, "error", resp.Error)
		for i := 1; i <= retry; i++ {
			sleep := time.Duration(i) * time.Second
			time.Sleep(sleep)
			r = FetchResponse(url)
			if r.Error == nil {
				break
			}
			log.Println("DAS WARNING", url, "retry", i, "error", r.Error)
		}
		resp = r
	}
	if resp.Error != nil {
		log.Println("DAS ERROR, fail to fetch data", url, "retries", retry, "error", resp.Error)
	}
	endTime := time.Now()
	if VERBOSE {
		log.Println("DAS fetch", url, endTime.Sub(startTime))
	}
	ch <- resp
}

// Helper function which validates given URL
func validate_url(url string) bool {
	if len(url) > 0 {
		pat := "(https|http)://[-A-Za-z0-9_+&@#/%?=~_|!:,.;]*[-A-Za-z0-9+&@#/%=~_|]"
		matched, err := regexp.MatchString(pat, url)
		if err == nil {
			if matched == true {
				return true
			}
		}
		log.Println("ERROR invalid URL:", url)
	}
	return false
}

// represent final response in a form of JSON structure
// we use custorm representation
func Response(url string, data []byte) []byte {
	b := []byte(`{"url":`)
	u := []byte(url)
	c := []byte(",")
	d := []byte(`"data":`)
	e := []byte(`}`)
	a := [][]byte{b, u, c, d, data, e}
	s := []byte(" ")
	r := bytes.Join(a, s)
	return r

}
