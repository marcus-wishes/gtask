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

func TestParseTaskRef_SeparatedRef_Error(t *testing.T) {
	_, err := ParseTaskRef([]string{"c", "3"})
	if err == nil {
		t.Fatal("expected error for separated ref")
	}
	expectedMsg := "invalid task reference: c"
	if err.Error() != expectedMsg {
		t.Errorf("expected %q, got %q", expectedMsg, err.Error())
	}
}

func TestParseTaskRef_LetterOnly_Error(t *testing.T) {
	_, err := ParseTaskRef([]string{"a"})
	if err == nil {
		t.Fatal("expected error for letter only")
	}
	expectedMsg := "invalid task reference: a"
	if err.Error() != expectedMsg {
		t.Errorf("expected %q, got %q", expectedMsg, err.Error())
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

func TestParseTaskRefs_Mixed(t *testing.T) {
	refs, err := ParseTaskRefs([]string{"a1", "2", "b3"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(refs) != 3 {
		t.Fatalf("expected 3 refs, got %d", len(refs))
	}

	if !refs[0].HasLetter || refs[0].Letter != 'a' || refs[0].TaskNum != 1 {
		t.Errorf("unexpected ref[0]: %#v", refs[0])
	}
	if refs[1].HasLetter || refs[1].TaskNum != 2 {
		t.Errorf("unexpected ref[1]: %#v", refs[1])
	}
	if !refs[2].HasLetter || refs[2].Letter != 'b' || refs[2].TaskNum != 3 {
		t.Errorf("unexpected ref[2]: %#v", refs[2])
	}
}

func TestParseTaskRefs_TrailingLetter_Error(t *testing.T) {
	_, err := ParseTaskRefs([]string{"a1", "b"})
	if err == nil {
		t.Fatal("expected error for trailing letter")
	}
	expectedMsg := "invalid task reference: b"
	if err.Error() != expectedMsg {
		t.Errorf("expected %q, got %q", expectedMsg, err.Error())
	}
}

func TestParseTaskRefs_InvalidToken_Error(t *testing.T) {
	_, err := ParseTaskRefs([]string{"1", "abc"})
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
	expectedMsg := "invalid task reference: abc"
	if err.Error() != expectedMsg {
		t.Errorf("expected %q, got %q", expectedMsg, err.Error())
	}
}
