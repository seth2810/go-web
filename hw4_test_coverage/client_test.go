package main

import (
	"encoding/json"
	"encoding/xml"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

// код писать тут
func UnauthorizedHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusUnauthorized)
}

func InternalErrorHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusInternalServerError)
}

func BadRequestEmptyHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusBadRequest)
}

func BadRequestUnknownErrorHandler(w http.ResponseWriter, r *http.Request) {
	res, _ := json.Marshal(&SearchErrorResponse{"UnknownError"})
	w.WriteHeader(http.StatusBadRequest)
	w.Write(res)
}

func TimeoutHandler(w http.ResponseWriter, r *http.Request) {
	time.Sleep(time.Second * 2)
}

func EmptyHanlder(w http.ResponseWriter, r *http.Request) {}

type UserData struct {
	ID        int    `xml:"id"`
	Age       int    `xml:"age"`
	About     string `xml:"about"`
	Gender    string `xml:"gender"`
	LastName  string `xml:"last_name"`
	FirstName string `xml:"first_name"`
}

type UsersData struct {
	Users []UserData `xml:"row"`
}

type byID []User

func (a byID) Len() int           { return len(a) }
func (a byID) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byID) Less(i, j int) bool { return a[i].Id < a[j].Id }

type byAge []User

func (a byAge) Len() int           { return len(a) }
func (a byAge) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byAge) Less(i, j int) bool { return a[i].Age < a[j].Age }

type byName []User

func (a byName) Len() int           { return len(a) }
func (a byName) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byName) Less(i, j int) bool { return a[i].Name < a[j].Name }

func readUsersData() (data *UsersData, err error) {
	// read dataset
	content, err := ioutil.ReadFile("dataset.xml")

	if err != nil {
		return
	}

	err = xml.Unmarshal(content, &data)

	return
}

// filters users by query parameter
func filterUsersData(data *UsersData, query string) (users []User) {
	for _, user := range data.Users {
		name := user.FirstName + " " + user.LastName

		if query == "" {
			// append
		} else if strings.Contains(name, query) || strings.Contains(user.About, query) {
			// append
		} else {
			// skip
			continue
		}

		users = append(users, User{
			Id:     user.ID,
			Name:   name,
			Age:    user.Age,
			About:  user.About,
			Gender: user.Gender,
		})
	}

	return
}

func getUsersPage(users []User, offset int, limit int) []User {
	ln := len(users)
	idxStart := offset * 25
	idxEnd := idxStart + limit

	if idxStart >= ln {
		return users[ln:]
	} else if idxEnd >= ln {
		return users[idxStart:ln]
	} else {
		return users[idxStart:idxEnd]
	}
}

func sortUsers(users []User, orderField string, orderBy int) {
	var sortBy sort.Interface

	switch orderField {
	case "Id":
		sortBy = byID(users)
	case "Name":
		sortBy = byName(users)
	case "Age":
		sortBy = byAge(users)
	}

	if orderBy == OrderByAsc {
		sort.Sort(sortBy)
	} else if orderBy == OrderByDesc {
		sort.Sort(sort.Reverse(sortBy))
	}
}

func SearchHandler(w http.ResponseWriter, r *http.Request) {
	// handle order field query parameter
	orderField := r.FormValue("order_field")

	switch orderField {
	case "Id", "Age", "Name":
		// ignore
	case "":
		orderField = "Name"
	default:
		res, _ := json.Marshal(&SearchErrorResponse{"ErrorBadOrderField"})
		w.WriteHeader(http.StatusBadRequest)
		w.Write(res)
		return
	}

	data, err := readUsersData()

	if err != nil {
		panic(err)
	}

	users := filterUsersData(data, r.FormValue("query"))

	orderBy, _ := strconv.Atoi(r.FormValue("order_by"))

	sortUsers(users, orderField, orderBy)

	offset, _ := strconv.Atoi(r.FormValue("offset"))
	limit, _ := strconv.Atoi(r.FormValue("limit"))

	page := getUsersPage(users, offset, limit)

	if res, err := json.Marshal(page); err != nil {
		panic(err)
	} else {
		w.Write(res)
	}
}

func TestUnknownError(t *testing.T) {
	c := SearchClient{
		URL: "http://",
	}

	_, err := c.FindUsers(SearchRequest{})

	checkSearchCaseError(t, err, "unknown error", strings.HasPrefix)
}

func TestSearchTimeoutError(t *testing.T) {
	executeSearchCase(t, &searchCase{
		request: SearchRequest{},
		handler: http.HandlerFunc(TimeoutHandler),
		check: func(t *testing.T, res *SearchResponse, err error) {
			checkSearchCaseError(t, err, "timeout for", strings.HasPrefix)
		},
	})
}

func TestSearchUnauthorizedError(t *testing.T) {
	executeSearchCase(t, &searchCase{
		request: SearchRequest{},
		handler: http.HandlerFunc(UnauthorizedHandler),
		check: func(t *testing.T, res *SearchResponse, err error) {
			checkSearchCaseError(t, err, "Bad AccessToken", isStringsEquals)
		},
	})
}

func TestSearchInternalError(t *testing.T) {
	executeSearchCase(t, &searchCase{
		request: SearchRequest{},
		handler: http.HandlerFunc(InternalErrorHandler),
		check: func(t *testing.T, res *SearchResponse, err error) {
			checkSearchCaseError(t, err, "SearchServer fatal error", isStringsEquals)
		},
	})
}

func TestSearchBadRequestUnpackError(t *testing.T) {
	executeSearchCase(t, &searchCase{
		request: SearchRequest{},
		handler: http.HandlerFunc(BadRequestEmptyHandler),
		check: func(t *testing.T, res *SearchResponse, err error) {
			checkSearchCaseError(t, err, "cant unpack error json:", strings.HasPrefix)
		},
	})
}

