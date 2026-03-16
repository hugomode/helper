package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"go.hugomode/helper/logger"
	"go.uber.org/zap"
	"moul.io/http2curl"
)

// JSONData represents the standard response structure for JSON services.
type JSONData struct {
	Data       any       `json:"data"`
	TotalPages *uint     `json:"total_pages,omitempty"`
	PageNumber *uint     `json:"page_number,omitempty"`
	PageSize   *uint     `json:"page_size,omitempty"`
	Count      *uint64   `json:"count,omitempty"`
	Errors     []*string `json:"errors,omitempty"`
}

// ResponseWithPagination creates a JSONData structure with calculated pagination metadata.
func ResponseWithPagination(data interface{}, count int64, pageSize uint, pageNumber uint) *JSONData {
	totalPage := uint(int64(count) / int64(pageSize))
	if uint(int64(count)%int64(pageSize)) != 0 {
		totalPage++
	}

	countUint64 := uint64(count)
	return &JSONData{
		Data:       data,
		TotalPages: &totalPage,
		PageNumber: &pageNumber,
		PageSize:   &pageSize,
		Count:      &countUint64,
	}
}

// HttpUtil is the primary structure for making HTTP requests with integrated logging and retries.
type HttpUtil struct {
	logger         *zap.SugaredLogger
	client         *http.Client
	header         *http.Header
	callRetry      int
	ctx            context.Context
	printCurl      bool
	authentication *authenticacionRest
}

type authenticacionRest struct {
	User string
	Pass string
}

type ResponseData struct {
	Data       []byte
	StatusCode int
}

// New creates a new HttpUtil instance with default configuration.
func New(ctx context.Context) HttpUtil {
	// Creates a new logger with the project's base configuration.
	zLog := logger.NewLoggerWithLevel("")

	client := http.Client{}
	defaultHeaders := http.Header{
		"Content-Type": []string{"application/json"},
	}

	return HttpUtil{
		logger:    zLog.Sugar(),
		client:    &client,
		header:    &defaultHeaders,
		callRetry: 1,
		ctx:       ctx,
	}
}

// SetLevel sets the log level for this HttpUtil instance.
func (l *HttpUtil) SetLevel(level string) {
	if level == "" {
		return
	}

	// Obtains a new logger instance with the desired level, maintaining the project's configuration.
	newLog := logger.NewLoggerWithLevel(level)
	l.logger = newLog.Sugar()
}

// SetCallRetry sets the number of retry attempts for failed requests.
func (l *HttpUtil) SetCallRetry(callRetry int) {
	l.callRetry = callRetry
}

// SetHeader sets the request headers.
func (l *HttpUtil) SetHeader(header http.Header) {
	l.header = &header
}

// AddHeader appends headers to the existing request headers.
func (l *HttpUtil) AddHeader(header http.Header) {
	if l.header == nil {
		l.header = &http.Header{}
	}
	for k, v := range header {
		for _, val := range v {
			l.header.Add(k, val)
		}
	}
}

// SetClient establece el cliente HTTP
func (l *HttpUtil) SetClient(client *http.Client) {
	l.client = client
}

// SetPrintCurl enables or disables printing requests in CURL format to the log.
func (l *HttpUtil) SetPrintCurl(arg bool) {
	l.printCurl = arg
}

// SetBasicAuthenticacion sets the basic authentication credentials for the HTTP client.
func (l *HttpUtil) SetBasicAuthenticacion(user, pass string) {
	l.authentication = &authenticacionRest{User: user, Pass: pass}
}

// GetRest performs an HTTP GET request.
func (l *HttpUtil) GetRest(url string, timeout time.Duration) (ResponseData, error) {
	l.logger.Debug("-----------------GetRest--------------------------")
	return l.rest(url, http.MethodGet, nil, timeout)
}

// PostRest performs an HTTP POST request with a JSON body.
func (l *HttpUtil) PostRest(url string, body interface{}, timeout time.Duration) (ResponseData, error) {
	l.logger.Debug("-----------------PostRest--------------------------")
	return l.rest(url, http.MethodPost, body, timeout)
}

