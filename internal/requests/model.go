package requests

import (
	"encoding/base64"
	"encoding/json"
	"time"
	"unicode/utf8"
)

// RequestData captures details about the inbound HTTP request.
type RequestData struct {
	Method  string              `json:"method"`
	URL     string              `json:"url"`
	Path    string              `json:"path"`
	Query   map[string][]string `json:"query"`
	Headers map[string][]string `json:"headers"`
	Body    []byte              `json:"-"`
	Size    int64               `json:"size"`
}

// MarshalJSON provides developer-friendly JSON output for RequestData,
// representing UTF-8 bodies as readable strings and binary bodies as base64.
func (r RequestData) MarshalJSON() ([]byte, error) {
	type Alias RequestData
	isBinary := !utf8.Valid(r.Body)
	var bodyStr string
	if isBinary {
		bodyStr = base64.StdEncoding.EncodeToString(r.Body)
	} else {
		bodyStr = string(r.Body)
	}

	return json.Marshal(&struct {
		Alias
		Body     string `json:"body"`
		IsBinary bool   `json:"is_binary"`
	}{
		Alias:    Alias(r),
		Body:     bodyStr,
		IsBinary: isBinary,
	})
}

// UnmarshalJSON handles deserializing RequestData from JSON.
func (r *RequestData) UnmarshalJSON(data []byte) error {
	type Alias RequestData
	aux := &struct {
		Alias
		Body     string `json:"body"`
		IsBinary bool   `json:"is_binary"`
	}{}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}
	*r = RequestData(aux.Alias)
	if aux.IsBinary {
		decoded, err := base64.StdEncoding.DecodeString(aux.Body)
		if err == nil {
			r.Body = decoded
		} else {
			r.Body = []byte(aux.Body)
		}
	} else {
		r.Body = []byte(aux.Body)
	}
	return nil
}

// ResponseData captures details about the outbound HTTP response.
type ResponseData struct {
	StatusCode int                 `json:"status_code"`
	Headers    map[string][]string `json:"headers"`
	Body       []byte              `json:"-"`
	Size       int64               `json:"size"`
}

// MarshalJSON provides developer-friendly JSON output for ResponseData.
func (r ResponseData) MarshalJSON() ([]byte, error) {
	type Alias ResponseData
	isBinary := !utf8.Valid(r.Body)
	var bodyStr string
	if isBinary {
		bodyStr = base64.StdEncoding.EncodeToString(r.Body)
	} else {
		bodyStr = string(r.Body)
	}

	return json.Marshal(&struct {
		Alias
		Body     string `json:"body"`
		IsBinary bool   `json:"is_binary"`
	}{
		Alias:    Alias(r),
		Body:     bodyStr,
		IsBinary: isBinary,
	})
}

// UnmarshalJSON handles deserializing ResponseData from JSON.
func (r *ResponseData) UnmarshalJSON(data []byte) error {
	type Alias ResponseData
	aux := &struct {
		Alias
		Body     string `json:"body"`
		IsBinary bool   `json:"is_binary"`
	}{}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}
	*r = ResponseData(aux.Alias)
	if aux.IsBinary {
		decoded, err := base64.StdEncoding.DecodeString(aux.Body)
		if err == nil {
			r.Body = decoded
		} else {
			r.Body = []byte(aux.Body)
		}
	} else {
		r.Body = []byte(aux.Body)
	}
	return nil
}

// HTTPTransaction represents a complete request-response cycle.
type HTTPTransaction struct {
	ID        string       `json:"id"`
	Timestamp time.Time    `json:"timestamp"`
	Duration  int64        `json:"duration"` // duration in milliseconds
	Request   RequestData  `json:"request"`
	Response  ResponseData `json:"response"`
	CreatedAt time.Time    `json:"created_at"`
}

// ToSummary converts a full transaction to a lightweight summary for listing and events.
func (t *HTTPTransaction) ToSummary() RequestSummary {
	return RequestSummary{
		ID:             t.ID,
		Timestamp:      t.Timestamp,
		Method:         t.Request.Method,
		URL:            t.Request.URL,
		Path:           t.Request.Path,
		ResponseStatus: t.Response.StatusCode,
		Duration:       t.Duration,
		RequestSize:    t.Request.Size,
		ResponseSize:   t.Response.Size,
	}
}

// RequestSummary represents summary data for the request table and real-time events.
type RequestSummary struct {
	ID             string    `json:"id"`
	Timestamp      time.Time `json:"timestamp"`
	Method         string    `json:"method"`
	URL            string    `json:"url"`
	Path           string    `json:"path"`
	ResponseStatus int       `json:"status"`
	Duration       int64     `json:"duration"` // ms
	RequestSize    int64     `json:"request_size"`
	ResponseSize   int64     `json:"response_size"`
}

// RequestFilter defines search and filter criteria.
type RequestFilter struct {
	Method string
	Status int
	Search string
	Page   int
	Limit  int
}
