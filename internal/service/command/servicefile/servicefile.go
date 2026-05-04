package servicefile

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/jeffinity/otter/internal/otterfs"
)

type Source string

const (
	SourceClassic Source = "classic"
	SourcePackage Source = "package"
)

const (
	MetadataSection = "[X-Otter]"
)

type File struct {
	Name   string
	Path   string
	Source Source
}

type Finder interface {
	Find(ctx context.Context, serviceName string) (File, error)
}

type FSFinder struct {
	FS otterfs.Provider
}

func (f FSFinder) Find(ctx context.Context, serviceName string) (File, error) {
	_ = ctx

	name := NormalizeName(serviceName)
	classicPath := path.Join(f.FS.ClassicServicePath(), name+".service")
	if _, err := os.Stat(classicPath); err == nil {
		return File{Name: name, Path: classicPath, Source: SourceClassic}, nil
	} else if !os.IsNotExist(err) {
		return File{}, err
	}

	matches, err := filepath.Glob(path.Join(f.FS.PackageServicePath(), "*", "*", name+".service"))
	if err != nil || len(matches) == 0 {
		return File{}, fmt.Errorf("cannot found service %s", name)
	}
	return File{Name: name, Path: matches[0], Source: SourcePackage}, nil
}

func NormalizeName(serviceName string) string {
	return strings.TrimSuffix(serviceName, ".service")
}

func Read(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func Option(data string, name string) (string, bool, error) {
	options, exists, err := Options(data)
	if err != nil || !exists {
		return "", false, err
	}
	value, ok := options[name]
	return value, ok, nil
}

func Values(data string, name string) ([]string, bool, error) {
	scanner := bufio.NewScanner(strings.NewReader(data))
	inSection := false
	seenSection := false
	values := make([]string, 0)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if isSection(line) {
			nextInSection, err := nextSection(line, seenSection)
			if err != nil {
				return nil, true, err
			}
			inSection = nextInSection
			seenSection = seenSection || nextInSection
			continue
		}
		value, ok := optionValue(line, name, inSection)
		if ok {
			values = append(values, value)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, seenSection, err
	}
	return values, seenSection, nil
}

func Options(data string) (map[string]string, bool, error) {
	scanner := bufio.NewScanner(strings.NewReader(data))
	inSection := false
	seenSection := false
	options := map[string]string{}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if isSection(line) {
			nextInSection, err := nextSection(line, seenSection)
			if err != nil {
				return nil, true, err
			}
			inSection = nextInSection
			seenSection = seenSection || nextInSection
			continue
		}
		addOption(options, line, inSection)
	}
	if err := scanner.Err(); err != nil {
		return nil, seenSection, err
	}
	return options, seenSection, nil
}

func nextSection(line string, seen bool) (bool, error) {
	if !isMetadataSection(line) {
		return false, nil
	}
	if seen {
		return false, fmt.Errorf("only one service metadata section is allowed")
	}
	return true, nil
}

func isMetadataSection(line string) bool {
	return strings.EqualFold(line, MetadataSection)
}

func addOption(options map[string]string, line string, inSection bool) {
	key, value, ok := optionKeyValue(line, inSection)
	if !ok {
		return
	}
	if _, exists := options[key]; !exists {
		options[key] = value
	}
}

func optionValue(line string, name string, inSection bool) (string, bool) {
	key, value, ok := optionKeyValue(line, inSection)
	if !ok || key != name {
		return "", false
	}
	return value, true
}

func optionKeyValue(line string, inSection bool) (string, string, bool) {
	if !inSection || line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
		return "", "", false
	}
	key, value, ok := strings.Cut(line, "=")
	if !ok {
		return "", "", false
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return "", "", false
	}
	return key, strings.TrimSpace(value), true
}

func isSection(line string) bool {
	return strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]")
}
