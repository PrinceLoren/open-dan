interface Props {
  provider: string;
  apiKey: string;
  model: string;
  baseURL: string;
  onProviderChange: (v: string) => void;
  onApiKeyChange: (v: string) => void;
  onModelChange: (v: string) => void;
  onBaseURLChange: (v: string) => void;
}

const providerDefaults: Record<string, { model: string; placeholder: string }> = {
  openai: { model: 'gpt-4o-mini', placeholder: 'sk-...' },
  anthropic: { model: 'claude-sonnet-4-20250514', placeholder: 'sk-ant-...' },
  openrouter: { model: 'anthropic/claude-sonnet-4-20250514', placeholder: 'sk-or-...' },
  local: { model: 'llama3', placeholder: 'not required for local models' },
};

function ProviderForm({ provider, apiKey, model, baseURL, onProviderChange, onApiKeyChange, onModelChange, onBaseURLChange }: Props) {
  const defaults = providerDefaults[provider] || providerDefaults.openai;

  const handleProviderChange = (newProvider: string) => {
    onProviderChange(newProvider);
    const pd = providerDefaults[newProvider];
    if (pd) {
      onModelChange(pd.model);
    }
    if (newProvider === 'openrouter') {
      onBaseURLChange('https://openrouter.ai/api/v1');
    } else if (newProvider === 'local') {
      onBaseURLChange('http://localhost:11434/v1');
    } else {
      onBaseURLChange('');
    }
  };

  return (
    <div className="form-container">
      <div className="form-group">
        <label>Provider</label>
        <select className="input" value={provider} onChange={(e) => handleProviderChange(e.target.value)}>
          <option value="openai">OpenAI</option>
          <option value="anthropic">Anthropic</option>
          <option value="openrouter">OpenRouter</option>
          <option value="local">Local Model (Ollama / LM Studio)</option>
        </select>
      </div>

      <div className="form-group">
        <label>API Key</label>
        <input
          type="password"
          className="input"
          value={apiKey}
          onChange={(e) => onApiKeyChange(e.target.value)}
          placeholder={defaults.placeholder}
        />
      </div>

      <div className="form-group">
        <label>Model</label>
        <input
          type="text"
          className="input"
          value={model || defaults.model}
          onChange={(e) => onModelChange(e.target.value)}
          placeholder={defaults.model}
        />
      </div>

      {(provider === 'local' || provider === 'openrouter') && (
        <div className="form-group">
          <label>Base URL</label>
          <input
            type="text"
            className="input"
            value={baseURL}
            onChange={(e) => onBaseURLChange(e.target.value)}
            placeholder="http://localhost:11434/v1"
          />
        </div>
      )}
    </div>
  );
}

export default ProviderForm;
