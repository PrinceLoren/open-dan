import { useState } from 'react';
import {
  SaveLLMConfig,
  SaveTelegramConfig,
  SaveSecurityConfig,
  CompleteSetup,
  TestLLMConnection,
  TestTelegramConnection,
} from '../../wailsjs/go/main/App';
import ProviderForm from '../components/ProviderForm';
import ChannelForm from '../components/ChannelForm';

interface Props {
  onComplete: () => void;
}

function SetupWizard({ onComplete }: Props) {
  const [step, setStep] = useState(0);
  const [error, setError] = useState('');

  // LLM state
  const [provider, setProvider] = useState('openai');
  const [apiKey, setApiKey] = useState('');
  const [model, setModel] = useState('');
  const [baseURL, setBaseURL] = useState('');
  const [llmStatus, setLlmStatus] = useState<'idle' | 'testing' | 'ok' | 'error'>('idle');

  // Telegram state
  const [tgToken, setTgToken] = useState('');
  const [tgStatus, setTgStatus] = useState<'idle' | 'testing' | 'ok' | 'error'>('idle');

  // Security state
  const [piiEnabled, setPiiEnabled] = useState(true);
  const [filterEmails, setFilterEmails] = useState(true);
  const [filterPhones, setFilterPhones] = useState(true);
  const [filterCards, setFilterCards] = useState(true);
  const [filterIPs, setFilterIPs] = useState(false);
  const [filterSSN, setFilterSSN] = useState(true);

  const testLLM = async () => {
    setLlmStatus('testing');
    setError('');
    try {
      const result = await TestLLMConnection(provider, apiKey, model, baseURL);
      if (result === 'OK') {
        setLlmStatus('ok');
      } else {
        setLlmStatus('error');
        setError(result);
      }
    } catch (e: any) {
      setLlmStatus('error');
      setError(e.toString());
    }
  };

  const testTelegram = async () => {
    if (!tgToken) return;
    setTgStatus('testing');
    setError('');
    try {
      const result = await TestTelegramConnection(tgToken);
      if (result === 'OK') {
        setTgStatus('ok');
      } else {
        setTgStatus('error');
        setError(result);
      }
    } catch (e: any) {
      setTgStatus('error');
      setError(e.toString());
    }
  };

  const finish = async () => {
    setError('');
    try {
      await SaveLLMConfig(provider, apiKey, model, baseURL);
      if (tgToken) {
        await SaveTelegramConfig(tgToken, []);
      }
      await SaveSecurityConfig(piiEnabled, filterEmails, filterPhones, filterCards, filterIPs, filterSSN);
      await CompleteSetup();
      onComplete();
    } catch (e: any) {
      setError(e.toString());
    }
  };

  const steps = [
    // Step 0: Welcome
    <div className="wizard-step" key="welcome">
      <h1>Welcome to OpenDan</h1>
      <p className="subtitle">Your autonomous AI assistant. Let's get you set up in a few steps.</p>
      <button className="btn btn-primary" onClick={() => setStep(1)}>
        Start Setup
      </button>
    </div>,

    // Step 1: LLM Provider
    <div className="wizard-step" key="llm">
      <h2>Step 1: LLM Provider</h2>
      <p className="subtitle">Choose your AI provider and enter the API key.</p>
      <ProviderForm
        provider={provider}
        apiKey={apiKey}
        model={model}
        baseURL={baseURL}
        onProviderChange={setProvider}
        onApiKeyChange={setApiKey}
        onModelChange={setModel}
        onBaseURLChange={setBaseURL}
      />
      <div className="button-row">
        <button
          className={`btn ${llmStatus === 'ok' ? 'btn-success' : llmStatus === 'error' ? 'btn-danger' : 'btn-secondary'}`}
          onClick={testLLM}
          disabled={!apiKey || llmStatus === 'testing'}
        >
          {llmStatus === 'testing' ? 'Testing...' : llmStatus === 'ok' ? 'Connected!' : 'Test Connection'}
        </button>
        <button
          className="btn btn-primary"
          onClick={() => setStep(2)}
          disabled={llmStatus !== 'ok'}
        >
          Next
        </button>
      </div>
    </div>,

    // Step 2: Messenger
    <div className="wizard-step" key="channel">
      <h2>Step 2: Messenger</h2>
      <p className="subtitle">Connect a messaging platform (optional).</p>
      <ChannelForm
        token={tgToken}
        onTokenChange={setTgToken}
      />
      <div className="button-row">
        {tgToken && (
          <button
            className={`btn ${tgStatus === 'ok' ? 'btn-success' : tgStatus === 'error' ? 'btn-danger' : 'btn-secondary'}`}
            onClick={testTelegram}
            disabled={tgStatus === 'testing'}
          >
            {tgStatus === 'testing' ? 'Testing...' : tgStatus === 'ok' ? 'Connected!' : 'Test Connection'}
          </button>
        )}
        <button className="btn btn-primary" onClick={() => setStep(3)}>
          {tgToken ? 'Next' : 'Skip'}
        </button>
      </div>
    </div>,

    // Step 3: Security
    <div className="wizard-step" key="security">
      <h2>Step 3: Security</h2>
      <p className="subtitle">Configure PII filtering for messages sent to the AI.</p>
      <div className="form-group">
        <label className="checkbox-label">
          <input type="checkbox" checked={piiEnabled} onChange={(e) => setPiiEnabled(e.target.checked)} />
          Enable PII Filtering
        </label>
      </div>
      {piiEnabled && (
        <div className="filter-options">
          <label className="checkbox-label">
            <input type="checkbox" checked={filterEmails} onChange={(e) => setFilterEmails(e.target.checked)} />
            Filter email addresses
          </label>
          <label className="checkbox-label">
            <input type="checkbox" checked={filterPhones} onChange={(e) => setFilterPhones(e.target.checked)} />
            Filter phone numbers
          </label>
          <label className="checkbox-label">
            <input type="checkbox" checked={filterCards} onChange={(e) => setFilterCards(e.target.checked)} />
            Filter credit card numbers
          </label>
          <label className="checkbox-label">
            <input type="checkbox" checked={filterIPs} onChange={(e) => setFilterIPs(e.target.checked)} />
            Filter IP addresses
          </label>
          <label className="checkbox-label">
            <input type="checkbox" checked={filterSSN} onChange={(e) => setFilterSSN(e.target.checked)} />
            Filter SSN numbers
          </label>
        </div>
      )}
      <div className="button-row">
        <button className="btn btn-primary" onClick={() => setStep(4)}>
          Next
        </button>
      </div>
    </div>,

    // Step 4: Done
    <div className="wizard-step" key="done">
      <h1>All Set!</h1>
      <div className="status-summary">
        <div className={`status-item ${llmStatus === 'ok' ? 'status-ok' : ''}`}>
          LLM Provider: {provider} {llmStatus === 'ok' ? '(connected)' : ''}
        </div>
        <div className={`status-item ${tgStatus === 'ok' ? 'status-ok' : ''}`}>
          Telegram: {tgToken ? (tgStatus === 'ok' ? 'connected' : 'configured') : 'skipped'}
        </div>
        <div className="status-item status-ok">
          PII Filtering: {piiEnabled ? 'enabled' : 'disabled'}
        </div>
      </div>
      <button className="btn btn-primary btn-large" onClick={finish}>
        Launch OpenDan
      </button>
    </div>,
  ];

  return (
    <div className="wizard-container">
      <div className="wizard-progress">
        {['Welcome', 'LLM', 'Messenger', 'Security', 'Done'].map((label, i) => (
          <div key={i} className={`progress-step ${i <= step ? 'active' : ''} ${i < step ? 'completed' : ''}`}>
            <div className="progress-dot">{i < step ? '\u2713' : i + 1}</div>
            <span>{label}</span>
          </div>
        ))}
      </div>
      {error && <div className="error-banner">{error}</div>}
      {steps[step]}
    </div>
  );
}

export default SetupWizard;
