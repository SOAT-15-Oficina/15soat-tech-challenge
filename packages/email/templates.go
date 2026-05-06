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

type BudgetServiceItem struct {
	Title       string
	Amount      string
	Estimated   string
	ApproveLink string
	RejectLink  string
}

type BudgetEmailData struct {
	CustomerName   string
	Amount         string
	BudgetLink     string
	Services       []BudgetServiceItem
	ApproveAllLink string
	RejectAllLink  string
}

func RenderBudgetEmail(data BudgetEmailData) (string, error) {
	var buf bytes.Buffer
	if err := templates.ExecuteTemplate(&buf, "budget.html", data); err != nil {
		return "", fmt.Errorf("email: render budget template: %w", err)
	}
	return buf.String(), nil
}
