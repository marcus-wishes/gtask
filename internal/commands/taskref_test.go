package commands

import (
	"testing"
)

func TestParseTaskRef_NumericOnly(t *testing.T) {
	ref, err := ParseTaskRef([]string{"5"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ref.HasLetter {
		t.Error("expected HasLetter to be false")
	}
	if ref.TaskNum != 5 {
		t.Errorf("expected TaskNum 5, got %d", ref.TaskNum)
	}
}

func TestParseTaskRef_CombinedRef(t *testing.T) {
	ref, err := ParseTaskRef([]string{"a1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ref.HasLetter {
		t.Error("expected HasLetter to be true")
	}
	if ref.Letter != 'a' {
		t.Errorf("expected Letter 'a', got %c", ref.Letter)
	}
	if ref.TaskNum != 1 {
		t.Errorf("expected TaskNum 1, got %d", ref.TaskNum)
	}
}

func TestParseTaskRef_CombinedRefMultiDigit(t *testing.T) {
	ref, err := ParseTaskRef([]string{"b12"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ref.HasLetter {
		t.Error("expected HasLetter to be true")
	}
	if ref.Letter != 'b' {
		t.Errorf("expected Letter 'b', got %c", ref.Letter)
	}
	if ref.TaskNum != 12 {
		t.Errorf("expected TaskNum 12, got %d", ref.TaskNum)
	}
}

func TestParseTaskRef_SeparatedRef(t *testing.T) {
	ref, err := ParseTaskRef([]string{"c", "3"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ref.HasLetter {
		t.Error("expected HasLetter to be true")
	}
	if ref.Letter != 'c' {
		t.Errorf("expected Letter 'c', got %c", ref.Letter)
	}
	if ref.TaskNum != 3 {
		t.Errorf("expected TaskNum 3, got %d", ref.TaskNum)
	}
}

func TestParseTaskRef_LetterOnly_Error(t *testing.T) {
	_, err := ParseTaskRef([]string{"a"})
	if err == nil {
		t.Fatal("expected error for letter only")
	}
	if err != ErrTaskRefRequired {
		t.Errorf("expected ErrTaskRefRequired, got %v", err)
	}
}

func TestParseTaskRef_NoArgs_Error(t *testing.T) {
	_, err := ParseTaskRef([]string{})
	if err == nil {
		t.Fatal("expected error for no args")
	}
	if err != ErrTaskRefRequired {
		t.Errorf("expected ErrTaskRefRequired, got %v", err)
	}
}

func TestParseTaskRef_InvalidRef_Error(t *testing.T) {
	_, err := ParseTaskRef([]string{"abc"})
	if err == nil {
		t.Fatal("expected error for invalid ref")
	}
	expectedMsg := "invalid task reference: abc"
	if err.Error() != expectedMsg {
		t.Errorf("expected %q, got %q", expectedMsg, err.Error())
	}
}

func TestParseTaskRef_UppercaseLetter_Error(t *testing.T) {
	// Uppercase letters are not valid list letters
	_, err := ParseTaskRef([]string{"A1"})
	if err == nil {
		t.Fatal("expected error for uppercase letter")
	}
}

func TestParseTaskRef_LastLetter(t *testing.T) {
	ref, err := ParseTaskRef([]string{"z99"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ref.HasLetter {
		t.Error("expected HasLetter to be true")
	}
	if ref.Letter != 'z' {
		t.Errorf("expected Letter 'z', got %c", ref.Letter)
	}
	if ref.TaskNum != 99 {
		t.Errorf("expected TaskNum 99, got %d", ref.TaskNum)
	}
}

func TestParseTaskRef_SeparatedWithNonDigitSecond_Error(t *testing.T) {
	_, err := ParseTaskRef([]string{"a", "xyz"})
	if err == nil {
		t.Fatal("expected error for non-digit second arg")
	}
}
