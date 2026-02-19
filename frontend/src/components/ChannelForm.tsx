interface Props {
  token: string;
  onTokenChange: (v: string) => void;
}

function ChannelForm({ token, onTokenChange }: Props) {
  return (
    <div className="form-container">
      <div className="form-group">
        <label>Telegram Bot Token</label>
        <input
          type="password"
          className="input"
          value={token}
          onChange={(e) => onTokenChange(e.target.value)}
          placeholder="123456:ABC-DEF..."
        />
        <p className="help-text">
          Create a bot via{' '}
          <a href="https://t.me/BotFather" target="_blank" rel="noopener noreferrer">
            @BotFather
          </a>{' '}
          on Telegram and paste the token here.
        </p>
      </div>
    </div>
  );
}

export default ChannelForm;
