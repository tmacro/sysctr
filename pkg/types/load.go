package types

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

func ReadSpecFromFile(path string) (*Spec, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	if strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml") {
		return loadYAML(file)
	}

	return loadJSON(file)
}

func loadJSON(f *os.File) (*Spec, error) {
	var v Spec
	decoder := json.NewDecoder(f)
	err := decoder.Decode(&v)
	if err != nil {
		return nil, err
	}

	return &v, nil
}

func loadYAML(f *os.File) (*Spec, error) {
	var v Spec
	decoder := yaml.NewDecoder(f)
	err := decoder.Decode(&v)
	if err != nil {
		return nil, err
	}

	return &v, nil
}

func HashSpec(spec *Spec) (string, error) {
	specJson, err := json.Marshal(spec)
	if err != nil {
		return "", err
	}

	h := sha256.New()
	h.Write(specJson)
	return hex.EncodeToString(h.Sum(nil)), nil
}
