package admin

import (
	"context"
	"fmt"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
)

type ServerStats struct {
	CPUPercent       float64
	MemUsed          uint64
	MemTotal         uint64
	MemPercent       float64
	UpMbpsTotal      float64
	DownMbpsTotal    float64
	TopIfName        string
	TopIfUpMbps      float64
	TopIfDownMbps    float64
	IntervalMeasured time.Duration
}

func ServerStatHandler(update tgbotapi.Update, bot *tgbotapi.BotAPI) {
	stats, err := collectServerStats(context.Background(), 1*time.Second)
	if err != nil {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "❌ Не удалось получить метрики: "+err.Error())
		_, _ = bot.Send(msg)
		return
	}

	report := fmt.Sprintf(
		"📊 *Загрузка сервера*\n"+
			"• CPU: *%.1f%%*\n"+
			"• RAM: *%s / %s* (%.1f%%)\n"+
			"• NET (%ds окно): ↑ *%s* ↓ *%s*\n"+
			"%s",
		stats.CPUPercent,
		formatBytes(stats.MemUsed), formatBytes(stats.MemTotal), stats.MemPercent,
		int(stats.IntervalMeasured.Seconds()),
		formatMbps(stats.UpMbpsTotal), formatMbps(stats.DownMbpsTotal),
		formatTopIface(stats),
	)

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, report)
	msg.ParseMode = "Markdown"
	_, _ = bot.Send(msg)
}

func collectServerStats(ctx context.Context, window time.Duration) (*ServerStats, error) {
	// Память — снимок сразу
	vm, err := mem.VirtualMemory()
	if err != nil {
		return nil, fmt.Errorf("mem: %w", err)
	}

	before, err := net.IOCounters(true)
	if err != nil {
		return nil, fmt.Errorf("net before: %w", err)
	}

	cpuPercChan := make(chan []float64, 1)
	errChan := make(chan error, 1)
	go func() {
		p, err := cpu.Percent(window, false)
		if err != nil {
			errChan <- err
			return
		}
		cpuPercChan <- p
	}()

	var cpuPercent float64
	select {
	case p := <-cpuPercChan:
		if len(p) > 0 {
			cpuPercent = p[0]
		}
	case err := <-errChan:
		return nil, fmt.Errorf("cpu: %w", err)
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	after, err := net.IOCounters(true)
	if err != nil {
		return nil, fmt.Errorf("net after: %w", err)
	}

	type ifDelta struct {
		name    string
		upbps   float64
		downbps float64
	}
	var (
		totalUpbps, totalDownbps float64
		top                      ifDelta
	)

	beforeMap := map[string]net.IOCountersStat{}
	for _, b := range before {
		beforeMap[b.Name] = b
	}

	secs := window.Seconds()
	for _, a := range after {
		if isIgnoredIF(a.Name) {
			continue
		}
		b, ok := beforeMap[a.Name]
		if !ok {
			continue
		}

		upBps := float64(a.BytesSent-b.BytesSent) / secs
		downBps := float64(a.BytesRecv-b.BytesRecv) / secs

		upbps := upBps * 8
		downbps := downBps * 8

		totalUpbps += upbps
		totalDownbps += downbps

		if (upbps + downbps) > (top.upbps + top.downbps) {
			top = ifDelta{name: a.Name, upbps: upbps, downbps: downbps}
		}
	}

	stats := &ServerStats{
		CPUPercent:       cpuPercent,
		MemUsed:          vm.Used,
		MemTotal:         vm.Total,
		MemPercent:       vm.UsedPercent,
		UpMbpsTotal:      bpsToMbps(totalUpbps),
		DownMbpsTotal:    bpsToMbps(totalDownbps),
		TopIfName:        top.name,
		TopIfUpMbps:      bpsToMbps(top.upbps),
		TopIfDownMbps:    bpsToMbps(top.downbps),
		IntervalMeasured: window,
	}
	return stats, nil
}

func isIgnoredIF(name string) bool {
	n := strings.ToLower(name)
	if n == "lo" {
		return true
	}

	prefixes := []string{"veth", "br-", "vmnet", "kube", "tailscale", "wg", "zt"}
	for _, p := range prefixes {
		if strings.HasPrefix(n, p) {
			return true
		}
	}
	return false
}

func bpsToMbps(bps float64) float64 {
	return bps / 1_000_000.0
}

func formatMbps(v float64) string {
	return fmt.Sprintf("%.2f Mbps", v)
}

func formatBytes(b uint64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)
	switch {
	case b >= GB:
		return fmt.Sprintf("%.2f GiB", float64(b)/float64(GB))
	case b >= MB:
		return fmt.Sprintf("%.2f MiB", float64(b)/float64(MB))
	case b >= KB:
		return fmt.Sprintf("%.2f KiB", float64(b)/float64(KB))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

func formatTopIface(s *ServerStats) string {
	if s.TopIfName == "" {
		return "• Активный интерфейс: не обнаружен"
	}
	return fmt.Sprintf("• Топ-интерфейс: `%s` ↑ %s ↓ %s",
		s.TopIfName, formatMbps(s.TopIfUpMbps), formatMbps(s.TopIfDownMbps))
}
