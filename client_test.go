package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

const (
	accToken = "secure"
)

type TestCases struct {
	TestRequest SearchRequest
	IsError     bool
	AccessToken string
	URL string
}

type TestUsers struct {
	Users []User
}

func (tu *TestUsers) sortByLimitAndOffset(limit, offset int) error {

	users := tu.Users
	if len(users) >= offset {
		users = users[offset:]
	LOOP:
		for {
			if limit < len(users) {
				users = users[:len(users)-1]
			} else {
				break LOOP
			}

		}

		tu.Users = users

		return nil
	} else {
		err := fmt.Errorf("unavailable offset")
		log.Println(err)
		return err
	}

}

func (tu *TestUsers) sortByValidateField(query string) {

	if query != "" {
		users := tu.Users
		validatedUsers := []User{}
		for _, user := range users {
			if strings.Contains(user.Name, query) {
				validatedUsers = append(validatedUsers, user)
			} else if strings.Contains(user.About, query) {
				validatedUsers = append(validatedUsers, user)
			}
		}

		tu.Users = validatedUsers
	}
}

func (tu *TestUsers) sortByField(field, orderBy string) error {

	switch field {
	case "Id", "id", "Age", "age", "Name", "name":
		strings.ToLower(field)

		switch field {
		case "id":
			switch orderBy {
			case "1":
				sort.Slice(tu.Users, func(i, j int) bool {
					return tu.Users[i].Id < tu.Users[j].Id
				})
			case "-1":
				sort.Slice(tu.Users, func(i, j int) bool {
					return tu.Users[i].Id > tu.Users[j].Id
				})
			case "0":
				break
			default:
				err := fmt.Errorf("ErrorBadOrderBY")
				log.Println(err)
				return err
			}
		case "name":
			switch orderBy {
			case "1":
				sort.Slice(tu.Users, func(i, j int) bool {
					return tu.Users[i].Name < tu.Users[j].Name
				})
			case "-1":
				sort.Slice(tu.Users, func(i, j int) bool {
					return tu.Users[i].Name > tu.Users[j].Name
				})
			case "0":
				break
			default:
				err := fmt.Errorf("ErrorBadOrderBY")
				log.Println(err)
				return err
			}
		case "age":
			switch orderBy {
			case "1":
				sort.Slice(tu.Users, func(i, j int) bool {
					return tu.Users[i].Age < tu.Users[j].Age
				})
			case "-1":
				sort.Slice(tu.Users, func(i, j int) bool {
					return tu.Users[i].Age > tu.Users[j].Age
				})
			case "0":
				break
			default:
				err := fmt.Errorf("ErrorBadOrderBY")
				log.Println(err)
				return err
			}
		}
	case "":
		switch orderBy {
		case "1":
			sort.Slice(tu.Users, func(i, j int) bool {
				return tu.Users[i].Age < tu.Users[j].Age
			})
		case "-1":
			sort.Slice(tu.Users, func(i, j int) bool {
				return tu.Users[i].Age > tu.Users[j].Age
			})
		case "0":
			break
		default:
			err := fmt.Errorf("ErrorBadOrderBY")
			log.Println(err)
			return err
		}
	default:
		err := fmt.Errorf("ErrorBadOrderField")
		log.Println(err)
		return err
	}

	return nil
}

func (tu *TestUsers) parseFile() error {

	file, err := os.Open("dataset.xml")
	if err != nil {
		log.Println(err)
		return err
	}
	defer file.Close()

	fileContent, err := ioutil.ReadAll(file)
	if err != nil {
		log.Println(err)
		return err
	}

	var (
		input = bytes.NewReader(fileContent)
		decoder = xml.NewDecoder(input)
		firstName, lastName string
		user User
	)

	for {
		tok, tokErr := decoder.Token()
		if tokErr != nil && tokErr != io.EOF {
			log.Println(tokErr)
			return tokErr
		} else if tokErr == io.EOF {
			break
		}

		switch tok := tok.(type) {
		case xml.StartElement:
			switch tok.Name.Local {
			case "id":
				if err := decoder.DecodeElement(&user.Id, &tok); err != nil {
					log.Println(err)
					return err
				}
				tu.Users = append(tu.Users, user)
			case "first_name":
				if err := decoder.DecodeElement(&firstName, &tok); err != nil {
					log.Println(err)
					return err
				}
			case "last_name":
				if err := decoder.DecodeElement(&lastName, &tok); err != nil {
					log.Println(err)
					return err
				}
				user.Name = firstName + " " + lastName
			case "age":
				if err := decoder.DecodeElement(&user.Age, &tok); err != nil {
					log.Println(err)
					return err
				}
			case "about":
				if err := decoder.DecodeElement(&user.About, &tok); err != nil {
					log.Println(err)
					return err
				}
			case "gender":
				if err := decoder.DecodeElement(&user.Gender, &tok); err != nil {
					log.Println(err)
					return err
				}
			}
		}
	}

	return nil
}

