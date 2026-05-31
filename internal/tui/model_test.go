package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"

	"github.com/user/ws/pkg/types"
)

func TestNewModel(t *testing.T) {
	m := NewModel("sync", "myproject", "auto")
	assert.Equal(t, "sync", m.command)
	assert.Equal(t, "myproject", m.workspace)
	assert.Equal(t, "auto", m.mode)
	assert.Equal(t, 0, len(m.results))
}

func TestModelUpdateWithResults(t *testing.T) {
	m := NewModel("sync", "test", "auto")
	results := []types.Result{
		{RepoName: "a", Status: types.StatusSuccess, Detail: "ok", Duration: time.Second},
		{RepoName: "b", Status: types.StatusFailed, Detail: "fail", Duration: 100 * time.Millisecond},
	}
	msg := TickMsg{Results: results, Complete: false}
	m2, _ := m.Update(msg)
	updated := m2.(Model)
	assert.Equal(t, 2, len(updated.results))
	assert.Equal(t, types.StatusSuccess, updated.results[0].Status)
	assert.Equal(t, types.StatusFailed, updated.results[1].Status)
}

func TestModelQuitOnComplete(t *testing.T) {
	m := NewModel("sync", "test", "auto")
	msg := TickMsg{Results: []types.Result{{RepoName: "a", Status: types.StatusSuccess}}, Complete: true}
	_, cmd := m.Update(msg)
	assert.NotNil(t, cmd)
}

func TestModelQuitOnQKey(t *testing.T) {
	m := NewModel("sync", "test", "auto")
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	assert.NotNil(t, cmd)
}

func TestStatusIcon(t *testing.T) {
	assert.Equal(t, "✅", StatusIcon(string(types.StatusSuccess)))
	assert.Equal(t, "✗", StatusIcon(string(types.StatusFailed)))
	assert.Equal(t, "⏹", StatusIcon(string(types.StatusCancelled)))
	assert.Equal(t, "◌", StatusIcon(string(types.StatusSkipped)))
	assert.Equal(t, "⚠", StatusIcon(string(types.StatusWarning)))
	assert.Equal(t, "⏳", StatusIcon(string(types.StatusRunning)))
	assert.Equal(t, "○", StatusIcon(string(types.StatusPending)))
	assert.Equal(t, "?", StatusIcon("unknown"))
}

func TestViewRendersHeader(t *testing.T) {
	m := NewModel("sync", "myproject", "auto")
	view := m.View()
	assert.Contains(t, view, "sync")
	assert.Contains(t, view, "myproject")
}

func TestViewRendersFooter(t *testing.T) {
	m := NewModel("sync", "test", "auto")
	m.results = []types.Result{
		{RepoName: "a", Status: types.StatusSuccess},
		{RepoName: "b", Status: types.StatusFailed},
	}
	view := m.View()
	assert.Contains(t, view, "failed")
}
