package formats

import (
	"context"
	"fmt"
	"net/netip"
	"strconv"
	"strings"

	"github.com/fredbi/core/strfmt"
	"golang.org/x/net/idna"
)

var _ strfmt.Format = NewHostname()

var idnaHostChecker = idna.New(
	idna.ValidateForRegistration(), // shorthand for [idna.StrictDomainName],  [idna.ValidateLabels], [idna.VerifyDNSLength], [idna.BidiRule]
)

var zeroip = netip.Addr{}

// IsHostname returns true when the string is a valid hostname.
//
// It follows the rules detailed at https://url.spec.whatwg.org/#concept-host-parser
// and implemented by most modern web browsers.
//
// It supports IDNA rules regarding internationalized names with unicode.
//
// Besides:
// * the empty string is not a valid host name
// * a trailing dot is allowed in names and IPv4's (not IPv6)
// * a host name can be a valid IPv4 (with decimal, octal or hexadecimal numbers) or IPv6 address
// * IPv6 zones are disallowed
// * top-level domains can be unicode (cf. https://www.iana.org/domains/root/db).
//
// NOTE: this validator doesn't check top-level domains against the IANA root database.
// It merely ensures that a top-level domain in a FQDN is at least 2 code points long.
func IsHostname(str string) bool {
	_, ok := isHostname(str)

	return ok
}

func isHostname(str string) (ip netip.Addr, ok bool) {
	if len(str) == 0 {
		return ip, false
	}

	// IP v6 check
	if ipv6Cleaned, found := strings.CutPrefix(str, "["); found {
		ipv6Cleaned, found = strings.CutSuffix(ipv6Cleaned, "]")
		if !found {
			return ip, false
		}

		addr, ok := isValidIPv6(ipv6Cleaned)

		return addr, ok
	}

	// IDNA check
	res, err := idnaHostChecker.ToASCII(strings.ToLower(str))
	if err != nil || res == "" {
		return ip, false
	}

	parts := strings.Split(res, ".")

	// IP v4 check
	lastPart, lastIndex, shouldBeIPv4 := domainEndsAsNumber(parts)
	if shouldBeIPv4 {
		// domain ends in a number: must be an IPv4
		addrAsBytes, ok := isValidIPv4(parts[:lastIndex+1]) // if the last part is a trailing dot, remove it

		return netip.AddrFrom4(addrAsBytes), ok
	}

	// check TLD length (excluding trailing dot)
	const minTLDLength = 2
	if lastIndex > 0 && len(lastPart) < minTLDLength {
		return ip, false
	}

	return ip, true
}

// domainEndsAsNumber determines if a domain name ends with a decimal, octal or hex digit,
// accounting for a possible trailing dot (the last part being empty in that case).
//
// It returns the last non-trailing dot part and if that part consists only of (dec/hex/oct) digits.
func domainEndsAsNumber(parts []string) (lastPart string, lastIndex int, ok bool) {
	// NOTE: using ParseUint(x, 0, 32) is not an option, as the IPv4 format supported why WHATWG
	// doesn't support notations such as "0b1001" (binary digits) or "0o666" (alternate notation for octal digits).
	lastIndex = len(parts) - 1
	lastPart = parts[lastIndex]
	if len(lastPart) == 0 {
		// trailing dot
		if len(parts) == 1 { // dot-only string: normally already ruled out by the IDNA check above
			return lastPart, lastIndex, false
		}

		lastIndex--
		lastPart = parts[lastIndex]
	}

	if startOfHexDigit(lastPart) {
		for _, b := range []byte(lastPart[2:]) {
			if !isHexDigit(b) {
				return lastPart, lastIndex, false
			}
		}

		return lastPart, lastIndex, true
	}

	// check for decimal and octal
	for _, b := range []byte(lastPart) {
		if !isASCIIDigit(b) {
			return lastPart, lastIndex, false
		}
	}

	return lastPart, lastIndex, true
}

func startOfHexDigit(str string) bool {
	return strings.HasPrefix(str, "0x") // the input has already been lower-cased
}

func startOfOctalDigit(str string) bool {
	if str == "0" {
		// a single "0" is considered decimal
		return false
	}

	return strings.HasPrefix(str, "0")
}

func isValidIPv6(str string) (ip netip.Addr, ok bool) {
	// disallow empty ipv6 address
	if len(str) == 0 {
		return ip, false
	}

	addr, err := netip.ParseAddr(str)
	if err != nil {
		return ip, false
	}

	if !addr.Is6() {
		return ip, false
	}

	// explicit desupport of IPv6 zones
	if addr.Zone() != "" {
		return ip, false
	}

	return addr, true
}

