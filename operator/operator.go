package operator

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/template"
)

type DefaultBackendRequest struct {
	StatusCode    int
	Status        string
	StatusMessage string
	Accept        string
	OriginalURI   string
	Host          string
	Namespace     string
	IngressName   string
	ServiceName   string
	ServicePort   string
}

type DefaultBackendResponse struct {
	StatusCodeMatcher *regexp.Regexp
	Order             int
	Template          *template.Template
}

type Operator struct {
	backends []*DefaultBackendResponse
}

func NewOperator(defaultErrorPageFolder string) (*Operator, error) {
	operator := Operator{
		backends: make([]*DefaultBackendResponse, 0),
	}

	files, err := ioutil.ReadDir(defaultErrorPageFolder)
	if err != nil {
		return nil, err
	}

	for _, f := range files {
		name := strings.Split(f.Name(), ".")[0]
		order := 0
		for i := 0; i < len(name); i++ {
			if string(name[i]) == "x" {
				order++
			}
		}
		r := strings.ReplaceAll(name, "x", "\\d")
		regex, err := regexp.Compile(r)
		if err != nil {
			return nil, err
		}

		dat, err := ioutil.ReadFile(path.Join(defaultErrorPageFolder, f.Name()))
		if err != nil {
			return nil, err
		}

		tpl, err := template.New(f.Name()).Parse(string(dat))
		if err != nil {
			return nil, err
		}

		bResponse := &DefaultBackendResponse{
			StatusCodeMatcher: regex,
			Order:             order,
			Template:          tpl,
		}
		newList := append(operator.backends, bResponse)
		sort.Slice(newList, func(i, j int) bool {
			return newList[i].Order > newList[j].Order
		})
		operator.backends = newList
	}

	return &operator, nil
}

func (o *Operator) ServeHTTP(w http.ResponseWriter, request *http.Request) {
	bRequest, err := parseDefaultBackendRequest(request)
	if err != nil {
		http.Error(w, fmt.Sprintf("Bad request: %s", err.Error()), http.StatusBadRequest)
	}

	// todo: determin which response to use

	for _, backend := range o.backends {
		// match
		if backend.StatusCodeMatcher != nil {
			if !backend.StatusCodeMatcher.MatchString(bRequest.Status) {
				continue
			}
		}

		// use backend
		w.WriteHeader(bRequest.StatusCode)
		err := backend.Template.Execute(w, bRequest)
		if err != nil {
			log.Printf("Failed to execute template: %s", err)
			continue
		}

		return
	}

	fmt.Fprintf(w, "default backend")
	log.Printf("No default-backend found for %+v\n", bRequest)
}

func parseDefaultBackendRequest(request *http.Request) (*DefaultBackendRequest, error) {
	status := request.Header.Get("X-Code")
	if status == "" {
		status = "503"
	}
	statusCode, err := strconv.Atoi(status)
	if err != nil {
		return nil, err
	}

	dbRequest := &DefaultBackendRequest{
		Status:        status,
		StatusCode:    statusCode,
		StatusMessage: http.StatusText(statusCode),
	}

	// todo: parse request

	return dbRequest, nil
}
