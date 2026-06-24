package auth

import "testing"

func TestHashAndVerifyPassword(t *testing.T) {
	hash, err := HashPassword("secret123")
	if err != nil {
		t.Fatal(err)
	}
	ok, err := VerifyPassword(hash, "secret123")
	if err != nil || !ok {
		t.Fatalf("expected valid password, ok=%v err=%v", ok, err)
	}
	ok, err = VerifyPassword(hash, "wrong")
	if err != nil || ok {
		t.Fatalf("expected invalid password")
	}
}

func TestValidatePassword(t *testing.T) {
	t.Run("too short", func(t *testing.T) {
		if err := ValidatePassword("pass1", "admin"); err != ErrPasswordTooShort {
			t.Fatalf("expected ErrPasswordTooShort, got %v", err)
		}
	})

	t.Run("missing digit", func(t *testing.T) {
		if err := ValidatePassword("abcdefgh", "admin"); err != ErrPasswordTooWeak {
			t.Fatalf("expected ErrPasswordTooWeak, got %v", err)
		}
	})

	t.Run("missing letter", func(t *testing.T) {
		if err := ValidatePassword("12345678", "admin"); err != ErrPasswordTooWeak {
			t.Fatalf("expected ErrPasswordTooWeak, got %v", err)
		}
	})

	t.Run("same as login", func(t *testing.T) {
		if err := ValidatePassword("admin", "admin"); err != ErrPasswordTooShort {
			t.Fatalf("expected ErrPasswordTooShort for short login-like password, got %v", err)
		}
		if err := ValidatePassword("Admin123", "admin123"); err != ErrPasswordSameAsLogin {
			t.Fatalf("expected ErrPasswordSameAsLogin, got %v", err)
		}
	})

	t.Run("valid", func(t *testing.T) {
		if err := ValidatePassword("Secret123", "admin"); err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
	})
}