// isValidIPv4 parses an IPv4 with decimal, hex or octal digit parts.
//
// We can't rely on [netip.ParseAddr] because we may get a mix of decimal, octal and hex digits.
//
// Examples of valid addresses not supported by [netip.ParseAddr] or [net.ParseIP]:
//
//	"192.0x00A80001"
//	"0300.0250.0340.001"
//	"1.0x.1.1"
//
// But not:
//
//	"0b1010.2.3.4"
//	"0o07.2.3.4"
func isValidIPv4(parts []string) (ip [4]byte, ok bool) {
	// NOTE: using ParseUint(x, 0, 32) is not an option, even though it would simplify this code a lot.
	// The IPv4 format supported why WHATWG doesn't support notations such as "0b1001" (binary digits)
	// or "0o666" (alternate notation for octal digits).
	const (
		maxPartsInIPv4  = 4
		maxDigitsInPart = 11 // max size of a 4-bytes hex or octal digit
	)

	if len(parts) == 0 || len(parts) > maxPartsInIPv4 {
		return ip, false
	}

	// we call this when we know that the last part is a digit part, so len(lastPart)>0

	digits := make([]uint64, 0, maxPartsInIPv4)
	for _, part := range parts {
		if len(part) == 0 { // empty part: this case has normally been already ruled out by the IDNA check above
			return ip, false
		}

		if len(part) > maxDigitsInPart { // whether decimal, octal or hex, an address can't exceed that length
			return ip, false
		}

		if !isASCIIDigit(part[0]) { // start of an IPv4 part is always a digit
			return ip, false
		}

		switch {
		case startOfHexDigit(part):
			const hexDigitOffset = 2
			hexString := part[hexDigitOffset:]
			if len(hexString) == 0 { // 0x part: assume 0
				digits = append(digits, 0)

				continue
			}

			hexDigit, err := strconv.ParseUint(hexString, 16, 32)
			if err != nil {
				return ip, false
			}

			digits = append(digits, hexDigit)

			continue

		case startOfOctalDigit(part):
			const octDigitOffset = 1
			octString := part[octDigitOffset:] // we know that this is not empty
			octDigit, err := strconv.ParseUint(octString, 8, 32)
			if err != nil {
				return ip, false
			}

			digits = append(digits, octDigit)

		default: // assume decimal digits (0-255)
			// we know that we don't have a leading 0 (would have been caught by octal digit)
			decDigit, err := strconv.ParseUint(part, 10, 8)
			if err != nil {
				return ip, false
			}

			digits = append(digits, decDigit)
		}
	}

	// now check the digits: the last digit may encompass several parts of the address
	lastDigit := digits[len(digits)-1]
	if lastDigit > uint64(1)<<uint64(8*(maxPartsInIPv4+1-len(digits))) { //nolint:gosec,mnd // 256^(5 - len(digits)) - safe conversion
		return ip, false
	}
	const maxUint8 = uint64(^uint8(0))
	if lastDigit > maxUint8 {
		shifted := lastDigit
		for i := maxPartsInIPv4 + 1 - len(digits); i >= 0; i-- {
			shifted = lastDigit << uint64(8*i) & uint64(0xff))
		}

		// TODO: split last digit if necessary
	} else {
		ip[len(digits)-1] = byte(lastDigit)
	}

	if len(digits) > 1 {
		for i := 0; i < len(digits)-2; i++ {
			if digits[i] > maxUint8 {
				return ip, false
			}
			ip[i] = byte(digits[i])
		}
	}

	return ip, true
}

func isHexDigit(c byte) bool {
	switch {
	case '0' <= c && c <= '9':
		return true
	case 'a' <= c && c <= 'f': // assume the input string to be lower case
		return true
	}
	return false
}

func isASCIIDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

type Hostname struct {
	ip   netip.Addr
	host string
}

func NewHostname() *Hostname {
	h := MakeHostname()

	return &h
}

func MakeHostname() Hostname {
	return Hostname{}
}

func (h Hostname) String() string {
	return h.host
}

func (h Hostname) IsIPv4() bool {
	return h.ip != zeroip && h.ip.Is4()
}

func (h Hostname) IsIPv6() bool {
	return h.ip != zeroip && h.ip.Is6()
}

func (h Hostname) IP() netip.Addr {
	return h.ip
}

func (h Hostname) ToASCII() string {
	punycode, _ := idnaHostChecker.ToASCII(h.host)

	return punycode
}

func (h Hostname) MarshalText() ([]byte, error) {
	return []byte(h.host), nil
}

func (h *Hostname) UnmarshalText(data []byte) error {
	host := string(data)
	addr, ok := isHostname(host)

	if !ok {
		return fmt.Errorf("invalid hostname: %w", ErrFormat)
	}

	h.ip = addr
	h.host, _ = idnaHostChecker.ToUnicode(host)

	return nil
}

func (h Hostname) Validate(_ context.Context) error {
	if !IsHostname(string(h.host)) {
		return fmt.Errorf("invalid hostname: %w", ErrFormat)
	}

	return nil
}
