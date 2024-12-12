package ui

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/magodo/pipeform/internal/terraform/views/json"
)

type ResourceStatus string

const (
	// Once received one OperationStart hook message
	ResourceStatusStart ResourceStatus = "start"
	// Once received one or more OperationProgress hook message
	ResourceStatusProgress ResourceStatus = "progress"
	// Once received one OperationComplete hook message
	ResourceStatusComplete ResourceStatus = "complete"
	// Once received one OperationErrored hook message
	ResourceStatusErrored ResourceStatus = "error"

	// TODO: Support refresh? (refresh is an independent lifecycle than the resource apply lifecycle)
	// TODO: Support provision? (provision is a intermidiate stage in the resource apply lifecycle)
)

func ResourceStatusEmoji(status ResourceStatus) string {
	switch status {
	case ResourceStatusStart:
		return "🕛"
	case ResourceStatusProgress:
		states := []string{"🕛", "🕐", "🕑", "🕒", "🕓", "🕔", "🕕", "🕖", "🕗", "🕘", "🕙", "🕚"}
		return states[rand.Intn(len(states))]
	case ResourceStatusComplete:
		return "✅"
	case ResourceStatusErrored:
		return "❌"
	default:
		return "❓"
	}
}

type ResourceInfoLocator struct {
	Module       string
	ResourceAddr string
	Action       json.ChangeAction
}

type ResourceInfo struct {
	Loc       ResourceInfoLocator
	Status    ResourceStatus
	StartTime time.Time
	EndTime   time.Time
	Diags     []json.Diagnostic
}

type ResourceInfoUpdate struct {
	Status  *ResourceStatus
	Endtime *time.Time
}

// ResourceInfos records the operation information for each resource's action.
// The first key is the ResourceAddr.
// The second key is the resource Action.
// It can happen that one single resource have more than one actions conducted in one apply,
// e.g., a resource being re-created (remove + create).
// type ResourceInfos map[string]map[json.ChangeAction]*ResourceInfo
type ResourceInfos []*ResourceInfo

func (infos ResourceInfos) Update(loc ResourceInfoLocator, update ResourceInfoUpdate) bool {
	for _, info := range infos {
		if info.Loc == loc {
			if update.Status != nil {
				info.Status = *update.Status
			}
			if update.Endtime != nil {
				info.EndTime = *update.Endtime
			}
			return true
		}
	}
	return false
}

func (infos ResourceInfos) AddDiags(loc ResourceInfoLocator, diags ...json.Diagnostic) bool {
	for _, info := range infos {
		if info.Loc == loc {
			info.Diags = append(info.Diags, diags...)
			return true
		}
	}
	return false
}

func (infos ResourceInfos) ToRows(total int) []table.Row {
	var rows []table.Row
	for i, info := range infos {
		row := []string{
			fmt.Sprintf("%d/%d", i+1, total),
			ResourceStatusEmoji(info.Status),
			string(info.Loc.Action),
			info.Loc.Module,
			info.Loc.ResourceAddr,
		}
		rows = append(rows, row)
	}
	return rows
}
