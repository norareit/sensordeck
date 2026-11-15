// Package uploadstate is the state file of the uploads: which theme file each screen holds, by the serial of the
// screen and the hash of the file. It is not limited to a platform.
package uploadstate

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
)

// Hash gives the SHA-256 of the theme file as 64 lower-case hexadecimal digits.
func Hash(file []byte) string {
	sum := sha256.Sum256(file)
	return hex.EncodeToString(sum[:])
}

// Path gives the state file in the cache directory of the user.
func Path(cacheDirectory string) string {
	return filepath.Join(cacheDirectory, "sensordeck", "uploaded-themes.json")
}

// Decode reads a JSON object whose keys are serials and whose values are hashes. Anything else is an error.
func Decode(content []byte) (map[string]string, error) {
	// The decoder reads null into a map with no error, so the first byte is checked here.
	if trimmed := bytes.TrimSpace(content); len(trimmed) == 0 || trimmed[0] != '{' {
		return nil, errors.New("not a JSON object of serials and hashes")
	}
	// A null value decodes into a string with no error, so the values are read as pointers.
	var decoded map[string]*string
	if err := json.Unmarshal(content, &decoded); err != nil {
		return nil, err
	}
	entries := make(map[string]string, len(decoded))
	for serial, hash := range decoded {
		if hash == nil {
			return nil, errors.New("the hash of " + strconv.Quote(serial) + " is null")
		}
		entries[serial] = *hash
	}
	return entries, nil
}

// Encode gives the object with its keys sorted and a final line feed.
func Encode(entries map[string]string) []byte {
	// Marshalling a map of strings cannot fail.
	content, _ := json.MarshalIndent(entries, "", "  ")
	return append(content, '\n')
}

// Read gives the entries of the file. A file that does not exist holds none and is no error. A file that cannot be
// read or decoded gives no entry and the error.
func Read(path string) (map[string]string, error) {
	content, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return map[string]string{}, nil
	}
	if err != nil {
		return map[string]string{}, err
	}
	entries, err := Decode(content)
	if err != nil {
		return map[string]string{}, err
	}
	return entries, nil
}

// Write creates the directory of the file, writes the entries to a temporary file in it and renames that over the
// file. On a failure the temporary file is removed.
func Write(path string, entries map[string]string) error {
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(directory, ".uploaded-themes-*.json")
	if err != nil {
		return err
	}
	_, err = temporary.Write(Encode(entries))
	if closeErr := temporary.Close(); err == nil {
		err = closeErr
	}
	if err == nil {
		err = os.Rename(temporary.Name(), path)
	}
	if err != nil {
		_ = os.Remove(temporary.Name())
	}
	return err
}
