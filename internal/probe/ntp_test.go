package probe

import (
	"encoding/binary"
	"net"
	"testing"
	"time"

	"monitoring-platform/internal/domain"
)

// startFakeNTPServer answers standard client requests with a valid mode-4
// response whose clocks match the local machine (offset ~0).
func startFakeNTPServer(t *testing.T, stratum byte, leap byte, mutate func(request, response []byte)) string {
	t.Helper()

	packetConn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen udp failed: %v", err)
	}

	go func() {
		buffer := make([]byte, ntpPacketSize)
		for {
			read, addr, err := packetConn.ReadFrom(buffer)
			if err != nil {
				return
			}
			if read < ntpPacketSize {
				continue
			}

			response := make([]byte, ntpPacketSize)
			response[0] = leap<<6 | 4<<3 | ntpServerMode
			response[1] = stratum
			// reference id
			copy(response[12:16], []byte{203, 0, 113, 1})

			// origin = client transmit
			copy(response[24:32], buffer[40:48])

			now := toNTPTimestamp(time.Now())
			binary.BigEndian.PutUint64(response[32:40], now)
			binary.BigEndian.PutUint64(response[40:48], now)

			if mutate != nil {
				mutate(buffer[:read], response)
			}

			_, _ = packetConn.WriteTo(response, addr)
		}
	}()

	t.Cleanup(func() { _ = packetConn.Close() })

	return packetConn.LocalAddr().String()
}

func ntpJobFor(address string, config map[string]any) domain.ProbeJob {
	host, portRaw, _ := net.SplitHostPort(address)
	if config == nil {
		config = map[string]any{}
	}
	config["port"] = portRaw

	return testJob(domain.MonitorNTP, host, config)
}

func TestNTPExecutorSuccess(t *testing.T) {
	address := startFakeNTPServer(t, 2, 0, nil)

	executor := NewNTPExecutor(testDeps())
	result := executor.Execute(execCtx(t), ntpJobFor(address, nil))

	if !result.Success {
		t.Fatalf("expected success, got %+v (error: %s)", result, result.ErrorMessage)
	}
	if result.Metrics["stratum"] != 2 {
		t.Fatalf("expected stratum 2, got %v", result.Metrics["stratum"])
	}
	if result.Attributes["reference_id"] != "203.0.113.1" {
		t.Fatalf("unexpected reference id %v", result.Attributes["reference_id"])
	}
}

func TestNTPExecutorUnsynchronized(t *testing.T) {
	address := startFakeNTPServer(t, 2, ntpLeapUnsynced, nil)

	executor := NewNTPExecutor(testDeps())
	result := executor.Execute(execCtx(t), ntpJobFor(address, nil))

	if result.Success || result.ErrorCode != "ntp_unsynchronized" {
		t.Fatalf("expected ntp_unsynchronized, got %+v", result)
	}
}

func TestNTPExecutorKissOfDeath(t *testing.T) {
	address := startFakeNTPServer(t, 0, 0, func(request, response []byte) {
		copy(response[12:16], []byte("RATE"))
	})

	executor := NewNTPExecutor(testDeps())
	result := executor.Execute(execCtx(t), ntpJobFor(address, nil))

	if result.Success || result.ErrorCode != "ntp_kiss_of_death" {
		t.Fatalf("expected ntp_kiss_of_death, got %+v", result)
	}
}

func TestNTPExecutorStratumOutOfRange(t *testing.T) {
	address := startFakeNTPServer(t, 14, 0, nil)

	executor := NewNTPExecutor(testDeps())
	result := executor.Execute(execCtx(t), ntpJobFor(address, map[string]any{
		"allowed_stratum_max": float64(4),
	}))

	if result.Success || result.ErrorCode != "ntp_stratum_invalid" {
		t.Fatalf("expected ntp_stratum_invalid, got %+v", result)
	}
}

func TestNTPExecutorOriginMismatch(t *testing.T) {
	address := startFakeNTPServer(t, 2, 0, func(request, response []byte) {
		binary.BigEndian.PutUint64(response[24:32], 12345)
	})

	executor := NewNTPExecutor(testDeps())
	result := executor.Execute(execCtx(t), ntpJobFor(address, nil))

	if result.Success || result.ErrorCode != "ntp_invalid_response" {
		t.Fatalf("expected ntp_invalid_response, got %+v", result)
	}
}
