package auth

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
)

// SMSSender abstracts the SMS provider so production gateways (Kavenegar,
// Twilio, ...) can be plugged in without touching the auth service.
type SMSSender interface {
	Send(ctx context.Context, phone, message string) error
}

// LogSender is the development sender: it only logs the message.
type LogSender struct {
	Logger *slog.Logger
}

func (s *LogSender) Send(_ context.Context, phone, message string) error {
	s.Logger.Info("sms (dev mode)", "phone", phone, "message", message)
	return nil
}

var (
	phoneAllowedPattern = regexp.MustCompile(`^\+[1-9][0-9]{9,14}$`)
	phoneStripPattern   = regexp.MustCompile(`[\s\-().]`)
)

var persianDigits = strings.NewReplacer(
	"۰", "0", "۱", "1", "۲", "2", "۳", "3", "۴", "4",
	"۵", "5", "۶", "6", "۷", "7", "۸", "8", "۹", "9",
	"٠", "0", "١", "1", "٢", "2", "٣", "3", "٤", "4",
	"٥", "5", "٦", "6", "٧", "7", "٨", "8", "٩", "9",
)

// NormalizePhone converts user input into E.164. Iranian local numbers
// (09xxxxxxxxx) are converted to +989xxxxxxxxx.
func NormalizePhone(raw string) (string, error) {
	phone := persianDigits.Replace(strings.TrimSpace(raw))
	phone = phoneStripPattern.ReplaceAllString(phone, "")

	switch {
	case strings.HasPrefix(phone, "00"):
		phone = "+" + phone[2:]
	case strings.HasPrefix(phone, "09") && len(phone) == 11:
		phone = "+98" + phone[1:]
	case strings.HasPrefix(phone, "9") && len(phone) == 10:
		phone = "+98" + phone
	}

	if !phoneAllowedPattern.MatchString(phone) {
		return "", fmt.Errorf("invalid phone number")
	}

	return phone, nil
}
