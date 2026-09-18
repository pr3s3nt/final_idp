package authentication

import "testing"

func TestPasswordHasher(t *testing.T) {
	hasher := PasswordHasher{}
	encoded, err := hasher.Hash("a sufficiently long password")
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}
	if !hasher.Verify("a sufficiently long password", encoded) {
		t.Fatal("Verify() rejected the password used to create the hash")
	}
	if hasher.Verify("a different long password", encoded) {
		t.Fatal("Verify() accepted the wrong password")
	}
	if hasher.NeedsRehash(encoded) {
		t.Fatal("NeedsRehash() rejected a hash created with the current policy")
	}
	if !hasher.NeedsRehash("not-an-encoded-hash") {
		t.Fatal("NeedsRehash() accepted a malformed hash")
	}
}

func TestValidatePassword(t *testing.T) {
	for _, password := range []string{"short", "password123456", "administrator123"} {
		if err := ValidatePassword(password); err == nil {
			t.Fatalf("ValidatePassword(%q) unexpectedly succeeded", password)
		}
	}
	if err := ValidatePassword("spaces are allowed here"); err != nil {
		t.Fatalf("ValidatePassword() error = %v", err)
	}
}
