package csv

import (
	"encoding/csv"
	"io"
	"strconv"
	"time"

	"github.com/benbjohnson/wtf"
)

// DialEncoder encodes dial information in CSV format to a writer.
type DialEncoder struct {
	w *csv.Writer
}

// NewDialEncoder returns a new instance of DialEncoder that writes to w.
func NewDialEncoder(w io.Writer) *DialEncoder {
	enc := &DialEncoder{w: csv.NewWriter(w)}

	// Write header to underlying writer.
	_ = enc.w.Write([]string{
		"id",
		"name",
		"value",
		"created_by",
		"created_at",
		"updated_at",
	})

	return enc
}

// Close flushes the underlying writer.
func (enc *DialEncoder) Close() error {
	enc.w.Flush()
	return enc.w.Error()
}

// EncodeDial encodes a dial row to the underlying CSV writer.
func (enc *DialEncoder) EncodeDial(dial *wtf.Dial) error {
	return enc.w.Write([]string{
		strconv.Itoa(dial.ID),
		dial.Name,
		strconv.Itoa(dial.Value),
		dial.User.Name,
		dial.CreatedAt.Format(time.RFC3339),
		dial.UpdatedAt.Format(time.RFC3339),
	})
}
