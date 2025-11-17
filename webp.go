package main

import (
	"errors"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/rs/zerolog/log"
)

func EncodeWebP(filePath string) (*string, error) {
	if _, err := os.Stat(filePath); errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	ext := path.Ext(filePath)

	if ext == ".webp" {
		return nil, nil
	}

	if ext != ".jpg" && ext != ".png" && ext != ".jpeg" {
		return nil, errors.New("not an image")
	}

	fileName := strings.Replace(filePath, ext, ".webp", -1)

	output, err := os.Create(fileName)
	if err != nil {
		log.Error().Err(err).Str("file", filePath).Msg("failed to create file")

		return nil, err
	}
	defer output.Close()

	cmd := exec.Command("cwebp", filePath, "-o", fileName)
	o, err := cmd.CombinedOutput()
	if err != nil {
		log.Error().Err(err).Str("output", string(o)).Str("file", filePath).Msg("failed to encode webp file")
	}

	return &fileName, nil
}

func DecodeWebP(filePath string) {

}
