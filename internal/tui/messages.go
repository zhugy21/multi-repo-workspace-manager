package tui

import "github.com/user/ws/pkg/types"

type TickMsg struct {
	Results  []types.Result
	Complete bool
}
