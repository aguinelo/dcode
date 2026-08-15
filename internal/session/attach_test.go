package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A PNG on disk, named by the person, becomes something the model can see.
func TestAnImageIsReadFromDisk(t *testing.T) {
	dir := t.TempDir()
	png := filepath.Join(dir, "shot.png")
	// The eight-byte PNG signature is enough to be recognised.
	if err := os.WriteFile(png, []byte("\x89PNG\r\n\x1a\n....."), 0o644); err != nil {
		t.Fatal(err)
	}

	img, err := ReadImage(png, 10<<20)
	if err != nil {
		t.Fatal(err)
	}
	if img.MediaType != "image/png" {
		t.Errorf("media type is %q", img.MediaType)
	}
	if len(img.Data) == 0 {
		t.Error("no bytes came back")
	}
}

// The type is decided by what the bytes are, not by what the name claims. A
// file called shot.png that is a JPEG would be rejected by the provider with an
// error nobody could act on.
func TestTheTypeComesFromTheBytesNotTheName(t *testing.T) {
	dir := t.TempDir()
	lying := filepath.Join(dir, "shot.png")
	if err := os.WriteFile(lying, []byte("\xff\xd8\xff\xe0 jpeg really"), 0o644); err != nil {
		t.Fatal(err)
	}
	img, err := ReadImage(lying, 10<<20)
	if err != nil {
		t.Fatal(err)
	}
	if img.MediaType != "image/jpeg" {
		t.Errorf("media type is %q, want what the bytes say", img.MediaType)
	}
}

// Anything that is not a picture is refused with a reason. Sending a text file
// as an image gets a provider error the person cannot connect to what they did.
func TestSomethingThatIsNotAPictureIsRefused(t *testing.T) {
	dir := t.TempDir()
	notes := filepath.Join(dir, "notes.txt")
	if err := os.WriteFile(notes, []byte("just some text"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadImage(notes, 10<<20); err == nil {
		t.Fatal("a text file was accepted as an image")
	} else if !strings.Contains(err.Error(), "not an image") {
		t.Errorf("the refusal does not say why: %v", err)
	}
}

// The provider caps an image at ten megabytes, so refusing here is better than
// sending it and having the request rejected as a whole.
func TestAnImageTooLargeIsRefusedBeforeItIsSent(t *testing.T) {
	dir := t.TempDir()
	big := filepath.Join(dir, "big.png")
	body := append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, 2048)...)
	if err := os.WriteFile(big, body, 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := ReadImage(big, 1024)
	if err == nil {
		t.Fatal("an oversized image was accepted")
	}
	if !strings.Contains(err.Error(), "too large") {
		t.Errorf("the refusal does not say why: %v", err)
	}
}

func TestAnImageThatIsNotThereSaysSo(t *testing.T) {
	if _, err := ReadImage(filepath.Join(t.TempDir(), "absent.png"), 1<<20); err == nil {
		t.Fatal("a missing file was accepted")
	}
}