func SearchServer(w http.ResponseWriter, r *http.Request) {

	var(
		token = r.Header.Get("AccessToken")
		params = r.URL.Query()
		orderField = params.Get("order_field")
		orderBy = params.Get("order_by")
		query = params.Get("query")
		limit, _ = strconv.Atoi(params.Get("limit"))
		offset, _ = strconv.Atoi(params.Get("offset"))

		users = &TestUsers{}
	)

	if token != accToken {
		http.Error(w, "", http.StatusUnauthorized)
		return
	}
	if err := users.parseFile(); err != nil {
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	users.sortByValidateField(query)

	if err := users.sortByField(orderField, orderBy); err != nil {
		errorOutput := SearchErrorResponse{err.Error()}
		response, err := json.Marshal(errorOutput)
		if err != nil {
			http.Error(w, string(response), http.StatusInternalServerError)
			log.Println(err)
			return
		}

		http.Error(w, string(response), http.StatusBadRequest)
		return
	}

	if err := users.sortByLimitAndOffset(limit, offset); err != nil {
		errorOutput := SearchErrorResponse{Error: err.Error()}
		response, err := json.Marshal(errorOutput)
		if err != nil {
			http.Error(w, string(response), http.StatusInternalServerError)
			log.Println(err)
			return
		}

		http.Error(w, string(response), http.StatusBadRequest)
		return
	}

	output, err := json.Marshal(users.Users)
	if err != nil {
		http.Error(w, "", http.StatusInternalServerError)
		log.Println(err)
		return
	}

	io.WriteString(w, string(output))
}

func TestLimit(t *testing.T) {

	cases := []TestCases{
		{
			TestRequest: SearchRequest{
				Limit: -1,
				Offset: 2,
				Query: "",
				OrderField: "id",
				OrderBy: OrderByAsc,
			},
			IsError: true,
		},
		{
			TestRequest: SearchRequest{
				Limit: 100,
				Offset: 2,
				Query: "",
				OrderField: "id",
				OrderBy: OrderByAsc,
			},
			IsError: false,
		},

	}

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()

	for caseNum, params := range cases {
		req := &SearchClient{
			AccessToken: accToken,
			URL:         ts.URL,
		}
		_, err := req.FindUsers(params.TestRequest)
		if err != nil && !params.IsError{
			t.Errorf("Case №%d expected error: %#v", caseNum, err)
		}
		if err == nil && params.IsError{
			t.Errorf("Case №%d unexpected error: %#v", caseNum, err)
		}
	}
}

func TestOffset(t *testing.T) {

	cases := []TestCases{
		{
			TestRequest: SearchRequest{
				Limit: 1,
				Offset: -1,
				Query: "",
				OrderField: "id",
				OrderBy: OrderByAsc,
			},
			IsError: true,
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()

	for caseNum, params := range cases {
		req := &SearchClient{
			AccessToken: accToken,
			URL:         ts.URL,
		}
		_, err := req.FindUsers(params.TestRequest)
		if err != nil && !params.IsError{
			t.Errorf("Case №%d expected error: %#v", caseNum, err)
		}
		if err == nil && params.IsError{
			t.Errorf("Case №%d unexpected error: %#v", caseNum, err)
		}
	}
}

func TestQuery(t *testing.T) {

	cases := []TestCases{
		{
			TestRequest: SearchRequest{
				Limit: 1,
				Offset: 1,
				Query: "",
				OrderField: "id",
				OrderBy: OrderByAsc,
			},
			IsError: false,
		},
		{
			TestRequest: SearchRequest{
				Limit: 1,
				Offset: 1,
				Query: "Travis",
				OrderField: "id",
				OrderBy: OrderByAsc,
			},
			IsError: false,
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()

	for caseNum, params := range cases {
		req := &SearchClient{
			AccessToken: accToken,
			URL:         ts.URL,
		}
		_, err := req.FindUsers(params.TestRequest)
		if err != nil && !params.IsError{
			t.Errorf("Case №%d expected error: %#v", caseNum, err)
		}
		if err == nil && params.IsError{
			t.Errorf("Case №%d unexpected error: %#v", caseNum, err)
		}
	}
}

func TestAccessToken(t *testing.T) {

	cases := []TestCases{
		{
			TestRequest: SearchRequest{
				Limit: 1,
				Offset: 1,
				Query: "",
				OrderField: "id",
				OrderBy: OrderByAsc,
			},
			AccessToken: "token",
			IsError: true,
		},
		{
			TestRequest: SearchRequest{
				Limit: 1,
				Offset: 1,
				Query: "Travis",
				OrderField: "id",
				OrderBy: OrderByAsc,
			},
			AccessToken: accToken,
			IsError: false,
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()

	for caseNum, params := range cases {
		req := &SearchClient{
			AccessToken: params.AccessToken,
			URL:         ts.URL,
		}
		_, err := req.FindUsers(params.TestRequest)
		if err != nil && !params.IsError{
			t.Errorf("Case №%d expected error: %#v", caseNum, err)
		}
		if err == nil && params.IsError{
			t.Errorf("Case №%d unexpected error: %#v", caseNum, err)
		}
	}
}

func TestOrderField(t *testing.T) {

	cases := []TestCases{
		{
			TestRequest: SearchRequest{
				Limit: 1,
				Offset: 1,
				Query: "",
				OrderField: "errorField",
				OrderBy: OrderByAsc,
			},
			IsError: true,
		},
		{
			TestRequest: SearchRequest{
				Limit: 1,
				Offset: 1,
				Query: "Travis",
				OrderField: "id",
				OrderBy: OrderByAsc,
			},
			IsError: false,
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()

	for caseNum, params := range cases {
		req := &SearchClient{
			AccessToken: accToken,
			URL:         ts.URL,
		}
		_, err := req.FindUsers(params.TestRequest)
		if err != nil && !params.IsError{
			t.Errorf("Case №%d expected error: %#v", caseNum, err)
		}
		if err == nil && params.IsError{
			t.Errorf("Case №%d unexpected error: %#v", caseNum, err)
		}
	}
}

func TestOrderBy(t *testing.T) {

	cases := []TestCases{
		{
			TestRequest: SearchRequest{
				Limit: 1,
				Offset: 1,
				Query: "",
				OrderField: "name",
				OrderBy: 2,
			},
			IsError: true,
		},
		{
			TestRequest: SearchRequest{
				Limit: 1,
				Offset: 1,
				Query: "Travis",
				OrderField: "id",
				OrderBy: OrderByAsc,
			},
			IsError: false,
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()

	for caseNum, params := range cases {
		req := &SearchClient{
			AccessToken: accToken,
			URL:         ts.URL,
		}
		_, err := req.FindUsers(params.TestRequest)
		if err != nil && !params.IsError{
			t.Errorf("Case №%d expected error: %#v", caseNum, err)
		}
		if err == nil && params.IsError{
			t.Errorf("Case №%d unexpected error: %#v", caseNum, err)
		}
	}
}

func ErrorSearchServer(w http.ResponseWriter, r *http.Request) {

	var (
		params = r.URL.Query()
		offset, _ = strconv.Atoi(params.Get("offset"))
		query = params.Get("query")
	)

	if offset > 50 {
		errorOutput := SearchErrorResponse{}
		response, err := json.Marshal(errorOutput)
		if err != nil {
			http.Error(w, string(response), http.StatusInternalServerError)
		}

		http.Error(w, string(response), http.StatusInternalServerError)
	} else if strings.Contains(query, "&") {
		errorOutput := []SearchErrorResponse{}
		response, err := json.Marshal(errorOutput)
		if err != nil {
			http.Error(w, string(response), http.StatusInternalServerError)
		}

		http.Error(w, string(response), http.StatusBadRequest)
	} else if query == "" {
		time.Sleep(1 * time.Second)
		http.Error(w, "", http.StatusBadRequest)
	} else if query == "errorJson" {
		output := User{}
		response, err := json.Marshal(output)
		if err != nil {
			http.Error(w, string(response), http.StatusInternalServerError)
		}

		io.WriteString(w, string(response))
	} else {
		output := []User{}
		response, err := json.Marshal(output)
		if err != nil {
			http.Error(w, string(response), http.StatusInternalServerError)
		}

		io.WriteString(w, string(response))
	}
}

func TestErrorSearchServerInternalError(t *testing.T) {

	cases := []TestCases{
		{
			TestRequest: SearchRequest{
				Limit: 1,
				Offset: 51,
				Query: "",
				OrderField: "name",
				OrderBy: OrderByAsIs,
			},
			IsError: true,
		},
		{
			TestRequest: SearchRequest{
				Limit: 1,
				Offset: 1,
				Query: "Travis",
				OrderField: "id",
				OrderBy: OrderByAsc,
			},
			IsError: false,
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(ErrorSearchServer))
	defer ts.Close()

	for caseNum, params := range cases {
		req := &SearchClient{
			AccessToken: accToken,
			URL:         ts.URL,
		}
		_, err := req.FindUsers(params.TestRequest)
		if err != nil && !params.IsError{
			t.Errorf("Case №%d expected error: %#v", caseNum, err)
		}
		if err == nil && params.IsError{
			t.Errorf("Case №%d unexpected error: %#v", caseNum, err)
		}
	}
}

func TestErrorSearchServerBadRequest(t *testing.T) {

	cases := []TestCases{
		{
			TestRequest: SearchRequest{
				Limit: 1,
				Offset: 2,
				Query: "&",
				OrderField: "name",
				OrderBy: OrderByAsIs,
			},
			IsError: true,
		},
		{
			TestRequest: SearchRequest{
				Limit: 1,
				Offset: 1,
				Query: "Travis",
				OrderField: "id",
				OrderBy: OrderByAsc,
			},
			IsError: false,
		},
	}
	ts := httptest.NewServer(http.HandlerFunc(ErrorSearchServer))
	defer ts.Close()

	for caseNum, params := range cases {
		req := &SearchClient{
			AccessToken: accToken,
			URL:         ts.URL,
		}
		_, err := req.FindUsers(params.TestRequest)
		if err != nil && !params.IsError{
			t.Errorf("Case №%d expected error: %#v", caseNum, err)
		}
		if err == nil && params.IsError{
			t.Errorf("Case №%d unexpected error: %#v", caseNum, err)
		}
	}
}

func TestErrorServerTimeOutAndURL(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(ErrorSearchServer))
	defer ts.Close()

	cases := []TestCases{
		{
			TestRequest: SearchRequest{
				Limit: 1,
				Offset: 2,
				Query: "",
				OrderField: "name",
				OrderBy: OrderByAsIs,
			},
			URL: ts.URL,
			IsError: true,
		},
		{
			TestRequest: SearchRequest{
				Limit: 1,
				Offset: 1,
				Query: "Travis",
				OrderField: "id",
				OrderBy: OrderByAsc,
			},
			URL: ts.URL,
			IsError: false,
		},
		{
			TestRequest: SearchRequest{
				Limit: 1,
				Offset: 2,
				Query: "Trav",
				OrderField: "name",
				OrderBy: OrderByAsIs,
			},
			URL: "errorURL",
			IsError: true,
		},
		{
			TestRequest: SearchRequest{
				Limit: 1,
				Offset: 1,
				Query: "Travis",
				OrderField: "id",
				OrderBy: OrderByAsc,
			},
			URL: ts.URL,
			IsError: false,
		},

	}
	for caseNum, params := range cases {
		req := &SearchClient{
			AccessToken: accToken,
			URL:         params.URL,
		}
		_, err := req.FindUsers(params.TestRequest)
		if err != nil && !params.IsError{
			t.Errorf("Case №%d expected error: %#v", caseNum, err)
		}
		if err == nil && params.IsError{
			t.Errorf("Case №%d unexpected error: %#v", caseNum, err)
		}
	}
}

func TestErrorServerErrorJson(t *testing.T) {
	cases := []TestCases{
		{
			TestRequest: SearchRequest{
				Limit: 1,
				Offset: 2,
				Query: "errorJson",
				OrderField: "name",
				OrderBy: OrderByAsIs,
			},
			IsError: true,
		},
		{
			TestRequest: SearchRequest{
				Limit: 1,
				Offset: 1,
				Query: "Travis",
				OrderField: "id",
				OrderBy: OrderByAsc,
			},
			IsError: false,
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(ErrorSearchServer))
	defer ts.Close()

	for caseNum, params := range cases {
		req := &SearchClient{
			AccessToken: accToken,
			URL:         ts.URL,
		}
		_, err := req.FindUsers(params.TestRequest)
		if err != nil && !params.IsError{
			t.Errorf("Case №%d expected error: %#v", caseNum, err)
		}
		if err == nil && params.IsError{
			t.Errorf("Case №%d unexpected error: %#v", caseNum, err)
		}
	}
}