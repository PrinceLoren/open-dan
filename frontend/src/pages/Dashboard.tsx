import { useState, useEffect, useRef } from 'react';
import {
  GetConfig,
  GetChannelStatus,
  GetLogs,
  GetMemStats,
  SendMessage,
} from '../../wailsjs/go/main/App';
import StatusCard from '../components/StatusCard';

interface Props {
  onNavigate: (page: 'settings' | 'dashboard') => void;
}

interface LogEntry {
  level: string;
  message: string;
  time: string;
}

function Dashboard({ onNavigate }: Props) {
  const [config, setConfig] = useState<any>(null);
  const [channels, setChannels] = useState<Record<string, boolean>>({});
  const [logs, setLogs] = useState<LogEntry[]>([]);
  const [chatInput, setChatInput] = useState('');
  const [chatMessages, setChatMessages] = useState<{ role: string; text: string }[]>([]);
  const [sending, setSending] = useState(false);
  const [memStats, setMemStats] = useState<any>(null);
  const chatEndRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    loadStatus();
    const interval = setInterval(loadStatus, 5000);
    return () => clearInterval(interval);
  }, []);

  useEffect(() => {
    chatEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [chatMessages]);

  const loadStatus = async () => {
    try {
      const [cfg, chs, lgs, mem] = await Promise.all([
        GetConfig(),
        GetChannelStatus(),
        GetLogs(),
        GetMemStats(),
      ]);
      setConfig(cfg);
      setChannels(chs || {});
      setLogs(lgs || []);
      setMemStats(mem);
    } catch (e) {
      console.error('Failed to load status:', e);
    }
  };

  const sendMessage = async () => {
    if (!chatInput.trim() || sending) return;
    const text = chatInput;
    setChatInput('');
    setChatMessages((prev) => [...prev, { role: 'user', text }]);
    setSending(true);
    try {
      const response = await SendMessage(text);
      setChatMessages((prev) => [...prev, { role: 'assistant', text: response }]);
    } catch (e: any) {
      setChatMessages((prev) => [...prev, { role: 'error', text: e.toString() }]);
    }
    setSending(false);
  };

  return (
    <div className="dashboard">
      <header className="dashboard-header">
        <h1>OpenDan</h1>
        <button className="btn btn-secondary" onClick={() => onNavigate('settings')}>
          Settings
        </button>
      </header>

      <div className="dashboard-grid">
        <div className="dashboard-left">
          <div className="status-cards">
            <StatusCard
              title="LLM Provider"
              value={config ? `${config.provider} (${config.model})` : 'Loading...'}
              status={config?.api_key_masked ? 'ok' : 'error'}
            />
            <StatusCard
              title="Telegram"
              value={config?.has_telegram ? 'Connected' : 'Not configured'}
              status={config?.has_telegram ? (channels['telegram'] ? 'ok' : 'warn') : 'off'}
            />
            <StatusCard
              title="PII Filtering"
              value={config?.pii_filtering ? 'Enabled' : 'Disabled'}
              status={config?.pii_filtering ? 'ok' : 'warn'}
            />
            <StatusCard
              title="Browser"
              value={config?.browser_enabled ? `Enabled${config?.browser_headless ? ' (headless)' : ''}` : 'Disabled'}
              status={config?.browser_enabled ? 'ok' : 'off'}
            />
            <StatusCard
              title="Skills"
              value={config?.plugins_enabled ? `${config?.skills_count || 0} installed` : 'Disabled'}
              status={config?.plugins_enabled ? (config?.skills_count > 0 ? 'ok' : 'warn') : 'off'}
            />
            <StatusCard
              title="Memory (Go)"
              value={memStats ? `${memStats.heap_alloc_mb?.toFixed(1)} MB heap | ${memStats.sys_mb?.toFixed(1)} MB sys | ${memStats.goroutines} goroutines` : 'Loading...'}
              status="ok"
            />
          </div>

          <div className="log-panel">
            <h3>Logs</h3>
            <div className="log-entries">
              {logs.length === 0 ? (
                <p className="log-empty">No logs yet</p>
              ) : (
                logs.slice(-50).map((log, i) => (
                  <div key={i} className={`log-entry log-${log.level}`}>
                    <span className="log-level">[{log.level}]</span> {log.message}
                  </div>
                ))
              )}
            </div>
          </div>
        </div>

        <div className="dashboard-right">
          <div className="chat-panel">
            <h3>Chat</h3>
            <div className="chat-messages">
              {chatMessages.length === 0 && (
                <p className="chat-empty">Send a message to start chatting with OpenDan</p>
              )}
              {chatMessages.map((msg, i) => (
                <div key={i} className={`chat-message chat-${msg.role}`}>
                  <strong>{msg.role === 'user' ? 'You' : msg.role === 'assistant' ? 'OpenDan' : 'Error'}:</strong>
                  <pre>{msg.text}</pre>
                </div>
              ))}
              {sending && (
                <div className="chat-message chat-assistant">
                  <div className="typing-indicator">
                    <span /><span /><span />
                  </div>
                </div>
              )}
              <div ref={chatEndRef} />
            </div>
            <div className="chat-input-row">
              <input
                type="text"
                className="input chat-input"
                value={chatInput}
                onChange={(e) => setChatInput(e.target.value)}
                onKeyDown={(e) => e.key === 'Enter' && sendMessage()}
                placeholder="Type a message..."
                disabled={sending}
              />
              <button className="btn btn-primary" onClick={sendMessage} disabled={sending || !chatInput.trim()}>
                Send
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

export default Dashboard;
