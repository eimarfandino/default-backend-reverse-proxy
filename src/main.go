package main

import (
	"bytes"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
)

/*
	Utilities
*/

// Get env var or default
func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

/*
	Getters
*/

// Get the port to listen on
func getListenAddress() string {
	port := flag.Lookup("port").Value.(flag.Getter).Get().(string)
	return ":" + port
}

// Get the url for a given header
func getProxyURL(originalPathHeader string) string {

	if originalPathHeader == "" {
		return ""
	}

	defaultBackendNamespace := flag.Lookup("namespace").Value.(flag.Getter).Get().(string)
	defaultBackendService := flag.Lookup("service").Value.(flag.Getter).Get().(string)
	defaultBackendServicePort := flag.Lookup("service-port").Value.(flag.Getter).Get().(string)

	log.Printf("Original path: %s", originalPathHeader)

	var buffer bytes.Buffer
	buffer.WriteString("http://")
	buffer.WriteString(defaultBackendService)
	buffer.WriteString(".")
	buffer.WriteString(defaultBackendNamespace)
	buffer.WriteString(".")
	buffer.WriteString("svc.cluster.local")
	buffer.WriteString(":")
	buffer.WriteString(defaultBackendServicePort)
	buffer.WriteString(originalPathHeader)

	return buffer.String()
}

/*
	Logging
*/

// Log the typeform payload and redirect url
func logRequestPayload(originalPath string, proxyURL string) {
	log.Printf("original_path: %s, proxy_url: %s\n", originalPath, proxyURL)
}

// Log the env variables required for a reverse proxy
func logSetup() {

	defaultBackendNamespace := flag.Lookup("namespace").Value.(flag.Getter).Get().(string)
	defaultBackendService := flag.Lookup("service").Value.(flag.Getter).Get().(string)
	defaultBackendServicePort := flag.Lookup("service-port").Value.(flag.Getter).Get().(string)

	log.Printf("Server will run on: %s\n", getListenAddress())
	log.Printf("Redirecting to service: %s:%s in namespace %s\n", defaultBackendService, defaultBackendServicePort, defaultBackendNamespace)
}

/*
	Reverse Proxy Logic
*/
func return404(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	file, err := ioutil.ReadFile("./assets/404.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Fatal("Can't find error html page")
	}
	w.Write(file)
}

// Serve a reverse proxy for a given url
func serveReverseProxy(target string, host string, res http.ResponseWriter, req *http.Request) {
	// parse the url
	url, _ := url.Parse(target)

	// create the reverse proxy

	proxy := httputil.NewSingleHostReverseProxy(url)

	// Update the headers to allow for SSL redirection

	req.URL.Host = host
	req.URL.Scheme = url.Scheme
	req.Host = host

	// Note that ServeHttp is non blocking and uses a go routine under the hood
	proxy.ServeHTTP(res, req)
}

// Given a request send it to the appropriate url
func handleRequestAndRedirect(res http.ResponseWriter, req *http.Request) {

	requestOriginalPath := req.Header.Get("x-original-uri")
	host := flag.Lookup("host").Value.(flag.Getter).Get().(string)

	url := getProxyURL(requestOriginalPath)
	if url == "" {
		return404(res, req)
	} else {
		logRequestPayload(requestOriginalPath, url)
		serveReverseProxy(url, host, res, req)
	}
}

/*
	Entry
*/
func main() {

	flag.String("port", "8080", "port used to expose the default backend")
	flag.String("service", "service-name", "the name of the service where the request will be proxied")
	flag.String("service-port", "8080", "the port where the service is listening to")
	flag.String("namespace", "namespace", "the namespace where the default backend service is running")
	flag.String("host", "host", "hostname used to forward the calls")

	flag.Parse()

	// Log setup values
	logSetup()

	// start server
	http.HandleFunc("/", handleRequestAndRedirect)

	// return 200 on healthz path
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("healthy!"))
	})

	if err := http.ListenAndServe(getListenAddress(), nil); err != nil {
		panic(err)
	}
}
