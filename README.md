# httperrorfmt

A Go package for formatting HTTP errors with content negotiation support.

## Overview

httperrorfmt provides formatters for HTTP errors that support multiple content types including JSON, HTML, XML, and plain text. It includes automatic content negotiation based on the client's Accept header.

## Installation

```bash
go get github.com/perbu/httperrorfmt
```

## Usage

### Basic Formatting

```go
package main

import (
    "net/http"
    "github.com/perbu/httperrorfmt"
)

// Your HTTPError implementation
type MyError struct {
    code    int
    message string
    headers map[string]string
}

func (e MyError) Error() string              { return e.message }
func (e MyError) StatusCode() int            { return e.code }
func (e MyError) Message() string            { return e.message }
func (e MyError) Headers() map[string]string { return e.headers }

func handler(w http.ResponseWriter, r *http.Request) {
    err := MyError{
        code:    http.StatusNotFound,
        message: "Resource not found",
        headers: make(map[string]string),
    }
    
    formatter := &httperrorfmt.JSONFormatter{PrettyPrint: true}
    formatter.Format(w, r, err)
}
```

### Content Negotiation

```go
func handler(w http.ResponseWriter, r *http.Request) {
    err := MyError{code: 404, message: "Not found"}
    
    formatter := httperrorfmt.NewContentNegotiatingFormatter()
    formatter.Format(w, r, err)
    // Returns JSON for Accept: application/json
    // Returns HTML for Accept: text/html
    // Returns plain text otherwise
}
```

### Individual Formatters

#### JSON Formatter

```go
formatter := &httperrorfmt.JSONFormatter{
    PrettyPrint:  true,
    IncludeStack: false,
}
formatter.Format(w, r, err)
```

#### HTML Formatter

```go
formatter := httperrorfmt.NewHTMLFormatter()
formatter.Format(w, r, err)
```

#### XML Formatter

```go
formatter := &httperrorfmt.XMLFormatter{}
formatter.Format(w, r, err)
```

#### Text Formatter

```go
formatter := &httperrorfmt.TextFormatter{}
formatter.Format(w, r, err)
```

### Custom Content Negotiation

```go
negotiator := httperrorfmt.NewContentNegotiator().
    Register("application/json", &httperrorfmt.JSONFormatter{PrettyPrint: true}).
    Register("text/html", httperrorfmt.NewHTMLFormatter()).
    Register("application/xml", &httperrorfmt.XMLFormatter{}).
    SetDefault(&httperrorfmt.TextFormatter{})

negotiator.Format(w, r, err)
```

## HTTPError Interface

Your error types must implement the HTTPError interface:

```go
type HTTPError interface {
    error
    StatusCode() int
    Message() string
    Headers() map[string]string
}
```

## License

MIT