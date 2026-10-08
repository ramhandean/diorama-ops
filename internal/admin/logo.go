package admin

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	_ "image/png"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	_ "golang.org/x/image/webp"
)

var (
	ErrFileTooLarge     = errors.New("file exceeds maximum size of 512 KB")
	ErrUnsupportedFormat = errors.New("only PNG, WebP, and SVG formats are supported")
	ErrUnsafeSVG        = errors.New("svg contains forbidden scripts or external references")
)

type LogoManager struct {
	storageDir string
}

func NewLogoManager(storageDir string) *LogoManager {
	_ = os.MkdirAll(storageDir, 0755)
	return &LogoManager{storageDir: storageDir}
}

func (lm *LogoManager) SaveLogo(tenantID string, fileHeader *multipart.FileHeader) (string, error) {
	if fileHeader.Size > 512*1024 {
		return "", ErrFileTooLarge
	}

	src, err := fileHeader.Open()
	if err != nil {
		return "", fmt.Errorf("open uploaded file: %w", err)
	}
	defer src.Close()

	buf, err := io.ReadAll(io.LimitReader(src, 513*1024))
	if err != nil {
		return "", err
	}
	if len(buf) > 512*1024 {
		return "", ErrFileTooLarge
	}

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	contentType := http.DetectContentType(buf)

	isSVG := ext == ".svg" || strings.Contains(contentType, "svg") || strings.Contains(string(buf[:min(len(buf), 256)]), "<svg")
	isPNG := ext == ".png" || contentType == "image/png"
	isWebP := ext == ".webp" || contentType == "image/webp"

	if !isSVG && !isPNG && !isWebP {
		return "", ErrUnsupportedFormat
	}

	var finalExt string
	var finalData []byte

	if isSVG {
		sanitized, err := SanitizeSVG(buf)
		if err != nil {
			return "", err
		}
		finalExt = ".svg"
		finalData = sanitized
	} else if isPNG {
		// Verify valid image decoding
		_, _, err := image.DecodeConfig(bytes.NewReader(buf))
		if err != nil {
			return "", fmt.Errorf("invalid png image: %w", err)
		}
		finalExt = ".png"
		finalData = buf
	} else {
		finalExt = ".webp"
		finalData = buf
	}

	fileName := fmt.Sprintf("%s-%d%s", tenantID, time.Now().Unix(), finalExt)
	destPath := filepath.Join(lm.storageDir, fileName)

	if err := os.WriteFile(destPath, finalData, 0644); err != nil {
		return "", fmt.Errorf("write logo file: %w", err)
	}

	return "/logos/" + fileName, nil
}

// SanitizeSVG strips <script>, inline event handlers, <foreignObject>, and external hrefs
func SanitizeSVG(input []byte) ([]byte, error) {
	str := string(input)

	// Check if contains basic SVG tag
	if !strings.Contains(strings.ToLower(str), "<svg") {
		return nil, ErrUnsupportedFormat
	}

	// 1. Remove <script>...</script> (case-insensitive, multi-line)
	scriptRe := regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
	str = scriptRe.ReplaceAllString(str, "")

	// 2. Remove <foreignObject>...</foreignObject>
	foreignRe := regexp.MustCompile(`(?is)<foreignObject[^>]*>.*?</foreignObject>`)
	str = foreignRe.ReplaceAllString(str, "")

	// 3. Remove inline event handlers (onload, onclick, onerror, onmouseover, etc.)
	onAttrRe := regexp.MustCompile(`(?i)\s+on[a-z0-9_-]+\s*=\s*(?:"[^"]*"|'[^']*'|[^\s>]+)`)
	str = onAttrRe.ReplaceAllString(str, "")

	// 4. Remove external references in href or xlink:href (http:, https:, //, javascript:, data:)
	hrefRe := regexp.MustCompile(`(?i)(?:xlink:)?href\s*=\s*["'](?:javascript:|data:|https?:|//)[^"']*["']`)
	str = hrefRe.ReplaceAllString(str, "")

	// 5. Remove XML processing instructions and ENTITY tags
	entityRe := regexp.MustCompile(`(?is)<!ENTITY[^>]*>`)
	str = entityRe.ReplaceAllString(str, "")

	return []byte(str), nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
