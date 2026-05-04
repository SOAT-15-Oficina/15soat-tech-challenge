package domain

import (
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

type CustomerDocumentType string

const (
	DocumentTypeCPF  CustomerDocumentType = "CPF"
	DocumentTypeCNPJ CustomerDocumentType = "CNPJ"
)

type Customer struct {
	ID           uuid.UUID            `json:"id"`
	Name         string               `json:"name"`
	Email        string               `json:"email"`
	Document     string               `json:"document"`
	DocumentType CustomerDocumentType `json:"document_type"`
	CreatedAt    time.Time            `json:"created_at"`
	UpdatedAt    time.Time            `json:"updated_at"`
}

// TODO: Talvez criar um DTO e validar isso na camada de handler, mas por enquanto deixo aqui para manter a simplicidade
func (c *Customer) ValidateDocument() error {
	if c.Name == "" {
		return errors.New("name is required")
	}
	if c.Email == "" {
		return errors.New("email is required")
	}

	switch c.DocumentType {
	case DocumentTypeCPF:
		return validateCPF(c.Document)
	case DocumentTypeCNPJ:
		return validateCNPJ(c.Document)
	default:
		return errors.New("invalid document type")
	}
}

var onlyNumbers = regexp.MustCompile(`\d+`)

func validateCPF(doc string) error {
	digits := strings.Join(onlyNumbers.FindAllString(doc, -1), "")

	if len(digits) != 11 {
		return errors.New("invalid CPF format")
	}

	if !isValidCPFChecksum(digits) {
		return errors.New("invalid CPF checksum")
	}
	return nil
}

func validateCNPJ(doc string) error {
	digits := strings.Join(onlyNumbers.FindAllString(doc, -1), "")

	if len(digits) != 14 {
		return errors.New("invalid CNPJ format")
	}

	if !isValidCNPJChecksum(digits) {
		return errors.New("invalid CNPJ checksum")
	}
	return nil
}

func isValidCPFChecksum(cpf string) bool {
	if cpf == strings.Repeat(string(cpf[0]), 11) {
		return false
	}

	sum := 0
	for i, c := range cpf[:9] {
		sum += int(c-'0') * (10 - i)
	}
	digit1 := 11 - (sum % 11)
	if digit1 > 9 {
		digit1 = 0
	}

	sum = 0
	for i, c := range cpf[:10] {
		sum += int(c-'0') * (11 - i)
	}
	digit2 := 11 - (sum % 11)
	if digit2 > 9 {
		digit2 = 0
	}

	return int(cpf[9]-'0') == digit1 && int(cpf[10]-'0') == digit2
}

func isValidCNPJChecksum(cnpj string) bool {
	multiplier1 := []int{5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2}
	multiplier2 := []int{6, 5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2}

	sum := 0
	for i, c := range cnpj[:12] {
		sum += int(c-'0') * multiplier1[i]
	}
	digit1 := 11 - (sum % 11)
	if digit1 > 9 {
		digit1 = 0
	}

	sum = 0
	for i, c := range cnpj[:13] {
		sum += int(c-'0') * multiplier2[i]
	}
	digit2 := 11 - (sum % 11)
	if digit2 > 9 {
		digit2 = 0
	}

	return int(cnpj[12]-'0') == digit1 && int(cnpj[13]-'0') == digit2
}
