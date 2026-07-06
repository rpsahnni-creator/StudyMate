package scan

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const defaultMaxUploadBytes = 10 << 20 // 10MB

// ValidationError describes a single field validation failure.
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

var allowedBoards = map[string]struct{}{
	"ncert":           {},
	"cbse":            {},
	"icse":            {},
	"jharkhand_board": {},
	"bihar_board":     {},
	"state_board":     {}, // legacy alias
}

// NormalizeBoard maps common user input to a canonical board id.
func NormalizeBoard(board string) string {
	b := strings.ToLower(strings.TrimSpace(board))
	switch b {
	case "", "ncert":
		return "ncert"
	case "cbse":
		return "cbse"
	case "icse":
		return "icse"
	case "jharkhand", "jharkhand board", "jharkhand-board", "jharkhand_board", "jac":
		return "jharkhand_board"
	case "bihar", "bihar board", "bihar-board", "bihar_board", "bseb":
		return "bihar_board"
	case "state board", "state-board", "stateboard", "state_board":
		return "state_board"
	default:
		return b
	}
}

var scanSubjectPattern = regexp.MustCompile(`^[a-zA-Z0-9 ]*$`)

// ScanJobRequest is the validated scan job input shape.
type ScanJobRequest struct {
	Board     string
	Subject   string
	Chapter   string
	PageCount int
}

// ScanJobRequestFromCreate maps the API request into the validation struct.
func ScanJobRequestFromCreate(req CreateScanJobRequest) ScanJobRequest {
	pageCount := req.PageNo
	if pageCount <= 0 {
		pageCount = 1
	}
	board := NormalizeBoard(req.Board)
	if board == "" {
		board = "ncert"
	}
	return ScanJobRequest{
		Board:     board,
		Subject:   req.SourceText,
		Chapter:   "",
		PageCount: pageCount,
	}
}

// ValidateScanJob validates scan job metadata.
func ValidateScanJob(body ScanJobRequest) []ValidationError {
	var errs []ValidationError

	board := NormalizeBoard(body.Board)
	if board == "" {
		errs = append(errs, ValidationError{Field: "board", Message: "board is required"})
	} else if _, ok := allowedBoards[board]; !ok {
		errs = append(errs, ValidationError{Field: "board", Message: "board must be ncert, cbse, icse, jharkhand_board, or bihar_board"})
	}

	if body.Subject != "" {
		if len(body.Subject) > 100 {
			errs = append(errs, ValidationError{Field: "subject", Message: "subject must be at most 100 characters"})
		} else if !scanSubjectPattern.MatchString(body.Subject) {
			errs = append(errs, ValidationError{Field: "subject", Message: "subject must be alphanumeric with spaces"})
		}
	}

	if body.Chapter != "" && len(body.Chapter) > 200 {
		errs = append(errs, ValidationError{Field: "chapter", Message: "chapter must be at most 200 characters"})
	}

	if body.PageCount < 1 || body.PageCount > 10 {
		errs = append(errs, ValidationError{Field: "page_count", Message: "page_count must be between 1 and 10"})
	}

	return errs
}

// ValidateScanMode validates the scan product mode on create.
func ValidateScanMode(mode string) []ValidationError {
	m := strings.TrimSpace(mode)
	if m == "" {
		return nil
	}
	if m != "chapter" && m != "existing_questions" {
		return []ValidationError{{Field: "mode", Message: "mode must be chapter or existing_questions"}}
	}
	return nil
}

// MaxUploadSizeBytes returns the configured upload limit.
func MaxUploadSizeBytes() int64 {
	if mb := os.Getenv("MAX_UPLOAD_SIZE_MB"); mb != "" {
		var n int64
		if _, err := fmt.Sscan(mb, &n); err == nil && n > 0 {
			return n << 20
		}
	}
	return defaultMaxUploadBytes
}

// ValidateUpload validates multipart upload requests and returns a reader over file bytes.
func ValidateUpload(r *http.Request) (filename string, reader io.Reader, size int64, err error) {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || !strings.HasPrefix(mediaType, "multipart/") {
		return "", nil, 0, errors.New("content-type must be multipart/form-data")
	}

	maxBytes := MaxUploadSizeBytes()
	if r.ContentLength > maxBytes+1<<20 && r.ContentLength != -1 {
		return "", nil, 0, errors.New("upload exceeds maximum size")
	}

	if err := r.ParseMultipartForm(maxBytes + 1<<20); err != nil {
		return "", nil, 0, fmt.Errorf("invalid multipart upload: %w", err)
	}

	fileHeader, err := resolveUploadFile(r)
	if err != nil {
		return "", nil, 0, err
	}

	if fileHeader.Size > maxBytes {
		return "", nil, 0, errors.New("file exceeds maximum upload size")
	}

	file, err := fileHeader.Open()
	if err != nil {
		return "", nil, 0, errors.New("failed to open uploaded file")
	}
	defer file.Close()

	header := make([]byte, 512)
	n, readErr := io.ReadFull(file, header)
	if readErr != nil && !errors.Is(readErr, io.ErrUnexpectedEOF) && !errors.Is(readErr, io.EOF) {
		return "", nil, 0, errors.New("failed to read uploaded file")
	}
	header = header[:n]

	if err := validateImageMagic(header); err != nil {
		return "", nil, 0, err
	}

	safeName := sanitizeFilename(fileHeader.Filename)
	combined := io.MultiReader(bytes.NewReader(header), file)
	return safeName, combined, fileHeader.Size, nil
}

func resolveUploadFile(r *http.Request) (*multipart.FileHeader, error) {
	_, fh, err := r.FormFile("file")
	if err == nil {
		return fh, nil
	}
	_, fh, err = r.FormFile("image")
	if err == nil {
		return fh, nil
	}
	return nil, errors.New("file field is required")
}

func validateImageMagic(header []byte) error {
	if len(header) >= 3 && header[0] == 0xFF && header[1] == 0xD8 && header[2] == 0xFF {
		return nil
	}
	if len(header) >= 4 && header[0] == 0x89 && header[1] == 0x50 && header[2] == 0x4E && header[3] == 0x47 {
		return nil
	}
	if len(header) >= 12 && string(header[0:4]) == "RIFF" && string(header[8:12]) == "WEBP" {
		return nil
	}
	return errors.New("file must be JPEG, PNG, or WebP")
}

func sanitizeFilename(name string) string {
	name = filepath.Base(name)
	name = strings.Map(func(r rune) rune {
		if r == 0 {
			return -1
		}
		return r
	}, name)
	if len(name) > 255 {
		name = name[:255]
	}
	return name
}
