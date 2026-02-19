package memory

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"

	"open-dan/internal/llm"
)

// SQLiteMemory implements Memory using SQLite.
type SQLiteMemory struct {
	db *sql.DB
}

// NewSQLiteMemory opens (or creates) a SQLite database at the given path.
func NewSQLiteMemory(dbPath string) (*SQLiteMemory, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, err
	}

	m := &SQLiteMemory{db: db}
	if err := m.migrate(); err != nil {
		db.Close()
		return nil, err
	}

	return m, nil
}

func (m *SQLiteMemory) migrate() error {
	for _, stmt := range migrations {
		if _, err := m.db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

func (m *SQLiteMemory) SaveMessage(ctx context.Context, chatID string, msg llm.Message) error {
	var toolCallsJSON *string
	if len(msg.ToolCalls) > 0 {
		data, _ := json.Marshal(msg.ToolCalls)
		s := string(data)
		toolCallsJSON = &s
	}

	var toolCallID *string
	if msg.ToolCallID != "" {
		toolCallID = &msg.ToolCallID
	}

	_, err := m.db.ExecContext(ctx,
		`INSERT INTO messages (chat_id, role, content, tool_calls, tool_call_id) VALUES (?, ?, ?, ?, ?)`,
		chatID, msg.Role, msg.Content, toolCallsJSON, toolCallID,
	)
	return err
}

func (m *SQLiteMemory) GetHistory(ctx context.Context, chatID string, limit int) ([]llm.Message, error) {
	rows, err := m.db.QueryContext(ctx,
		`SELECT role, content, tool_calls, tool_call_id FROM (
			SELECT role, content, tool_calls, tool_call_id, id
			FROM messages WHERE chat_id = ? ORDER BY id DESC LIMIT ?
		) sub ORDER BY id ASC`,
		chatID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []llm.Message
	for rows.Next() {
		var msg llm.Message
		var toolCallsJSON, toolCallID sql.NullString

		if err := rows.Scan(&msg.Role, &msg.Content, &toolCallsJSON, &toolCallID); err != nil {
			return nil, err
		}

		if toolCallsJSON.Valid {
			_ = json.Unmarshal([]byte(toolCallsJSON.String), &msg.ToolCalls)
		}
		if toolCallID.Valid {
			msg.ToolCallID = toolCallID.String
		}

		messages = append(messages, msg)
	}

	return messages, rows.Err()
}

func (m *SQLiteMemory) SaveSummary(ctx context.Context, chatID string, summary string) error {
	_, err := m.db.ExecContext(ctx,
		`INSERT OR REPLACE INTO summaries (chat_id, summary, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP)`,
		chatID, summary,
	)
	return err
}

func (m *SQLiteMemory) GetSummary(ctx context.Context, chatID string) (string, error) {
	var summary string
	err := m.db.QueryRowContext(ctx,
		`SELECT summary FROM summaries WHERE chat_id = ?`,
		chatID,
	).Scan(&summary)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return summary, err
}

func (m *SQLiteMemory) Close() error {
	return m.db.Close()
}
