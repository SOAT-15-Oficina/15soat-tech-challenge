package email

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRenderBudgetEmail_Success(t *testing.T) {
	data := BudgetEmailData{
		CustomerName: "João",
		Amount:       "R$ 150,00",
		BudgetLink:   "https://example.com/budget",
		Services: []BudgetServiceItem{
			{Title: "Troca de óleo", Amount: "R$ 80,00", Estimated: "30 min", ApproveLink: "https://approve", RejectLink: "https://reject"},
		},
		ApproveAllLink: "https://approve-all",
		RejectAllLink:  "https://reject-all",
	}

	body, err := RenderBudgetEmail(data)
	assert.NoError(t, err)
	assert.NotEmpty(t, body)
	assert.Contains(t, body, "João")
}

func TestRenderPurchaseAlertEmail_Success(t *testing.T) {
	data := PurchaseAlertEmailData{
		WorkOrderCode:  "OS-20260101-AB12",
		WorkOrderTitle: "Revisão",
		Items: []PurchaseAlertItem{
			{ServiceTitle: "Troca de óleo", SupplyTitle: "Filtro", Required: 5, InStock: 2, ToBuy: 3},
		},
	}

	body, err := RenderPurchaseAlertEmail(data)
	assert.NoError(t, err)
	assert.NotEmpty(t, body)
	assert.Contains(t, body, "OS-20260101-AB12")
}
