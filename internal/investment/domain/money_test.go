package domain

import "testing"

func TestNewMoney(t *testing.T) {
	t.Parallel()
	money := NewMoney(100.50, EUR)

	if money.Amount != 100.50 {
		t.Errorf("Expected amount 100.50, got %.2f", money.Amount)
	}

	if money.Currency != EUR {
		t.Errorf("Expected currency EUR, got %v", money.Currency)
	}
}

func TestMoney_Add(t *testing.T) {
	t.Parallel()
	money1 := NewMoney(100.0, EUR)
	money2 := NewMoney(50.0, EUR)

	result := money1.Add(money2)

	if result.Amount != 150.0 {
		t.Errorf("Expected amount 150.0, got %.2f", result.Amount)
	}

	if result.Currency != EUR {
		t.Errorf("Expected currency EUR, got %v", result.Currency)
	}
}

func TestMoney_Multiply(t *testing.T) {
	t.Parallel()
	money := NewMoney(100.0, EUR)

	result := money.Multiply(2.5)

	if result.Amount != 250.0 {
		t.Errorf("Expected amount 250.0, got %.2f", result.Amount)
	}

	if result.Currency != EUR {
		t.Errorf("Expected currency EUR, got %v", result.Currency)
	}
}

func TestMoney_String(t *testing.T) {
	t.Parallel()
	money := NewMoney(123.45, EUR)

	str := money.String()
	expected := "123.45 EUR"
	if str != expected {
		t.Errorf("Expected string %s, got %s", expected, str)
	}
}

func TestCurrency_String(t *testing.T) {
	t.Parallel()
	tests := []struct {
		currency Currency
		expected string
	}{
		{EUR, "EUR"},
		{USD, "USD"},
		{GBP, "GBP"},
	}

	for _, test := range tests {
		if test.currency.String() != test.expected {
			t.Errorf("Expected currency string %s, got %s", test.expected, test.currency.String())
		}
	}
}
