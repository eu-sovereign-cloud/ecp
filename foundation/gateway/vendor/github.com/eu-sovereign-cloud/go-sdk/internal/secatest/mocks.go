package secatest

import (
	"bytes"
	"net/http"
	"text/template"
)

const (
	headerContentTypeKey  = "Content-Type"
	headerContentTypeJSON = "application/json"
)

func processTemplate(w http.ResponseWriter, name string, data any) error {
	tmpl := template.Must(template.New("response").Parse(name))

	var buffer bytes.Buffer
	if err := tmpl.Execute(&buffer, data); err != nil {
		return err
	}

	_, err := w.Write(buffer.Bytes())
	if err != nil {
		return err
	}

	return nil
}

func configHttpResponse(w http.ResponseWriter, statusCode int) {
	w.Header().Set(headerContentTypeKey, headerContentTypeJSON)
	w.WriteHeader(statusCode)
}

func configGetHttpResponse(w http.ResponseWriter, template string, data any) error {
	configHttpResponse(w, http.StatusOK)

	if err := processTemplate(w, template, data); err != nil {
		return err
	}

	return nil
}

func configPutHttpResponse(w http.ResponseWriter, template string, data any) error {
	configHttpResponse(w, http.StatusOK)

	if err := processTemplate(w, template, data); err != nil {
		return err
	}

	return nil
}

func configPostHttpResponse(w http.ResponseWriter) {
	configHttpResponse(w, http.StatusAccepted)
}

func configDeleteHttpResponse(w http.ResponseWriter) {
	configHttpResponse(w, http.StatusAccepted)
}
