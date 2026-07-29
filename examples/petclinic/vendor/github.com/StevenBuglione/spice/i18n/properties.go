package i18n

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"strings"
	"unicode/utf8"
)

func readCatalogFile(
	source fs.FS,
	name string,
	remaining *int64,
) ([]byte, error) {
	info, err := fs.Stat(source, name)
	if err != nil {
		return nil, fmt.Errorf("parse message catalog %q: stat: %w", name, err)
	}
	if !info.Mode().IsRegular() || info.Size() < 0 || info.Size() > *remaining {
		return nil, fmt.Errorf(
			"parse message catalog %q: source is invalid or aggregate exceeds %d bytes",
			name,
			maxCatalogBytes,
		)
	}
	file, err := source.Open(name)
	if err != nil {
		return nil, fmt.Errorf("parse message catalog %q: open: %w", name, err)
	}
	content, readErr := io.ReadAll(io.LimitReader(file, *remaining+1))
	closeErr := file.Close()
	if readErr != nil {
		return nil, fmt.Errorf("parse message catalog %q: read: %w", name, readErr)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("parse message catalog %q: close: %w", name, closeErr)
	}
	if int64(len(content)) > *remaining || !utf8.Valid(content) {
		return nil, fmt.Errorf(
			"parse message catalog %q: content is oversized or not UTF-8",
			name,
		)
	}
	*remaining -= int64(len(content))
	return content, nil
}

func parseProperties(name, content string) (map[string]string, error) {
	values := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(content))
	scanner.Buffer(make([]byte, 4096), 64<<10)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "!") {
			continue
		}
		separator := strings.IndexByte(line, '=')
		if separator < 1 {
			return nil, fmt.Errorf(
				"parse message catalog %q line %d: expected key=value",
				name,
				lineNumber,
			)
		}
		key := strings.TrimSpace(line[:separator])
		value := strings.TrimSpace(line[separator+1:])
		if key == "" {
			return nil, fmt.Errorf(
				"parse message catalog %q line %d: key is empty",
				name,
				lineNumber,
			)
		}
		if _, duplicate := values[key]; duplicate {
			return nil, fmt.Errorf(
				"parse message catalog %q line %d: key %q is duplicated",
				name,
				lineNumber,
				key,
			)
		}
		if len(values) == maxMessageCount {
			return nil, fmt.Errorf(
				"parse message catalog %q: exceeds %d messages",
				name,
				maxMessageCount,
			)
		}
		decoded, err := decodeProperty(value)
		if err != nil {
			return nil, fmt.Errorf(
				"parse message catalog %q line %d: %w",
				name,
				lineNumber,
				err,
			)
		}
		values[key] = decoded
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("parse message catalog %q: scan: %w", name, err)
	}
	if len(values) == 0 {
		return nil, fmt.Errorf("parse message catalog %q: no messages", name)
	}
	return values, nil
}

func decodeProperty(value string) (string, error) {
	var result strings.Builder
	for index := 0; index < len(value); index++ {
		if value[index] != '\\' {
			result.WriteByte(value[index])
			continue
		}
		index++
		if index == len(value) {
			return "", errors.New("value ends with an escape")
		}
		switch value[index] {
		case 'n':
			result.WriteByte('\n')
		case 'r':
			result.WriteByte('\r')
		case 't':
			result.WriteByte('\t')
		case '\\', '=', ':', '#', '!':
			result.WriteByte(value[index])
		default:
			return "", fmt.Errorf("unsupported escape \\%c", value[index])
		}
	}
	return result.String(), nil
}
