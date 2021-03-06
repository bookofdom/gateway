package request

import (
	"encoding/json"
	"fmt"
	"net/http"

	"gateway/logreport"

	"github.com/denisenkom/go-mssqldb"
)

// ErrorResponse holds an error string.
type ErrorResponse struct {
	StatusCode int    `json:"statusCode"`
	Error      string `json:"error"`
}

// JSON converts this response to JSON format.
func (r *ErrorResponse) JSON() ([]byte, error) {
	return json.Marshal(&r)
}

// Log returns the error message
func (r *ErrorResponse) Log() string {
	return fmt.Sprintf("Error: '%s'", r.Error)
}

// NewErrorResponse returns a new response that wraps the error.
func NewErrorResponse(err error) Response {
	return &ErrorResponse{http.StatusInternalServerError, err.Error()}
}

// SQLServerErrorResponse holds an error string as well as SQL Server specific error information.
// The additional fields below are copies of those held by mssql.Error.  See https://github.com/denisenkom/go-mssqldb
// for more info.
type SQLServerErrorResponse struct {
	*ErrorResponse
	Number     int32  `json:"number,omitempty"`
	State      uint8  `json:"state,omitempty"`
	Class      uint8  `json:"class,omitempty"`
	Message    string `json:"message,omitempty"`
	ServerName string `json:"serverName,omitempty"`
	ProcName   string `json:"procName,omitempty"`
	LineNo     int32  `json:"lineNumber,omitempty"`
}

// NewSQLErrorResponse returns a new response that wraps the error.
func NewSQLErrorResponse(err error, wrapMessage string) Response {
	var errorMessage string

	if wrapMessage == "" {
		errorMessage = err.Error()
	} else {
		errorMessage = fmt.Sprintf("%s: %s", wrapMessage, err.Error())
	}

	switch t := err.(type) {
	case mssql.Error:
		logreport.Printf("Encountered a SQL error: %v\n", t)
		return &SQLServerErrorResponse{&ErrorResponse{http.StatusInternalServerError, errorMessage}, t.Number, t.State, t.Class, t.Message, t.ServerName, t.ProcName, t.LineNo}
	default:
		logreport.Printf("Encountered an error, but not a SQL error: %v\n", t)
		return &ErrorResponse{http.StatusInternalServerError, errorMessage}
	}
}
