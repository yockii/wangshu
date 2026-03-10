package imageutil

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/disintegration/imaging"
)

const (
	maxImageSizeBytes = 3 * 1024 * 1024
	maxDimension      = 2048
	jpegQuality       = 85
)

func CompressImage(filePath string) ([]byte, string, error) {
	imgFile, err := os.Open(filePath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to open image file: %w", err)
	}
	defer imgFile.Close()

	img, format, err := image.Decode(imgFile)
	if err != nil {
		return nil, "", fmt.Errorf("failed to decode image: %w", err)
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	maxSide := width
	if height > maxSide {
		maxSide = height
	}

	needsResize := maxSide > maxDimension
	needsCompress := false

	if !needsResize {
		imgFile.Seek(0, 0)
		fileInfo, _ := imgFile.Stat()
		needsCompress = fileInfo.Size() > maxImageSizeBytes
	}

	if !needsResize && !needsCompress {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, "", fmt.Errorf("failed to read image file: %w", err)
		}
		return data, getMediaType(format), nil
	}

	if needsResize {
		var newWidth, newHeight int
		if width > height {
			newWidth = maxDimension
			newHeight = int(float64(maxDimension) / float64(width) * float64(height))
		} else {
			newHeight = maxDimension
			newWidth = int(float64(maxDimension) / float64(height) * float64(width))
		}

		img = imaging.Resize(img, newWidth, newHeight, imaging.Lanczos)
	}

	var buf bytes.Buffer
	mediaType := "image/jpeg"

	err = imaging.Encode(&buf, img, imaging.JPEG, imaging.JPEGQuality(jpegQuality))
	if err != nil {
		return nil, "", fmt.Errorf("failed to encode image to JPEG: %w", err)
	}

	compressedData := buf.Bytes()

	if len(compressedData) > maxImageSizeBytes {
		scaleFactor := float64(maxImageSizeBytes) / float64(len(compressedData))
		if scaleFactor < 1.0 {
			newWidth := int(float64(img.Bounds().Dx()) * scaleFactor)
			newHeight := int(float64(img.Bounds().Dy()) * scaleFactor)

			if newWidth < 100 || newHeight < 100 {
				return compressedData, mediaType, nil
			}

			resizedImg := imaging.Resize(img, newWidth, newHeight, imaging.Lanczos)
			var buf2 bytes.Buffer
			err = imaging.Encode(&buf2, resizedImg, imaging.JPEG, imaging.JPEGQuality(jpegQuality))
			if err != nil {
				return compressedData, mediaType, nil
			}
			compressedData = buf2.Bytes()
		}
	}

	return compressedData, mediaType, nil
}

func getMediaType(format string) string {
	switch strings.ToLower(format) {
	case "jpeg":
		return "image/jpeg"
	case "png":
		return "image/png"
	case "gif":
		return "image/gif"
	case "webp":
		return "image/webp"
	default:
		return "image/jpeg"
	}
}

func SaveCompressedImage(data []byte, mediaType string, outputDir string) (string, error) {
	ext := ".jpg"
	if mediaType == "image/png" {
		ext = ".png"
	}

	filename := fmt.Sprintf("compressed_%d%s", len(data), ext)
	outputPath := filepath.Join(outputDir, filename)

	err := os.WriteFile(outputPath, data, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write compressed image: %w", err)
	}

	return outputPath, nil
}

func ReadImageAsBase64(filePath string) (string, string, error) {
	data, mediaType, err := CompressImage(filePath)
	if err != nil {
		return "", "", err
	}

	return base64.StdEncoding.EncodeToString(data), mediaType, nil
}
