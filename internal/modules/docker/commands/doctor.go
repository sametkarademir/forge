package commands

import (
	"context"
	"fmt"
	"strconv"

	"github.com/fatih/color"
	"github.com/sametkarademir/forge/internal/core/config"
	dockerclient "github.com/sametkarademir/forge/internal/modules/docker/client"
	"github.com/sametkarademir/forge/internal/modules/docker/preset"
	"github.com/sametkarademir/forge/internal/modules/docker/service"
	"github.com/spf13/cobra"
)

var (
	checkOK   = color.New(color.FgGreen).Sprint("✓")
	checkWarn = color.New(color.FgYellow).Sprint("⚠")
	checkFail = color.New(color.FgRed).Sprint("✗")
)

func doctorLine(icon, label, detail string) {
	if detail != "" {
		fmt.Printf("  %s  %-32s %s\n", icon, label, detail)
	} else {
		fmt.Printf("  %s  %s\n", icon, label)
	}
}

// NewDoctorCommand returns the health-check command.
func NewDoctorCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check forge docker health: daemon, network, port range, preset integrity",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runDoctor(cmd.Context())
		},
	}
}

func runDoctor(ctx context.Context) error {
	fmt.Println("forge docker doctor")
	fmt.Println()

	// 1. Docker daemon reachable?
	dc, daemonErr := dockerclient.NewClient()
	if daemonErr != nil {
		doctorLine(checkFail, "Docker daemon", "unreachable — run: open -a Docker")
		fmt.Println()
		fmt.Println("Cannot continue without Docker. Fix daemon access first.")
		return daemonErr
	}
	doctorLine(checkOK, "Docker daemon", "reachable")

	// 2. forge-net exists?
	empty, err := dc.IsNetworkEmpty(ctx, "forge-net")
	if err != nil {
		doctorLine(checkWarn, "forge-net", "not found (will be created on first run)")
	} else {
		detail := "exists"
		if empty {
			detail = "exists (no containers connected)"
		}
		doctorLine(checkOK, "forge-net", detail)
	}

	// 3. Port range sanity.
	start := config.PortRangeStart()
	end := config.PortRangeEnd()
	if start >= end || start < 1024 || end > 65535 {
		doctorLine(checkFail, "Port range",
			fmt.Sprintf("%d–%d is invalid (must be 1024–65535 and start < end)", start, end))
	} else {
		free := 0
		for p := start; p <= end; p++ {
			if service.IsPortFree(p) {
				free++
			}
		}
		icon := checkOK
		detail := fmt.Sprintf("%d–%d  (%d/%d free)", start, end, free, end-start+1)
		if free == 0 {
			icon = checkFail
			detail += "  — port range exhausted!"
		} else if free < 5 {
			icon = checkWarn
			detail += "  — almost exhausted"
		}
		doctorLine(icon, "Port range", detail)
	}

	// 4. Preset integrity — invalid or stale schema.
	entries, _ := preset.ListEntries()
	invalid := 0
	v1 := 0
	for _, e := range entries {
		if e.Err != nil {
			invalid++
		} else if e.Preset != nil && e.Preset.SchemaVersion < 2 {
			v1++
		}
	}
	if invalid > 0 {
		doctorLine(checkFail, "Preset files",
			fmt.Sprintf("%d invalid (run 'forge docker list' to identify)", invalid))
	}
	if v1 > 0 {
		doctorLine(checkWarn, "Legacy presets",
			fmt.Sprintf("%d preset(s) on schema v1 — may need migration", v1))
	}
	if invalid == 0 && v1 == 0 {
		doctorLine(checkOK, "Preset files",
			fmt.Sprintf("%d preset(s), all valid", len(entries)))
	}

	// 5. Orphaned containers / volumes.
	orphans, err := service.FindOrphans(ctx)
	if err != nil {
		doctorLine(checkWarn, "Orphan check", "failed: "+err.Error())
	} else if len(orphans) > 0 {
		doctorLine(checkWarn, "Orphaned resources",
			strconv.Itoa(len(orphans))+" found — run 'forge docker prune' to clean up")
	} else {
		doctorLine(checkOK, "Orphaned resources", "none")
	}

	fmt.Println()
	return nil
}
