package mail

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/emersion/go-message"
	gomail "github.com/emersion/go-message/mail"
)

type AppleMailSummary struct {
	ID      string    `json:"id"`
	From    string    `json:"from"`
	Subject string    `json:"subject"`
	Date    time.Time `json:"date"`
}

// OpenAppleMail opens the most recent version of the current Mac user's cache.
func OpenAppleMail() (*os.Root, error) {
	if runtime.GOOS != "darwin" {
		return nil, errors.New("Apple Mail is available only on macOS")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(home, "Library", "Mail")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("open Apple Mail cache: %w", err)
	}
	var latest string
	var version int
	for _, entry := range entries {
		number, ok := strings.CutPrefix(entry.Name(), "V")
		n, err := strconv.Atoi(number)
		if entry.IsDir() && ok && err == nil && n > version {
			version, latest = n, entry.Name()
		}
	}
	if latest == "" {
		return nil, errors.New("no Apple Mail cache found")
	}
	return os.OpenRoot(filepath.Join(dir, latest))
}

type appleMailFile struct {
	name string
	id   string
}

func walkAppleMail(root *os.Root, visit func(appleMailFile) error) error {
	foundInbox := false
	inbox := ""
	err := fs.WalkDir(root.FS(), ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if inbox != "" && !strings.HasPrefix(name, inbox+"/") {
			inbox = ""
		}
		if entry.IsDir() {
			base := strings.ToLower(entry.Name())
			if inbox == "" && (base == "inbox.mbox" || base == "收件箱.mbox") {
				foundInbox, inbox = true, name
			} else if inbox != "" && strings.HasSuffix(base, ".mbox") {
				return fs.SkipDir
			}
			return nil
		}
		if inbox != "" && entry.Type().IsRegular() && path.Base(path.Dir(name)) == "Messages" &&
			strings.HasSuffix(name, ".emlx") && !strings.HasSuffix(name, ".partial.emlx") {
			hash := sha256.Sum256([]byte(name))
			return visit(appleMailFile{name: name, id: hex.EncodeToString(hash[:])})
		}
		return nil
	})
	if err != nil {
		return err
	}
	if !foundInbox {
		return errors.New("no Apple Mail inbox cache found")
	}
	return nil
}

// readEMLX exposes the RFC822 section and leaves the trailing Apple metadata out.
func readEMLX(root *os.Root, name string, consume func(io.Reader) error) error {
	f, err := root.Open(name)
	if err != nil {
		return err
	}
	defer f.Close()
	r := bufio.NewReader(f)
	header, err := r.ReadSlice('\n')
	if err != nil {
		return fmt.Errorf("read cached message length: %w", err)
	}
	size, err := strconv.ParseInt(strings.TrimSpace(string(header)), 10, 64)
	if err != nil || size <= 0 || size > 64<<20 {
		return errors.New("invalid or oversized cached message")
	}
	info, err := f.Stat()
	if err != nil {
		return err
	}
	if size > info.Size()-int64(len(header)) {
		return io.ErrUnexpectedEOF
	}
	return consume(io.LimitReader(r, size))
}

// ListAppleMail lists locally cached inbox messages by their Date header.
func ListAppleMail(root *os.Root, limit int) ([]AppleMailSummary, error) {
	summaries := []AppleMailSummary{}
	var firstError error
	err := walkAppleMail(root, func(file appleMailFile) error {
		err := readEMLX(root, file.name, func(r io.Reader) error {
			entity, err := message.Read(r)
			if err != nil && !message.IsUnknownCharset(err) && !message.IsUnknownEncoding(err) {
				return err
			}
			header := gomail.Header{Header: entity.Header}
			subject, _ := header.Subject()
			date, _ := header.Date()
			summaries = append(summaries, AppleMailSummary{
				ID: file.id, From: addrList(header.AddressList("From")), Subject: subject, Date: date,
			})
			return nil
		})
		if err != nil && !errors.Is(err, fs.ErrNotExist) && firstError == nil {
			firstError = err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(summaries) == 0 && firstError != nil {
		return nil, fmt.Errorf("read Apple Mail cache: %w", firstError)
	}
	if limit <= 0 {
		limit = 20
	}
	slices.SortFunc(summaries, func(a, b AppleMailSummary) int {
		if order := b.Date.Compare(a.Date); order != 0 {
			return order
		}
		return strings.Compare(a.ID, b.ID)
	})
	return summaries[:min(limit, len(summaries))], nil
}

var appleMailMessageFound = errors.New("cached message found")

// ReadAppleMail reads a cached inbox message identified by ListAppleMail.
func ReadAppleMail(root *os.Root, id string) (*Message, error) {
	if _, err := hex.DecodeString(id); len(id) != sha256.Size*2 || err != nil {
		return nil, errors.New("invalid Apple Mail message ID; use an ID from mail local list")
	}
	var parsed *Message
	err := walkAppleMail(root, func(file appleMailFile) error {
		if file.id != id {
			return nil
		}
		err := readEMLX(root, file.name, func(r io.Reader) error {
			var err error
			parsed, err = ParseMessage(r)
			return err
		})
		if err != nil {
			return err
		}
		return appleMailMessageFound
	})
	if errors.Is(err, appleMailMessageFound) {
		if parsed.Body == "" {
			return nil, errors.New("message has no cached text body")
		}
		return parsed, nil
	}
	if err != nil {
		return nil, err
	}
	return nil, errors.New("cached message no longer found; run mail local list again")
}
