package main

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
)

// loadEnvFile adds the variables of the file at path to the environment, for the
// ways this program runs without "collage dev": "collage export", which runs
// `go run . -collage-build`, and the built binary. Only "collage dev" reads an
// environment file itself, and a shell's `source .env` sets variables without
// exporting them, so without this both reach the CMS with no key and every
// request to it is a 401.
//
// A variable already set wins, so a platform's own environment, or
// `PORT=4000 ./furkanbaytekin`, still decides. A missing file is not an error:
// in production the environment usually comes from wherever the program runs.
// The format is the one "collage dev" reads: KEY=value lines, "#" comments,
// blank lines, an optional "export " prefix, and optional quotes around a value.
func loadEnvFile(path string) error {
	file, err := os.Open(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for number := 1; scanner.Scan(); number++ {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		key, value, ok := strings.Cut(line, "=")
		key = strings.TrimSpace(key)
		if !ok || key == "" {
			return fmt.Errorf("%s:%d: want KEY=value", path, number)
		}
		value = unquote(strings.TrimSpace(value))
		if _, set := os.LookupEnv(key); set {
			continue
		}
		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("%s:%d: %w", path, number, err)
		}
	}
	return scanner.Err()
}

// unquote strips one pair of matching quotes around value, or, from an unquoted
// value, a comment that follows whitespace.
func unquote(value string) string {
	if len(value) >= 2 && (value[0] == '"' || value[0] == '\'') && value[len(value)-1] == value[0] {
		return value[1 : len(value)-1]
	}
	if i := strings.Index(value, " #"); i >= 0 {
		return strings.TrimSpace(value[:i])
	}
	return value
}
