package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsExcludedFromListing(t *testing.T) {
	tests := []struct {
		name   string
		status WorkOrderStatus
		want   bool
	}{
		{"RECEBIDA is not excluded", WorkOrderStatusReceived, false},
		{"EM_DIAGNOSTICO is not excluded", WorkOrderStatusInDiagnosis, false},
		{"AGUARDANDO_APROVACAO is not excluded", WorkOrderStatusWaitingApproval, false},
		{"APROVADO is not excluded", WorkOrderStatusApproved, false},
		{"EM_EXECUCAO is not excluded", WorkOrderStatusInProgress, false},
		{"FINALIZADA is excluded", WorkOrderStatusFinished, true},
		{"ENTREGUE is excluded", WorkOrderStatusDelivered, true},
		{"CANCELADA is excluded", WorkOrderStatusCanceled, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsExcludedFromListing(tt.status))
		})
	}
}

func TestWorkOrderStatusSortPriorityOf(t *testing.T) {
	tests := []struct {
		name   string
		status WorkOrderStatus
		want   int
	}{
		{"EM_EXECUCAO has highest priority (1)", WorkOrderStatusInProgress, 1},
		{"APROVADO is priority 2", WorkOrderStatusApproved, 2},
		{"AGUARDANDO_APROVACAO is priority 3", WorkOrderStatusWaitingApproval, 3},
		{"EM_DIAGNOSTICO is priority 4", WorkOrderStatusInDiagnosis, 4},
		{"RECEBIDA is priority 5", WorkOrderStatusReceived, 5},
		{"CANCELADA is priority 6", WorkOrderStatusCanceled, 6},
		{"FINALIZADA falls to default 99", WorkOrderStatusFinished, 99},
		{"ENTREGUE falls to default 99", WorkOrderStatusDelivered, 99},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, WorkOrderStatusSortPriorityOf(tt.status))
		})
	}
}

func TestWorkOrderListingExcludedStatuses_ContainsExpected(t *testing.T) {
	assert.Len(t, WorkOrderListingExcludedStatuses, 3)
	assert.Contains(t, WorkOrderListingExcludedStatuses, WorkOrderStatusFinished)
	assert.Contains(t, WorkOrderListingExcludedStatuses, WorkOrderStatusDelivered)
	assert.Contains(t, WorkOrderListingExcludedStatuses, WorkOrderStatusCanceled)
}