// PutRest performs an HTTP PUT request.
func (l *HttpUtil) PutRest(url string, body interface{}, timeout time.Duration) (ResponseData, error) {
	l.logger.Debug("-----------------PutRest--------------------------")
	return l.rest(url, http.MethodPut, body, timeout)
}

// PatchRest performs an HTTP PATCH request.
func (l *HttpUtil) PatchRest(url string, body interface{}, timeout time.Duration) (ResponseData, error) {
	l.logger.Debug("-----------------PatchRest--------------------------")
	return l.rest(url, http.MethodPatch, body, timeout)
}

func (l *HttpUtil) rest(url string, method string, body interface{}, timeout time.Duration) (ResponseData, error) {
	t := time.Now()

	var response *http.Response
	var err error

	defer func() {
		if response != nil && response.Body != nil {
			response.Body.Close()
		}
		l.logger.Debugf("---REST--- Duration in milliseconds: %d", time.Since(t).Milliseconds())
	}()

	bodyR, err := prepareRequestBody(body, l.logger)
	if err != nil {
		return ResponseData{nil, http.StatusInternalServerError}, err
	}

	request, err := http.NewRequestWithContext(l.ctx, method, url, bodyR)
	if err != nil {
		l.logger.Errorf("<<<ERROR url: %s method: %s could not create 'NewRequest': %v", url, method, err)
		return ResponseData{nil, http.StatusInternalServerError}, err
	}

	if l.header != nil {
		request.Header = *l.header
	}

	if l.authentication != nil {
		request.SetBasicAuth(l.authentication.User, l.authentication.Pass)
	}

	l.client.Timeout = timeout

	if l.printCurl {
		printCurlCommand(request, l.logger)
	}

	for i := 0; i < l.callRetry; i++ {
		response, err = executeRequest(l.client, request, l.ctx, l.logger, method, url, l.callRetry, i)
		if err == nil {
			break
		}
	}

	if err != nil {
		l.logger.Errorf("<<<ERROR Request: %v", err)
		return ResponseData{nil, http.StatusInternalServerError}, err
	}

	data, err := readResponseBody(response, l.logger)
	if err != nil {
		return ResponseData{nil, response.StatusCode}, err
	}

	l.logger.Debugf("+++REST+++ service response: %s", string(data))
	return ResponseData{data, response.StatusCode}, nil
}

func prepareRequestBody(body interface{}, log *zap.SugaredLogger) (io.Reader, error) {
	if body == nil {
		return nil, nil
	}
	if reader, ok := body.(io.Reader); ok {
		return reader, nil
	}
	aux, err := json.Marshal(body)
	if err != nil {
		log.Errorf("<<<ERROR encoding json to bytes. ---> %s", err.Error())
		return nil, err
	}
	return bytes.NewReader(aux), nil
}

func printCurlCommand(request *http.Request, log *zap.SugaredLogger) {
	command, _ := http2curl.GetCurlCommand(request)
	log.Infof("## CURL--> %s", command)
}

func executeRequest(client *http.Client, request *http.Request, ctx context.Context, log *zap.SugaredLogger, method string, url string, callRetry int, attempt int) (*http.Response, error) {
	response, err := client.Do(request)
	if err != nil {
		if !errors.Is(err, context.Canceled) && attempt < callRetry-1 {
			log.Warnf("<<<ERROR [%v] performing %s to url: %s with remaining attempts: %d", err, method, url, callRetry-attempt-1)
			time.Sleep(2 * time.Second)
		}
	}
	return response, err
}

func readResponseBody(response *http.Response, log *zap.SugaredLogger) ([]byte, error) {
	if response.Body == nil {
		return nil, nil
	}
	data, err := io.ReadAll(response.Body)
	if err != nil {
		log.Errorf("<<<ERROR Reading data: %v", err)
		return nil, err
	}
	return data, nil
}
