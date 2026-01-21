package main

import (
	"path"
	"strings"

	"github.com/rs/zerolog/log"

	"os"
	"path/filepath"
	"time"
)

func CronInit() {
	for {
		go CheckFiles()

		time.Sleep(5 * time.Minute)
	}
}

func CheckFiles() {
	err := filepath.Walk(Config.PublicFolder, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			log.Error().Str("source", "cron").Err(err).Msg("filepath.Walk")
			return err
		}

		if !info.IsDir() {
			isImage := CheckFileExtension(p) == "image"

			if !isImage {
				return nil
			}

			webp := strings.Replace(p, path.Ext(p), ".webp", -1)
			exists := Exists(webp)
			in, _ := os.Stat(webp)

			if !exists || in.Size() == 0 {
				file, err := EncodeWebP(p)
				if err != nil {
					log.Error().Str("source", "cron").Err(err).Msg("failed to encode file")
					return nil
				}

				if file == nil {
					return nil
				}

				log.Info().Str("source", "cron").Str("file", *file).Msg("encoded image")

				return nil
			}

			if !strings.Contains(p, "_thumb") {
				_, err := ThumbnailImage(p)
				if err != nil {
					log.Error().Str("source", "cron").Err(err).Msg("failed to encode thumbnail image")
					return err
				}

				return nil
			}
		}

		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to walk public folder")
	}
}
