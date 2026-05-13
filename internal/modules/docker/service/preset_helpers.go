package service

import (
	"fmt"
	"net"
	"strconv"
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

	var connStr string
	var endpoints []engines.Endpoint
	if p != nil {
		if eng, ok := engines.Get(engineName); ok {
			ci := eng.ConnectionInfo(engines.ConnArgs{
				Host:     "localhost",
				HostPort: port,
				User:     p.Username,
				Password: p.Password,
				Database: p.Database,
				Options:  p.Options,
			})
			connStr = ci.Primary
			endpoints = ci.Endpoints
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
		ConnectionString: connStr,
		Endpoints:        endpoints,
	}
}

func readinessTimeout(override int) int {
	if override > 0 {
		return override
	}
	return config.ReadinessTimeoutSeconds()
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
