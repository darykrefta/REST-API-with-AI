package uploads

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

// MaxImageBytes is the maximum decoded multipart size for a single upload.
const MaxImageBytes = 10 << 20 // 10 MiB

// FormField is the multipart form field name clients should use for the file.
const FormField = "image"

var sniffToExt = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/gif":  ".gif",
	"image/webp": ".webp",
}

// EnsureDir creates the directory if it does not exist.
func EnsureDir(dir string) error {
	return os.MkdirAll(dir, 0o755)
}

// SaveImageIfPresent parses multipart (if needed), and if field "image" is present saves it under destDir.
// Returns the stored basename (no path), or ("", nil) when the field is absent.
func SaveImageIfPresent(r *http.Request, destDir string) (basename string, err error) {
	if r.MultipartForm == nil {
		if err := r.ParseMultipartForm(MaxImageBytes); err != nil {
			return "", fmt.Errorf("parse multipart: %w", err)
		}
	}
	f, hdr, err := r.FormFile(FormField)
	if err != nil {
		if errors.Is(err, http.ErrMissingFile) {
			return "", nil
		}
		return "", err
	}
	defer f.Close()
	return saveOpenedImage(f, hdr, destDir)
}

func saveOpenedImage(f multipart.File, hdr *multipart.FileHeader, destDir string) (string, error) {
	if hdr.Size > 0 && hdr.Size > MaxImageBytes {
		return "", errors.New("file too large")
	}

	head := make([]byte, 512)
	n, err := io.ReadFull(f, head)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		return "", err
	}
	if n == 0 {
		return "", errors.New("empty file")
	}
	kind := http.DetectContentType(head[:n])
	ext, ok := sniffToExt[kind]
	if !ok {
		return "", fmt.Errorf("unsupported image type %q", kind)
	}

	name := randomName() + ext
	path := filepath.Join(destDir, name)
	out, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer out.Close()

	body := io.MultiReader(bytes.NewReader(head[:n]), f)
	written, err := io.Copy(out, io.LimitReader(body, MaxImageBytes+1))
	if err != nil {
		_ = os.Remove(path)
		return "", err
	}
	if written > MaxImageBytes {
		_ = os.Remove(path)
		return "", errors.New("file too large")
	}

	return name, nil
}

func randomName() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic("uploads: crypto/rand: " + err.Error())
	}
	return hex.EncodeToString(b)
}
