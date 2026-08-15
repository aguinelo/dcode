package session

import (
	"bytes"
	"fmt"
	"net/http"
	"os"

	ce "github.com/aguinelo/dcode/internal/contextengine"
)

// ReadImage loads a picture the person named, ready to be shown to the model.
//
// The media type comes from the BYTES, never from the extension. A file called
// shot.png that is really a JPEG is common — every screenshot tool and every
// chat app renames things — and trusting the name would produce a provider
// error about a mismatch nobody can connect to what they did.
//
// Refusing here rather than at the provider is the whole point of the function.
// A rejected request tells the person their turn failed; this tells them the
// file is not a picture, or is too big, while they can still do something about
// it.
func ReadImage(path string, limit int64) (ce.Image, error) {
	info, err := os.Stat(path)
	if err != nil {
		return ce.Image{}, err
	}
	if limit > 0 && info.Size() > limit {
		return ce.Image{}, fmt.Errorf("%s is too large: %d bytes, and the limit is %d",
			path, info.Size(), limit)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return ce.Image{}, err
	}

	// DetectContentType reads the first 512 bytes, which is where every format
	// this accepts puts its signature.
	head := body
	if len(head) > 512 {
		head = head[:512]
	}
	ct := http.DetectContentType(head)
	if i := bytes.IndexByte([]byte(ct), ';'); i >= 0 {
		ct = ct[:i]
	}
	if !supportedImage(ct) {
		return ce.Image{}, fmt.Errorf("%s is not an image the model can read: it is %s", path, ct)
	}
	return ce.Image{MediaType: ct, Data: body}, nil
}

// supportedImage is what the providers take, and nothing else.
//
// A list rather than "anything starting with image/", because a TIFF or an SVG
// would be accepted here and refused on the wire — a failure moved later
// without being made less likely.
func supportedImage(ct string) bool {
	switch ct {
	case "image/png", "image/jpeg", "image/gif", "image/webp":
		return true
	}
	return false
}