func TestSearchBadOrderFieldError(t *testing.T) {
	executeSearchCase(t, &searchCase{
		request: SearchRequest{
			OrderField: "unexistent",
		},
		handler: http.HandlerFunc(SearchHandler),
		check: func(t *testing.T, res *SearchResponse, err error) {
			checkSearchCaseError(t, err, "OrderFeld unexistent invalid", isStringsEquals)
		},
	})
}

func TestSearchBadRequersUnknownError(t *testing.T) {
	executeSearchCase(t, &searchCase{
		request: SearchRequest{},
		handler: http.HandlerFunc(BadRequestUnknownErrorHandler),
		check: func(t *testing.T, res *SearchResponse, err error) {
			checkSearchCaseError(t, err, "unknown bad request error:", strings.HasPrefix)
		},
	})
}

func TestSearchUnpackError(t *testing.T) {
	executeSearchCase(t, &searchCase{
		request: SearchRequest{},
		handler: http.HandlerFunc(EmptyHanlder),
		check: func(t *testing.T, res *SearchResponse, err error) {
			checkSearchCaseError(t, err, "cant unpack result json:", strings.HasPrefix)
		},
	})
}

func TestSearchNegativeLimit(t *testing.T) {
	executeSearchCase(t, &searchCase{
		request: SearchRequest{
			Limit: -1,
		},
		handler: http.HandlerFunc(SearchHandler),
		check: func(t *testing.T, res *SearchResponse, err error) {
			checkSearchCaseError(t, err, "limit must be > 0", isStringsEquals)
		},
	})
}

func TestSearchLimitMax(t *testing.T) {
	executeSearchCase(t, &searchCase{
		request: SearchRequest{
			Limit: 27,
		},
		handler: http.HandlerFunc(SearchHandler),
		check: func(t *testing.T, res *SearchResponse, err error) {
			if len(res.Users) > 25 {
				t.Error("Search does not limit maximum page size")
			}
		},
	})
}

func TestSearchLimit(t *testing.T) {
	executeSearchCase(t, &searchCase{
		request: SearchRequest{
			Limit: 3,
		},
		handler: http.HandlerFunc(SearchHandler),
		check: func(t *testing.T, res *SearchResponse, err error) {
			if len(res.Users) != 3 {
				t.Error("Search does not limit page size")
			}
		},
	})
}

func TestSearchNegativeOffset(t *testing.T) {
	executeSearchCase(t, &searchCase{
		request: SearchRequest{
			Offset: -1,
		},
		handler: http.HandlerFunc(SearchHandler),
		check: func(t *testing.T, res *SearchResponse, err error) {
			checkSearchCaseError(t, err, "offset must be > 0", isStringsEquals)
		},
	})
}

func TestSearchOffset(t *testing.T) {
	executeSearchCase(t, &searchCase{
		request: SearchRequest{
			Offset: 1,
			Limit:  1,
		},
		handler: http.HandlerFunc(SearchHandler),
		check: func(t *testing.T, res *SearchResponse, err error) {
			if len(res.Users) != 1 && res.Users[0].Id != 25 {
				t.Error("Search does not apply offset")
			}
		},
	})
}

func TestSearchWithNextPage(t *testing.T) {
	executeSearchCase(t, &searchCase{
		request: SearchRequest{},
		handler: http.HandlerFunc(SearchHandler),
		check: func(t *testing.T, res *SearchResponse, err error) {
			if res.NextPage != true {
				t.Error("NextPage is not true")
			}
		},
	})
}

func TestSearchWithoutNextPage(t *testing.T) {
	executeSearchCase(t, &searchCase{
		request: SearchRequest{
			Offset: 2,
		},
		handler: http.HandlerFunc(SearchHandler),
		check: func(t *testing.T, res *SearchResponse, err error) {
			if res.NextPage != false {
				t.Error("NextPage is not false")
			}
		},
	})
}

func TestSearchQueryName(t *testing.T) {
	executeSearchCase(t, &searchCase{
		request: SearchRequest{
			Limit: 1,
			Query: "Hilda Mayer",
		},
		handler: http.HandlerFunc(SearchHandler),
		check: func(t *testing.T, res *SearchResponse, err error) {
			if len(res.Users) != 1 {
				t.Error("Search founded more than one record by unique user name")
			}
		},
	})
}

func TestSearchQueryAbout(t *testing.T) {
	executeSearchCase(t, &searchCase{
		request: SearchRequest{
			Limit: 1,
			Query: "Ipsum aliqua",
		},
		handler: http.HandlerFunc(SearchHandler),
		check: func(t *testing.T, res *SearchResponse, err error) {
			if len(res.Users) != 1 {
				t.Error("Search founded more than one record by unique about substring")
			}
		},
	})
}

// HELPERS
type searchCase struct {
	handler http.Handler
	request SearchRequest
	check   func(*testing.T, *SearchResponse, error)
}

func executeSearchCase(t *testing.T, sc *searchCase) {
	s := httptest.NewServer(sc.handler)

	c := &SearchClient{
		URL: s.URL,
	}

	res, err := c.FindUsers(sc.request)

	sc.check(t, res, err)

	defer s.Close()
}

func checkSearchCaseError(
	t *testing.T,
	actual error,
	expected string,
	condition func(a string, b string) bool,
) {
	if condition(actual.Error(), expected) != true {
		t.Errorf("Wrong error result, expected '%v', got '%v'", expected, actual.Error())
	}
}

// wrapper of strings.Compare
func isStringsEquals(a string, b string) bool {
	return strings.Compare(a, b) == 0
}
