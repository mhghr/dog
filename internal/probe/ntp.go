package probe

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"net"
	"strconv"
	"time"

	"monitoring-platform/internal/domain"
)

const (
	ntpPacketSize   = 48
	ntpEpochOffset  = 2208988800 // seconds between 1900-01-01 and 1970-01-01
	ntpClientMode   = 3
	ntpServerMode   = 4
	ntpLeapUnsynced = 3
)

type NTPExecutor struct {
	deps Deps
}

func NewNTPExecutor(deps Deps) *NTPExecutor {
	return &NTPExecutor{deps: deps}
}

func (e *NTPExecutor) Type() domain.MonitorType {
	return domain.MonitorNTP
}

// Execute sends a standard NTP client request over UDP/123 and validates the
// response: mode, stratum, leap indicator, clock offset, and round trip.
func (e *NTPExecutor) Execute(ctx context.Context, job domain.ProbeJob) domain.ProbeResult {
	result := newBaseResult(job)

	port := intConfig(job.Config, "port", 123)
	version := intConfig(job.Config, "version", 4)
	if version != 3 && version != 4 {
		version = 4
	}

	address := net.JoinHostPort(job.Target, strconv.Itoa(port))

	conn, err := e.deps.Guard.DialContext(ctx, "udp", address)
	if err != nil {
		return finishFailure(result, "ntp_timeout", err)
	}
	defer conn.Close()

	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}

	request := make([]byte, ntpPacketSize)
	request[0] = byte(version<<3 | ntpClientMode)

	sendTime := time.Now()
	transmitTimestamp := toNTPTimestamp(sendTime)
	binary.BigEndian.PutUint64(request[40:48], transmitTimestamp)

	if _, err := conn.Write(request); err != nil {
		return finishFailure(result, "ntp_timeout", err)
	}

	response := make([]byte, ntpPacketSize)
	if _, err := readFullPacket(conn, response); err != nil {
		return finishFailure(result, "ntp_timeout", err)
	}
	receiveTime := time.Now()

	leap := int(response[0] >> 6)
	mode := int(response[0] & 0x07)
	stratum := int(response[1])

	if mode != ntpServerMode {
		return finishFailure(result, "ntp_invalid_response", fmt.Errorf("unexpected NTP mode %d", mode))
	}

	originTimestamp := binary.BigEndian.Uint64(response[24:32])
	if originTimestamp != transmitTimestamp {
		return finishFailure(result, "ntp_invalid_response", fmt.Errorf("origin timestamp does not match request"))
	}

	referenceID := response[12:16]

	if stratum == 0 {
		kissCode := string(referenceID)
		return finishFailure(result, "ntp_kiss_of_death", fmt.Errorf("received kiss-of-death packet %q", kissCode))
	}

	if leap == ntpLeapUnsynced {
		return finishFailure(result, "ntp_unsynchronized", fmt.Errorf("server clock is not synchronized"))
	}

	serverReceive := fromNTPTimestamp(binary.BigEndian.Uint64(response[32:40]))
	serverTransmit := fromNTPTimestamp(binary.BigEndian.Uint64(response[40:48]))

	// RFC 5905 clock offset and round-trip delay.
	offset := (serverReceive.Sub(sendTime) + serverTransmit.Sub(receiveTime)) / 2
	roundTrip := receiveTime.Sub(sendTime) - serverTransmit.Sub(serverReceive)
	if roundTrip < 0 {
		roundTrip = 0
	}

	rootDelay := ntpShortToDuration(binary.BigEndian.Uint32(response[4:8]))
	rootDispersion := ntpShortToDuration(binary.BigEndian.Uint32(response[8:12]))

	offsetMillis := float64(offset.Microseconds()) / 1000
	roundTripMillis := float64(roundTrip.Microseconds()) / 1000

	result.Metrics["offset_ms"] = offsetMillis
	result.Metrics["round_trip_ms"] = roundTripMillis
	result.Metrics["stratum"] = stratum
	result.Metrics["root_delay_ms"] = float64(rootDelay.Microseconds()) / 1000
	result.Metrics["root_dispersion_ms"] = float64(rootDispersion.Microseconds()) / 1000
	result.Metrics["response_valid"] = 1

	result.Attributes["leap_indicator"] = leap
	result.Attributes["version"] = int(response[0] >> 3 & 0x07)
	result.Attributes["mode"] = mode
	result.Attributes["stratum"] = stratum
	result.Attributes["reference_id"] = formatReferenceID(stratum, referenceID)
	result.Attributes["server_time"] = serverTransmit.UTC()

	minStratum := intConfig(job.Config, "allowed_stratum_min", 1)
	maxStratum := intConfig(job.Config, "allowed_stratum_max", 15)
	if stratum < minStratum || stratum > maxStratum {
		return finishFailure(
			result,
			"ntp_stratum_invalid",
			fmt.Errorf("stratum %d outside allowed range [%d, %d]", stratum, minStratum, maxStratum),
		)
	}

	maxOffset := float64(intConfig(job.Config, "max_offset_millis", 1000))
	if math.Abs(offsetMillis) > maxOffset {
		return finishFailure(
			result,
			"ntp_offset_too_high",
			fmt.Errorf("clock offset %.2fms exceeds maximum %.0fms", offsetMillis, maxOffset),
		)
	}

	maxRoundTrip := float64(intConfig(job.Config, "max_round_trip_millis", 2000))
	if roundTripMillis > maxRoundTrip {
		return finishFailure(
			result,
			"ntp_round_trip_too_high",
			fmt.Errorf("round trip %.2fms exceeds maximum %.0fms", roundTripMillis, maxRoundTrip),
		)
	}

	return finishSuccess(result)
}

func readFullPacket(conn net.Conn, buffer []byte) (int, error) {
	read, err := conn.Read(buffer)
	if err != nil {
		return 0, err
	}
	if read < ntpPacketSize {
		return read, fmt.Errorf("short NTP response: %d bytes", read)
	}
	return read, nil
}

func toNTPTimestamp(t time.Time) uint64 {
	seconds := uint64(t.Unix()) + ntpEpochOffset
	fraction := uint64(float64(t.Nanosecond()) / 1e9 * (1 << 32))
	return seconds<<32 | fraction
}

func fromNTPTimestamp(timestamp uint64) time.Time {
	seconds := int64(timestamp>>32) - ntpEpochOffset
	fraction := timestamp & 0xFFFFFFFF
	nanoseconds := int64(float64(fraction) / (1 << 32) * 1e9)
	return time.Unix(seconds, nanoseconds)
}

func ntpShortToDuration(value uint32) time.Duration {
	seconds := value >> 16
	fraction := value & 0xFFFF
	return time.Duration(seconds)*time.Second +
		time.Duration(float64(fraction)/(1<<16)*float64(time.Second))
}

func formatReferenceID(stratum int, referenceID []byte) string {
	if stratum <= 1 {
		printable := make([]byte, 0, 4)
		for _, char := range referenceID {
			if char >= 32 && char < 127 {
				printable = append(printable, char)
			}
		}
		return string(printable)
	}

	return net.IPv4(referenceID[0], referenceID[1], referenceID[2], referenceID[3]).String()
}
