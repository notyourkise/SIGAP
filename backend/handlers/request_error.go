package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"unicode/utf8"

	"github.com/gofiber/fiber/v2"
)

// bodyParseDetail adalah detail terstruktur untuk error parsing body request.
// Field diisi bila field yang bermasalah berhasil diidentifikasi.
type bodyParseDetail struct {
	Field  string `json:"field,omitempty"`
	Reason string `json:"reason"`
	Value  string `json:"value,omitempty"`
	Line   int    `json:"line,omitempty"`
	Column int    `json:"column,omitempty"`
}

// sendBodyParseError mengembalikan 400 dengan pesan yang menjelaskan field
// mana yang bermasalah dan mengapa, bukan hanya "Format input tidak valid".
//
// dst harus pointer ke request struct yang dipakai pada c.BodyParser. Nilai dst
// dipakai untuk memetakan nama field JSON ketika error berasal dari custom
// unmarshaler (mis. OptionalUUID) yang tidak membawa nama field.
func sendBodyParseError(c *fiber.Ctx, dst interface{}, err error) error {
	message, detail := describeBodyParseError(dst, err, c.Body())

	payload := fiber.Map{
		"status":  "error",
		"message": message,
	}
	if detail != nil {
		payload["detail"] = detail
	}

	return c.Status(fiber.StatusBadRequest).JSON(payload)
}

func describeBodyParseError(dst interface{}, err error, body []byte) (string, *bodyParseDetail) {
	if len(bytes.TrimSpace(body)) == 0 {
		return "Format input tidak valid: body request kosong. Kirim data dalam format JSON.", &bodyParseDetail{
			Reason: "body request kosong",
		}
	}

	// Parse menjadi map agar bisa mencari field yang gagal secara terpisah.
	var raw map[string]json.RawMessage
	if mapErr := json.Unmarshal(body, &raw); mapErr != nil {
		return formatJSONSyntaxOrTypeError(mapErr, body)
	}

	if detail := findBodyFieldError(dst, raw); detail != nil {
		message := fmt.Sprintf("Format input tidak valid pada field %q: %s", detail.Field, detail.Reason)
		if detail.Value != "" {
			message += fmt.Sprintf(" (nilai: %s)", detail.Value)
		}
		if strings.Contains(strings.ToLower(detail.Reason), "uuid") {
			message += ". Gunakan UUID yang valid atau null"
		}
		return message + ".", detail
	}

	// Fallback: error yang tidak bisa dipetakan ke field tertentu.
	return fmt.Sprintf("Format input tidak valid: %s.", humanizeJSONError(err)), nil
}

// formatJSONSyntaxOrTypeError menangani error yang terjadi sebelum field
// individual bisa diperiksa (JSON rusak atau tipe body salah).
func formatJSONSyntaxOrTypeError(err error, body []byte) (string, *bodyParseDetail) {
	var syntaxErr *json.SyntaxError
	if errors.As(err, &syntaxErr) {
		line, column := lineColumn(body, syntaxErr.Offset)
		detail := &bodyParseDetail{
			Reason: syntaxErr.Error(),
			Line:   line,
			Column: column,
		}
		return fmt.Sprintf(
			"Format input tidak valid: JSON tidak dapat dibaca pada baris %d kolom %d (%s).",
			line, column, syntaxErr.Error(),
		), detail
	}

	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) {
		expected := typeErr.Type.String()
		received := typeErr.Value
		if received == "" {
			received = "nilai tidak dikenal"
		}
		detail := &bodyParseDetail{
			Field:  typeErr.Field,
			Reason: fmt.Sprintf("tipe data tidak sesuai; diharapkan %s, diterima %s", expected, received),
		}
		if typeErr.Field != "" {
			return fmt.Sprintf(
				"Format input tidak valid pada field %q: %s.",
				typeErr.Field, detail.Reason,
			), detail
		}
		return fmt.Sprintf("Format input tidak valid: %s.", detail.Reason), detail
	}

	return fmt.Sprintf("Format input tidak valid: %s.", humanizeJSONError(err)), nil
}

