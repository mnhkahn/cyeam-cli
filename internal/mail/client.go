package mail

import (
	"fmt"
	"mime"
	"slices"
	"strings"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-message/charset"
)

// Summary is one row in a mailbox listing.
type Summary struct {
	UID     uint32    `json:"uid"`
	From    string    `json:"from"`
	Subject string    `json:"subject"`
	Date    time.Time `json:"date"`
	Unread  bool      `json:"unread"`
}

// Client wraps an authenticated IMAP connection with INBOX selected.
type Client struct {
	c *imapclient.Client
}

// Dial connects to the account's IMAP server, logs in and selects INBOX.
func Dial(acc Account) (*Client, error) {
	user, err := acc.GetUsername()
	if err != nil {
		return nil, err
	}
	pass, err := acc.Password()
	if err != nil {
		return nil, err
	}
	opts := &imapclient.Options{
		WordDecoder: &mime.WordDecoder{CharsetReader: charset.Reader},
	}
	c, err := imapclient.DialTLS(acc.IMAPAddr(), opts)
	if err != nil {
		return nil, fmt.Errorf("connect %s: %w", acc.IMAPAddr(), err)
	}
	if err := c.Login(user, pass).Wait(); err != nil {
		c.Close()
		return nil, fmt.Errorf("login: %w", err)
	}
	if _, err := c.Select("INBOX", nil).Wait(); err != nil {
		c.Close()
		return nil, fmt.Errorf("select INBOX: %w", err)
	}
	return &Client{c: c}, nil
}

// Close logs out and closes the connection.
func (cl *Client) Close() {
	cl.c.Logout().Wait()
	cl.c.Close()
}

// ListRecent returns up to limit most-recent messages from INBOX, newest first.
func (cl *Client) ListRecent(limit int) ([]Summary, error) {
	total := cl.c.Mailbox().NumMessages
	if total == 0 {
		return nil, nil
	}
	if limit <= 0 {
		limit = 20
	}
	start := uint32(1)
	if total > uint32(limit) {
		start = total - uint32(limit) + 1
	}
	seqSet := imap.SeqSet{}
	seqSet.AddRange(start, total)

	cmd := cl.c.Fetch(seqSet, &imap.FetchOptions{
		UID:      true,
		Envelope: true,
		Flags:    true,
	})
	msgs, err := cmd.Collect()
	if err != nil {
		return nil, fmt.Errorf("fetch list: %w", err)
	}

	out := make([]Summary, 0, len(msgs))
	for _, m := range msgs {
		s := Summary{UID: uint32(m.UID), Unread: !hasSeenFlag(m.Flags)}
		if m.Envelope != nil {
			s.Subject = m.Envelope.Subject
			s.Date = m.Envelope.Date
			s.From = formatAddrs(m.Envelope.From)
		}
		out = append(out, s)
	}
	// Newest first.
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out, nil
}

// FetchRaw returns the full RFC822 source of the message with the given UID.
func (cl *Client) FetchRaw(uid uint32) ([]byte, error) {
	uidSet := imap.UIDSetNum(imap.UID(uid))
	// Peek so that reading a message does not set the server's \Seen flag —
	// the command must not change mailbox state.
	section := &imap.FetchItemBodySection{Peek: true}
	cmd := cl.c.Fetch(uidSet, &imap.FetchOptions{
		BodySection: []*imap.FetchItemBodySection{section},
	})
	msgs, err := cmd.Collect()
	if err != nil {
		return nil, fmt.Errorf("fetch body: %w", err)
	}
	if len(msgs) == 0 {
		return nil, fmt.Errorf("message UID %d not found", uid)
	}
	raw := msgs[0].FindBodySection(section)
	if raw == nil {
		return nil, fmt.Errorf("message UID %d has no body", uid)
	}
	return raw, nil
}

// MarkRead adds the \Seen flag to messages by UID.
// Returns the list of successfully updated UIDs.
func (cl *Client) MarkRead(uids []uint32) ([]uint32, error) {
	uidSet := imap.UIDSet{}
	for _, uid := range uids {
		uidSet.AddNum(imap.UID(uid))
	}
	store := &imap.StoreFlags{
		Op:     imap.StoreFlagsAdd,
		Silent: false,
		Flags:  []imap.Flag{imap.FlagSeen},
	}
	cmd := cl.c.Store(uidSet, store, nil)
	msgs, err := cmd.Collect()
	if err != nil {
		return nil, fmt.Errorf("mark read: %w", err)
	}
	result := make([]uint32, 0, len(msgs))
	for _, m := range msgs {
		result = append(result, uint32(m.UID))
	}
	return result, nil
}

// MarkUnread removes the \Seen flag from messages by UID.
// Returns the list of successfully updated UIDs.
func (cl *Client) MarkUnread(uids []uint32) ([]uint32, error) {
	uidSet := imap.UIDSet{}
	for _, uid := range uids {
		uidSet.AddNum(imap.UID(uid))
	}
	store := &imap.StoreFlags{
		Op:     imap.StoreFlagsDel,
		Silent: false,
		Flags:  []imap.Flag{imap.FlagSeen},
	}
	cmd := cl.c.Store(uidSet, store, nil)
	msgs, err := cmd.Collect()
	if err != nil {
		return nil, fmt.Errorf("mark unread: %w", err)
	}
	result := make([]uint32, 0, len(msgs))
	for _, m := range msgs {
		result = append(result, uint32(m.UID))
	}
	return result, nil
}

func hasSeenFlag(flags []imap.Flag) bool {
	return slices.Contains(flags, imap.FlagSeen)
}

func formatAddrs(addrs []imap.Address) string {
	parts := make([]string, 0, len(addrs))
	for _, a := range addrs {
		addr := a.Addr()
		if a.Name != "" {
			parts = append(parts, fmt.Sprintf("%s <%s>", a.Name, addr))
		} else {
			parts = append(parts, addr)
		}
	}
	return strings.Join(parts, ", ")
}
