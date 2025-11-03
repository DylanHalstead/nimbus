package nimbus

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Test types
type TestParams struct {
	ID string `path:"id"`
}

type TestBody struct {
	Name  string `json:"name" validate:"required,minlen=3"`
	Email string `json:"email" validate:"required,email"`
}

type TestQuery struct {
	Page  int    `json:"page" query:"page" validate:"min=1"`
	Limit int    `json:"limit" query:"limit" validate:"min=1,max=100"`
	Sort  string `json:"sort" query:"sort"`
}

// Test validators
var (
	testParamsValidator = NewValidator(&TestParams{})
	testBodyValidator   = NewValidator(&TestBody{})
	testQueryValidator  = NewValidator(&TestQuery{})
)

func TestWithTyped_OnlyParams(t *testing.T) {
	router := NewRouter()

	// Handler that only uses path params
	handler := func(ctx *Context, req *TypedRequest[TestParams, TestBody, TestQuery]) (any, int, error) {
		if req.Params == nil {
			t.Fatal("params should not be nil")
		}
		if req.Body != nil {
			t.Fatal("body should be nil")
		}
		if req.Query != nil {
			t.Fatal("query should be nil")
		}

		return map[string]string{"id": req.Params.ID}, http.StatusOK, nil
	}

	router.AddRoute(http.MethodGet, "/items/:id",
		WithTyped(handler, testParamsValidator, nil, nil))

	req := httptest.NewRequest(http.MethodGet, "/items/123", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response SuccessResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}

	data, ok := response.Data.(map[string]any)
	if !ok {
		t.Fatal("expected data to be a map")
	}

	if data["id"] != "123" {
		t.Errorf("expected id to be '123', got %v", data["id"])
	}
}

func TestWithTyped_OnlyBody(t *testing.T) {
	router := NewRouter()

	// Handler that only uses body
	handler := func(ctx *Context, req *TypedRequest[TestParams, TestBody, TestQuery]) (any, int, error) {
		if req.Params != nil {
			t.Fatal("params should be nil")
		}
		if req.Body == nil {
			t.Fatal("body should not be nil")
		}
		if req.Query != nil {
			t.Fatal("query should be nil")
		}

		return map[string]string{
			"name":  req.Body.Name,
			"email": req.Body.Email,
		}, http.StatusCreated, nil
	}

	router.AddRoute(http.MethodPost, "/users",
		WithTyped(handler, nil, testBodyValidator, nil))

	bodyData := map[string]string{
		"name":  "John Doe",
		"email": "john@example.com",
	}
	bodyJSON, _ := json.Marshal(bodyData)

	req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestWithTyped_OnlyQuery(t *testing.T) {
	router := NewRouter()

	// Handler that only uses query params
	handler := func(ctx *Context, req *TypedRequest[TestParams, TestBody, TestQuery]) (any, int, error) {
		if req.Params != nil {
			t.Fatal("params should be nil")
		}
		if req.Body != nil {
			t.Fatal("body should be nil")
		}
		if req.Query == nil {
			t.Fatal("query should not be nil")
		}

		return map[string]any{
			"page":  req.Query.Page,
			"limit": req.Query.Limit,
			"sort":  req.Query.Sort,
		}, http.StatusOK, nil
	}

	router.AddRoute(http.MethodGet, "/items",
		WithTyped(handler, nil, nil, testQueryValidator))

	req := httptest.NewRequest(http.MethodGet, "/items?page=2&limit=10&sort=name", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response SuccessResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}

	data, ok := response.Data.(map[string]any)
	if !ok {
		t.Fatal("expected data to be a map")
	}

	if data["page"].(float64) != 2 {
		t.Errorf("expected page to be 2, got %v", data["page"])
	}
	if data["limit"].(float64) != 10 {
		t.Errorf("expected limit to be 10, got %v", data["limit"])
	}
	if data["sort"] != "name" {
		t.Errorf("expected sort to be 'name', got %v", data["sort"])
	}
}

func TestWithTyped_AllThree(t *testing.T) {
	router := NewRouter()

	// Handler that uses all three: params, body, and query
	handler := func(ctx *Context, req *TypedRequest[TestParams, TestBody, TestQuery]) (any, int, error) {
		if req.Params == nil {
			t.Fatal("params should not be nil")
		}
		if req.Body == nil {
			t.Fatal("body should not be nil")
		}
		if req.Query == nil {
			t.Fatal("query should not be nil")
		}

		return map[string]any{
			"id":    req.Params.ID,
			"name":  req.Body.Name,
			"email": req.Body.Email,
			"page":  req.Query.Page,
		}, http.StatusOK, nil
	}

	router.AddRoute(http.MethodPut, "/users/:id",
		WithTyped(handler, testParamsValidator, testBodyValidator, testQueryValidator))

	bodyData := map[string]string{
		"name":  "Jane Doe",
		"email": "jane@example.com",
	}
	bodyJSON, _ := json.Marshal(bodyData)

	req := httptest.NewRequest(http.MethodPut, "/users/456?page=3&limit=20", bytes.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response SuccessResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}

	data, ok := response.Data.(map[string]any)
	if !ok {
		t.Fatal("expected data to be a map")
	}

	if data["id"] != "456" {
		t.Errorf("expected id to be '456', got %v", data["id"])
	}
	if data["name"] != "Jane Doe" {
		t.Errorf("expected name to be 'Jane Doe', got %v", data["name"])
	}
	if data["page"].(float64) != 3 {
		t.Errorf("expected page to be 3, got %v", data["page"])
	}
}

func TestWithTyped_NoParameters(t *testing.T) {
	router := NewRouter()

	// Handler with no parameters at all
	handler := func(ctx *Context, req *TypedRequest[TestParams, TestBody, TestQuery]) (any, int, error) {
		if req.Params != nil {
			t.Fatal("params should be nil")
		}
		if req.Body != nil {
			t.Fatal("body should be nil")
		}
		if req.Query != nil {
			t.Fatal("query should be nil")
		}

		return map[string]string{"message": "Hello World"}, http.StatusOK, nil
	}

	router.AddRoute(http.MethodGet, "/hello",
		WithTyped(handler, nil, nil, nil))

	req := httptest.NewRequest(http.MethodGet, "/hello", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}
