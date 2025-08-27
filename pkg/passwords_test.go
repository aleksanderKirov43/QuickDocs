package passwords

import "testing"

func TestValidatePassword_OK(t *testing.T) {
	if err := ValidatePassword("Aa1!aaaa"); err != nil {
		t.Fatalf("expected ok, got %v", err)
	}
}

func TestValidatePassword_TooShort(t *testing.T) {
	if err := ValidatePassword("Aa1!"); err == nil {
		t.Fatalf("expected error for short password")
	}

}

func TestHashAndCompare(t *testing.T) {
	h, err := HashPassword("Aa1!aaaa")
	if err != nil {
		t.Fatalf("hash error: %v", err)
	}
	if !ComparePassword(h, "Aa1!aaaa") {
		t.Fatalf("expected passwords to match")
	}
	if ComparePassword(h, "wrongPass1!") {
		t.Fatalf("expected mismatch")
	}
}
