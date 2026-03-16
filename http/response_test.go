package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestResponseWithPagination(t *testing.T) {
	data := []string{"item1", "item2"}
	count := int64(10)
	pageSize := uint(2)
	pageNumber := uint(1)

	resp := ResponseWithPagination(data, count, pageSize, pageNumber)

	assert.NotNil(t, resp)
	assert.Equal(t, data, resp.Data)
	assert.Equal(t, uint(5), *resp.TotalPages)
	assert.Equal(t, uint64(10), *resp.Count)
	assert.Equal(t, uint(2), *resp.PageSize)
	assert.Equal(t, uint(1), *resp.PageNumber)
}

func TestHttpUtil_Headers(t *testing.T) {
	ctx := context.Background()
	util := New(ctx)

	newHeaders := http.Header{
		"X-Custom": []string{"value1"},
	}
	util.SetHeader(newHeaders)
	assert.Equal(t, "value1", util.header.Get("X-Custom"))

	util.AddHeader(http.Header{"X-Another": []string{"value2"}})
	assert.Equal(t, "value1", util.header.Get("X-Custom"))
	assert.Equal(t, "value2", util.header.Get("X-Another"))
}

func TestHttpUtil_GetRest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	ctx := context.Background()
	util := New(ctx)
	util.SetPrintCurl(false)

	resp, err := util.GetRest(server.URL, 2*time.Second)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, string(resp.Data), "ok")
}

func TestHttpUtil_PostRest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		
		assert.Equal(t, "test", body["input"])
		
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":"123"}`))
	}))
	defer server.Close()

	ctx := context.Background()
	util := New(ctx)
	
	payload := map[string]string{"input": "test"}
	resp, err := util.PostRest(server.URL, payload, 2*time.Second)
	
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	assert.Contains(t, string(resp.Data), "123")
}
