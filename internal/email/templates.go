package email

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"path/filepath"
)

var templatesFS embed.FS

func ParseTemplate(templateName string, data interface{}) (string, error) {
	tmplPath := filepath.Join("internal", "email", "templates", templateName)
	tmpl, err := template.ParseFiles(tmplPath)
	if err != nil {
		return "", fmt.Errorf("parse template %s: %w", templateName, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}
