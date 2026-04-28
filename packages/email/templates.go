package email

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
)

//go:embed templates/*.html
var templatesFS embed.FS

var templates = template.Must(template.ParseFS(templatesFS, "templates/*.html"))

type BudgetEmailData struct {
	CustomerName string
	Amount       string
	BudgetLink   string
}

func RenderBudgetEmail(data BudgetEmailData) (string, error) {
	var buf bytes.Buffer
	if err := templates.ExecuteTemplate(&buf, "budget.html", data); err != nil {
		return "", fmt.Errorf("email: render budget template: %w", err)
	}
	return buf.String(), nil
}
