package main

import (
	"bufio"
	"crypto/tls"
	"flag"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

type paramCheck struct {
	url   string
	param string
}

var httpClient *http.Client

// Custom flag type to allow multiple headers
type headersFlag []string

func (h *headersFlag) String() string {
	return strings.Join(*h, ", ")
}

func (h *headersFlag) Set(value string) error {
	*h = append(*h, value)
	return nil
}

func main() {
	var headers headersFlag
	var proxyURL string

	flag.Var(&headers, "header", "Custom headers for the HTTP request in the format 'Header: Value'")
	flag.StringVar(&proxyURL, "proxy", "", "Custom HTTP proxy URL")

	flag.Parse()

	// Parse headers from the flags
	parsedHeaders := parseHeaders(headers)

	// Configure the HTTP client
	httpClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: time.Second,
				DualStack: true,
			}).DialContext,
		},
	}

	// Set the proxy if provided
	if proxyURL != "" {
		proxy, err := url.Parse(proxyURL)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error parsing proxy URL: %s\n", err)
			os.Exit(1)
		}
		httpClient.Transport.(*http.Transport).Proxy = http.ProxyURL(proxy)
	}

	httpClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	sc := bufio.NewScanner(os.Stdin)

	initialChecks := make(chan paramCheck, 40)

	appendChecks := makePool(initialChecks, func(c paramCheck, output chan paramCheck) {
		reflected, err := checkReflected(c.url, parsedHeaders)

		if err != nil {
			fmt.Fprintf(os.Stderr, "error from checkReflected: %s\n", err)
			return
		}

		for _, param := range reflected {
			output <- paramCheck{c.url, param}
		}
	})

	charChecks := makePool(appendChecks, func(c paramCheck, output chan paramCheck) {
		output <- paramCheck{c.url, c.param}
	})

	done := makePool(charChecks, func(c paramCheck, output chan paramCheck) {
		output_of_url := []string{c.url, c.param}

		// Extract the hostname from the URL
		parsedURL, err := url.Parse(c.url)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error parsing URL: %s\n", err)
			return
		}
		hostname := parsedURL.Hostname()

		// Define the payloads including the new ones based on the hostname
		payloads := []string{
			"http://quas.sh/",
			"http:/quas.sh",
			"https://quas.sh/",
			"https:/quas.sh",
			fmt.Sprintf("http://%s.quas.sh/", hostname),
			fmt.Sprintf("https://%s.quas.sh/", hostname),
			fmt.Sprintf("http://%s@quas.sh/", hostname),
                        fmt.Sprintf("https://%s@quas.sh/", hostname),
		}

		for _, char := range payloads {
			wasReflected, err := checkAppend(c.url, c.param, char+"asuffix", parsedHeaders)
			if err != nil {
				continue
			}

			if wasReflected {
				output_of_url = append(output_of_url, char)
			}
		}
		if len(output_of_url) > 2 {
			fmt.Printf("URL: %s Param: %s Unfiltered: %v\n", output_of_url[0], output_of_url[1], output_of_url[2:])
		}
	})

	for sc.Scan() {
		initialChecks <- paramCheck{url: sc.Text()}
	}

	close(initialChecks)
	<-done
}

func parseHeaders(headersList []string) http.Header {
	headers := http.Header{}
	for _, header := range headersList {
		parts := strings.SplitN(header, ": ", 2)
		if len(parts) == 2 {
			headers.Add(parts[0], parts[1])
		}
	}
	return headers
}

func checkReflected(targetURL string, headers http.Header) ([]string, error) {
	out := make([]string, 0)

	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		return out, err
	}

	for key, values := range headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return out, err
	}
	defer resp.Body.Close()

	loc := string(resp.Header.Get("Location"))

	u, err := url.Parse(targetURL)
	if err != nil {
		return out, err
	}
	for key, vv := range u.Query() {
		for _, v := range vv {
			if !strings.Contains(loc, v) {
				continue
			}

			out = append(out, key)
		}
	}

	return out, nil
}

func checkOpenRedirect(targetURL string, headers http.Header) ([]string, error) {
	out := make([]string, 0)

	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		return out, err
	}

	for key, values := range headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return out, err
	}
	defer resp.Body.Close()

	loc := string(resp.Header.Get("Location"))

	u, err := url.Parse(targetURL)
	if err != nil {
		return out, err
	}
	for key, vv := range u.Query() {
		for _, v := range vv {
			if !strings.HasPrefix(loc, v) {
				continue
			}

			out = append(out, key)
		}
	}

	return out, nil
}

func checkAppend(targetURL, param, suffix string, headers http.Header) (bool, error) {
	u, err := url.Parse(targetURL)
	if err != nil {
		return false, err
	}

	qs := u.Query()
	qs.Set(param, suffix)
	u.RawQuery = qs.Encode()

	reflected, err := checkOpenRedirect(u.String(), headers)
	if err != nil {
		return false, err
	}

	for _, r := range reflected {
		if r == param {
			return true, nil
		}
	}

	return false, nil
}

type workerFunc func(paramCheck, chan paramCheck)

func makePool(input chan paramCheck, fn workerFunc) chan paramCheck {
	var wg sync.WaitGroup

	output := make(chan paramCheck)
	for i := 0; i < 40; i++ {
		wg.Add(1)
		go func() {
			for c := range input {
				fn(c, output)
			}
			wg.Done()
		}()
	}

	go func() {
		wg.Wait()
		close(output)
	}()

	return output
}
