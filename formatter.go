package httperrorfmt

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html/template"
	"net/http"
	"strings"
)

// HTTPError represents an HTTP error with status code and message
type HTTPError interface {
	error
	StatusCode() int
	Message() string
	Headers() map[string]string
}

// Formatter handles error formatting for different content types
type Formatter interface {
	Format(w http.ResponseWriter, r *http.Request, err HTTPError)
}

// JSONFormatter formats errors as JSON
type JSONFormatter struct {
	PrettyPrint  bool
	IncludeStack bool
}

// ErrorResponse represents a JSON error response
type ErrorResponse struct {
	Error  string `json:"error"`
	Status int    `json:"status"`
	Code   string `json:"code,omitempty"`
}

// Format implements Formatter interface for JSON responses
func (f *JSONFormatter) Format(w http.ResponseWriter, r *http.Request, err HTTPError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(err.StatusCode())

	response := ErrorResponse{
		Error:  err.Message(),
		Status: err.StatusCode(),
		Code:   http.StatusText(err.StatusCode()),
	}

	var data []byte
	if f.PrettyPrint {
		data, _ = json.MarshalIndent(response, "", "  ")
	} else {
		data, _ = json.Marshal(response)
	}

	w.Write(data)
}

// HTMLFormatter formats errors as HTML
type HTMLFormatter struct {
	Template     *template.Template
	TemplateName string
}

// DefaultHTMLTemplate is a basic error template
const DefaultHTMLTemplate = `<!DOCTYPE html>
<html>
<head>
    <title>Error {{.Status}}</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        .error-container { max-width: 600px; margin: 0 auto; }
        .error-code { font-size: 48px; color: #e74c3c; margin-bottom: 20px; }
        .error-message { font-size: 18px; color: #333; margin-bottom: 20px; }
        .error-details { font-size: 14px; color: #666; }
    </style>
</head>
<body>
    <div class="error-container">
        <div class="error-code">{{.Status}}</div>
        <div class="error-message">{{.Error}}</div>
        <div class="error-details">{{.Code}}</div>
    </div>
</body>
</html>`

// NewHTMLFormatter creates a new HTML formatter with default template
func NewHTMLFormatter() *HTMLFormatter {
	tmpl, _ := template.New("error").Parse(DefaultHTMLTemplate)
	return &HTMLFormatter{
		Template:     tmpl,
		TemplateName: "error",
	}
}

// Format implements Formatter interface for HTML responses
func (f *HTMLFormatter) Format(w http.ResponseWriter, r *http.Request, err HTTPError) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(err.StatusCode())

	data := struct {
		Error  string
		Status int
		Code   string
	}{
		Error:  err.Message(),
		Status: err.StatusCode(),
		Code:   http.StatusText(err.StatusCode()),
	}

	if f.Template != nil {
		f.Template.ExecuteTemplate(w, f.TemplateName, data)
	} else {
		// Fallback to simple HTML
		fmt.Fprintf(w, "<h1>%d %s</h1><p>%s</p>",
			err.StatusCode(), http.StatusText(err.StatusCode()), err.Message())
	}
}

// TextFormatter formats errors as plain text
type TextFormatter struct{}

// Format implements Formatter interface for plain text responses
func (f *TextFormatter) Format(w http.ResponseWriter, r *http.Request, err HTTPError) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(err.StatusCode())
	w.Write([]byte(err.Message()))
}

// ContentNegotiator allows registration of formatters for different content types
type ContentNegotiator struct {
	formatters map[string]Formatter
	defaults   Formatter
}

// NewContentNegotiator creates a new content negotiator
func NewContentNegotiator() *ContentNegotiator {
	return &ContentNegotiator{
		formatters: make(map[string]Formatter),
		defaults:   &TextFormatter{},
	}
}

// Register adds a formatter for a specific content type
func (cn *ContentNegotiator) Register(contentType string, formatter Formatter) *ContentNegotiator {
	cn.formatters[contentType] = formatter
	return cn
}

// SetDefault sets the default formatter when no content type matches
func (cn *ContentNegotiator) SetDefault(formatter Formatter) *ContentNegotiator {
	cn.defaults = formatter
	return cn
}

// Format implements Formatter interface with pluggable content negotiation
func (cn *ContentNegotiator) Format(w http.ResponseWriter, r *http.Request, err HTTPError) {
	accept := r.Header.Get("Accept")

	// Parse Accept header and find best match
	contentType := cn.parseAcceptHeader(accept)

	// Look up formatter for content type
	if formatter, exists := cn.formatters[contentType]; exists {
		formatter.Format(w, r, err)
		return
	}

	// Fall back to default formatter
	cn.defaults.Format(w, r, err)
}

// parseAcceptHeader performs simple Accept header parsing
func (cn *ContentNegotiator) parseAcceptHeader(accept string) string {
	// Handle empty Accept header
	if accept == "" {
		return "text/plain"
	}

	// Simple implementation - just look for known types
	// In order of preference
	if strings.Contains(accept, "application/json") {
		return "application/json"
	}
	if strings.Contains(accept, "text/html") {
		return "text/html"
	}
	if strings.Contains(accept, "application/xml") {
		return "application/xml"
	}
	if strings.Contains(accept, "text/plain") {
		return "text/plain"
	}

	// Default to text/plain for unknown types
	return "text/plain"
}

// ContentNegotiatingFormatter provides backward compatibility
type ContentNegotiatingFormatter struct {
	*ContentNegotiator
}

// NewContentNegotiatingFormatter creates a formatter with default content negotiation
func NewContentNegotiatingFormatter() *ContentNegotiatingFormatter {
	negotiator := NewContentNegotiator().
		Register("application/json", &JSONFormatter{PrettyPrint: true}).
		Register("text/html", NewHTMLFormatter()).
		Register("text/plain", &TextFormatter{}).
		SetDefault(&TextFormatter{})

	return &ContentNegotiatingFormatter{
		ContentNegotiator: negotiator,
	}
}

// XMLFormatter formats errors as XML
type XMLFormatter struct{}

// XMLErrorResponse represents the XML structure for error responses
type XMLErrorResponse struct {
	XMLName xml.Name `xml:"error"`
	Message string   `xml:"message"`
	Status  int      `xml:"status"`
	Code    string   `xml:"code"`
}

// Format implements Formatter interface for XML responses
func (f *XMLFormatter) Format(w http.ResponseWriter, r *http.Request, err HTTPError) {
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(err.StatusCode())

	response := XMLErrorResponse{
		Message: err.Message(),
		Status:  err.StatusCode(),
		Code:    http.StatusText(err.StatusCode()),
	}

	// Write XML declaration manually since encoding/xml doesn't include it
	w.Write([]byte(xml.Header))

	encoder := xml.NewEncoder(w)
	encoder.Indent("", "    ")
	encoder.Encode(response)
}

// DefaultFormatter is a simple formatter that negotiates content type
type DefaultFormatter struct{}

// Format implements Formatter interface with simple content negotiation
func (f *DefaultFormatter) Format(w http.ResponseWriter, r *http.Request, err HTTPError) {
	accept := r.Header.Get("Accept")

	w.WriteHeader(err.StatusCode())

	if strings.Contains(accept, "application/json") {
		w.Header().Set("Content-Type", "application/json")
		response := ErrorResponse{
			Error:  err.Message(),
			Status: err.StatusCode(),
			Code:   http.StatusText(err.StatusCode()),
		}
		data, _ := json.Marshal(response)
		w.Write(data)
	} else {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(err.Message()))
	}
}