// findBodyFieldError mencoba mendekode setiap field satu per satu. Ini penting
// untuk custom unmarshaler seperti OptionalUUID yang errornya tidak membawa
// nama field dari encoding/json.
func findBodyFieldError(dst interface{}, raw map[string]json.RawMessage) *bodyParseDetail {
	v := reflect.ValueOf(dst)
	if !v.IsValid() || v.Kind() != reflect.Ptr || v.IsNil() {
		return nil
	}
	elem := v.Elem()
	if elem.Kind() != reflect.Struct {
		return nil
	}

	structType := elem.Type()
	candidate := reflect.New(structType).Elem()

	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		if field.PkgPath != "" || field.Anonymous {
			continue
		}

		name := jsonFieldName(field)
		if name == "" {
			continue
		}

		rawValue, ok := lookupJSONField(raw, name)
		if !ok {
			continue
		}

		fieldValue := candidate.Field(i).Addr().Interface()
		if fieldErr := json.Unmarshal(rawValue, fieldValue); fieldErr != nil {
			return newBodyFieldDetail(name, rawValue, fieldErr)
		}
	}

	return nil
}

func jsonFieldName(field reflect.StructField) string {
	tag := field.Tag.Get("json")
	if tag == "-" {
		return ""
	}
	name := strings.Split(tag, ",")[0]
	if name == "" {
		name = field.Name
	}
	return name
}

func lookupJSONField(raw map[string]json.RawMessage, name string) (json.RawMessage, bool) {
	if value, ok := raw[name]; ok {
		return value, true
	}
	// encoding/json mencocokkan nama field secara case-insensitive.
	for key, value := range raw {
		if strings.EqualFold(key, name) {
			return value, true
		}
	}
	return nil, false
}

func newBodyFieldDetail(field string, raw json.RawMessage, err error) *bodyParseDetail {
	reason := humanizeJSONError(err)

	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) {
		received := typeErr.Value
		if received == "" {
			received = "nilai tidak dikenal"
		}
		reason = fmt.Sprintf("tipe data tidak sesuai; diharapkan %s, diterima %s", typeErr.Type.String(), received)
	}

	return &bodyParseDetail{
		Field:  field,
		Reason: reason,
		Value:  displayJSONValue(field, raw),
	}
}

func displayJSONValue(field string, raw json.RawMessage) string {
	lowerField := strings.ToLower(field)
	for _, secret := range []string{"password", "secret", "token", "api_key", "apikey", "authorization"} {
		if strings.Contains(lowerField, secret) {
			return "*** (disembunyikan)"
		}
	}

	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return ""
	}
	if bytes.Equal(trimmed, []byte("null")) {
		return "null"
	}

	// Untuk object/array, jangan tampilkan seluruh isi agar pesan tetap ringkas.
	if trimmed[0] == '{' {
		return "object"
	}
	if trimmed[0] == '[' {
		return "array"
	}

	var text string
	if err := json.Unmarshal(trimmed, &text); err == nil {
		return truncateText(text, 100)
	}
	return truncateText(string(trimmed), 100)
}

func truncateText(value string, max int) string {
	if max <= 0 || utf8.RuneCountInString(value) <= max {
		return value
	}
	runes := []rune(value)
	return string(runes[:max]) + "..."
}

func humanizeJSONError(err error) string {
	if err == nil {
		return "format request tidak dapat dibaca"
	}

	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) {
		received := typeErr.Value
		if received == "" {
			received = "nilai tidak dikenal"
		}
		return fmt.Sprintf("tipe data tidak sesuai; diharapkan %s, diterima %s", typeErr.Type.String(), received)
	}

	message := err.Error()
	switch {
	case strings.Contains(message, "unexpected end of JSON input"):
		return "JSON tidak lengkap"
	case strings.Contains(message, "invalid character"):
		return message
	default:
		return message
	}
}

// lineColumn menghitung posisi baris/kolom (1-based) dari offset byte JSON.
func lineColumn(body []byte, offset int64) (int, int) {
	if offset < 1 {
		offset = 1
	}

	line, column := 1, 1
	limit := int(offset - 1)
	if limit > len(body) {
		limit = len(body)
	}
	for i := 0; i < limit; i++ {
		if body[i] == '\n' {
			line++
			column = 1
		} else {
			column++
		}
	}
	return line, column
}
