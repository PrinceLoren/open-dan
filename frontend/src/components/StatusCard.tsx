interface Props {
  title: string;
  value: string;
  status: 'ok' | 'warn' | 'error' | 'off';
}

const statusColors: Record<string, string> = {
  ok: '#22c55e',
  warn: '#eab308',
  error: '#ef4444',
  off: '#6b7280',
};

function StatusCard({ title, value, status }: Props) {
  return (
    <div className="status-card">
      <div className="status-indicator" style={{ backgroundColor: statusColors[status] }} />
      <div className="status-info">
        <h4>{title}</h4>
        <p>{value}</p>
      </div>
    </div>
  );
}

export default StatusCard;
