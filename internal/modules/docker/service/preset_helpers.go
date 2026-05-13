package service

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	dockertypes "github.com/docker/docker/api/types"
	"github.com/sametkarademir/forge/internal/core/config"
	"github.com/sametkarademir/forge/internal/core/logger"
	"github.com/sametkarademir/forge/internal/modules/docker/engines"
	"github.com/sametkarademir/forge/internal/modules/docker/preset"
)

func buildContainerInfo(c *dockertypes.ContainerJSON, p *preset.Preset) *ContainerInfo {
	labels := c.Config.Labels
	port, _ := strconv.Atoi(labels["forge.host_port"])
	createdAt, _ := time.Parse(time.RFC3339, labels["forge.created_at"])
	presetName := labels["forge.preset"]
	engineName := labels["forge.engine"]

	id := c.ID
	if len(id) > 12 {
		id = id[:12]
	}

	var dsn string
	if p != nil {
		if eng, ok := engines.Get(engineName); ok {
			dsn = eng.ConnectionString("localhost", port, p.Username, p.Password, p.Database)
		}
	}

	return &ContainerInfo{
		PresetName:       presetName,
		Engine:           engineName,
		ContainerID:      id,
		ContainerName:    PresetContainerName(presetName),
		VolumeName:       PresetVolumeName(presetName),
		HostPort:         port,
		Status:           c.State.Status,
		Image:            c.Config.Image,
		CreatedAt:        createdAt,
		ConnectionString: dsn,
	}
}

func readinessTimeout(override int) int {
	if override > 0 {
		return override
	}
	return config.ReadinessTimeoutSeconds()
}

func maskPasswordInDSN(dsn string) string {
	// postgres://user:pass@host:port/db
	if idx := strings.Index(dsn, "://"); idx != -1 {
		rest := dsn[idx+3:]
		atIdx := strings.LastIndex(rest, "@")
		if atIdx != -1 {
			userPass := rest[:atIdx]
			colonIdx := strings.Index(userPass, ":")
			if colonIdx != -1 {
				return dsn[:idx+3] + userPass[:colonIdx+1] + "****" + "@" + rest[atIdx+1:]
			}
		}
	}
	// mssql: Server=…;Password=val;
	if strings.HasPrefix(dsn, "Server=") {
		parts := strings.Split(dsn, ";")
		for i, p := range parts {
			if strings.HasPrefix(strings.ToLower(p), "password=") {
				parts[i] = "Password=****"
			}
		}
		return strings.Join(parts, ";")
	}
	return dsn
}

func waitForPresetReady(port, timeoutSecs int) {
	addr := fmt.Sprintf("localhost:%d", port)
	deadline := time.Now().Add(time.Duration(timeoutSecs) * time.Second)
	start := time.Now()
	logger.Info("Waiting for DB to accept connections…")
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, time.Second)
		if err == nil {
			_ = conn.Close()
			logger.Success(fmt.Sprintf("DB ready (%s)", time.Since(start).Round(time.Second)))
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
	logger.Warn(fmt.Sprintf(
		"DB did not become ready within %ds — try connecting again in a moment.",
		timeoutSecs,
	))
}
