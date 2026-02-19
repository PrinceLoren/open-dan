import { useState, useEffect } from 'react';
import {
  GetConfig,
  SaveLLMConfig,
  SaveTelegramConfig,
  SaveSecurityConfig,
  SaveBrowserConfig,
  SavePluginsConfig,
  GetInstalledSkills,
  TestLLMConnection,
  TestTelegramConnection,
} from '../../wailsjs/go/main/App';
import ProviderForm from '../components/ProviderForm';
import ChannelForm from '../components/ChannelForm';

interface SkillInfo {
  name: string;
  version: string;
  description: string;
  author: string;
  enabled: boolean;
}

interface Props {
  onBack: () => void;
}

function Settings({ onBack }: Props) {
  const [provider, setProvider] = useState('openai');
  const [apiKey, setApiKey] = useState('');
  const [model, setModel] = useState('');
  const [baseURL, setBaseURL] = useState('');
  const [tgToken, setTgToken] = useState('');
  const [piiEnabled, setPiiEnabled] = useState(true);
  const [filterEmails, setFilterEmails] = useState(true);
  const [filterPhones, setFilterPhones] = useState(true);
  const [filterCards, setFilterCards] = useState(true);
  const [filterIPs, setFilterIPs] = useState(false);
  const [filterSSN, setFilterSSN] = useState(true);
  const [browserEnabled, setBrowserEnabled] = useState(false);
  const [browserHeadless, setBrowserHeadless] = useState(true);
  const [browserTimeout, setBrowserTimeout] = useState(30);
  const [browserMaxTabs, setBrowserMaxTabs] = useState(3);
  const [browserAllowed, setBrowserAllowed] = useState('');
  const [browserDenied, setBrowserDenied] = useState('');
  const [pluginsEnabled, setPluginsEnabled] = useState(true);
  const [pluginsTimeout, setPluginsTimeout] = useState(60);
  const [pluginsSandbox, setPluginsSandbox] = useState(true);
  const [skills, setSkills] = useState<SkillInfo[]>([]);
  const [enabledSkills, setEnabledSkills] = useState<string[]>([]);
  const [message, setMessage] = useState('');
  const [messageType, setMessageType] = useState<'success' | 'error'>('success');

  useEffect(() => {
    GetConfig().then((cfg: any) => {
      if (cfg) {
        setProvider(cfg.provider || 'openai');
        setModel(cfg.model || '');
        setBaseURL(cfg.base_url || '');
        setPiiEnabled(cfg.pii_filtering ?? true);
        setBrowserEnabled(cfg.browser_enabled ?? false);
        setBrowserHeadless(cfg.browser_headless ?? true);
        setPluginsEnabled(cfg.plugins_enabled ?? true);
      }
    });
    GetInstalledSkills().then((list: SkillInfo[]) => {
      if (list) {
        setSkills(list);
        setEnabledSkills(list.filter((s) => s.enabled).map((s) => s.name));
      }
    });
  }, []);

  const showMessage = (msg: string, type: 'success' | 'error') => {
    setMessage(msg);
    setMessageType(type);
    setTimeout(() => setMessage(''), 3000);
  };

  const saveLLM = async () => {
    try {
      await SaveLLMConfig(provider, apiKey, model, baseURL);
      showMessage('LLM settings saved', 'success');
    } catch (e: any) {
      showMessage(e.toString(), 'error');
    }
  };

  const saveTelegram = async () => {
    try {
      await SaveTelegramConfig(tgToken, []);
      showMessage('Telegram settings saved', 'success');
    } catch (e: any) {
      showMessage(e.toString(), 'error');
    }
  };

  const saveSecurity = async () => {
    try {
      await SaveSecurityConfig(piiEnabled, filterEmails, filterPhones, filterCards, filterIPs, filterSSN);
      showMessage('Security settings saved', 'success');
    } catch (e: any) {
      showMessage(e.toString(), 'error');
    }
  };

  const testLLM = async () => {
    try {
      const result = await TestLLMConnection(provider, apiKey, model, baseURL);
      showMessage(result === 'OK' ? 'Connection successful' : result, result === 'OK' ? 'success' : 'error');
    } catch (e: any) {
      showMessage(e.toString(), 'error');
    }
  };

  const testTG = async () => {
    try {
      const result = await TestTelegramConnection(tgToken);
      showMessage(result === 'OK' ? 'Connection successful' : result, result === 'OK' ? 'success' : 'error');
    } catch (e: any) {
      showMessage(e.toString(), 'error');
    }
  };

  const saveBrowser = async () => {
    try {
      await SaveBrowserConfig(browserEnabled, browserHeadless, browserTimeout, browserMaxTabs, browserAllowed, browserDenied);
      showMessage('Browser settings saved', 'success');
    } catch (e: any) {
      showMessage(e.toString(), 'error');
    }
  };

  const toggleSkill = (name: string) => {
    setEnabledSkills((prev) =>
      prev.includes(name) ? prev.filter((s) => s !== name) : [...prev, name]
    );
  };

  const savePlugins = async () => {
    try {
      await SavePluginsConfig(pluginsEnabled, enabledSkills, pluginsTimeout, pluginsSandbox);
      showMessage('Plugins settings saved', 'success');
    } catch (e: any) {
      showMessage(e.toString(), 'error');
    }
  };

  return (
    <div className="settings">
      <header className="settings-header">
        <button className="btn btn-secondary" onClick={onBack}>Back</button>
        <h1>Settings</h1>
      </header>

      {message && <div className={`message-banner ${messageType}`}>{message}</div>}

      <div className="settings-sections">
        <section className="settings-section">
          <h2>LLM Provider</h2>
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
            <button className="btn btn-secondary" onClick={testLLM} disabled={!apiKey}>Test</button>
            <button className="btn btn-primary" onClick={saveLLM}>Save</button>
          </div>
        </section>

        <section className="settings-section">
          <h2>Telegram</h2>
          <ChannelForm token={tgToken} onTokenChange={setTgToken} />
          <div className="button-row">
            <button className="btn btn-secondary" onClick={testTG} disabled={!tgToken}>Test</button>
            <button className="btn btn-primary" onClick={saveTelegram}>Save</button>
          </div>
        </section>

        <section className="settings-section">
          <h2>Security</h2>
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
                Emails
              </label>
              <label className="checkbox-label">
                <input type="checkbox" checked={filterPhones} onChange={(e) => setFilterPhones(e.target.checked)} />
                Phones
              </label>
              <label className="checkbox-label">
                <input type="checkbox" checked={filterCards} onChange={(e) => setFilterCards(e.target.checked)} />
                Cards
              </label>
              <label className="checkbox-label">
                <input type="checkbox" checked={filterIPs} onChange={(e) => setFilterIPs(e.target.checked)} />
                IPs
              </label>
              <label className="checkbox-label">
                <input type="checkbox" checked={filterSSN} onChange={(e) => setFilterSSN(e.target.checked)} />
                SSN
              </label>
            </div>
          )}
          <div className="button-row">
            <button className="btn btn-primary" onClick={saveSecurity}>Save</button>
          </div>
        </section>

        <section className="settings-section">
          <h2>Browser Control</h2>
          <div className="form-group">
            <label className="checkbox-label">
              <input type="checkbox" checked={browserEnabled} onChange={(e) => setBrowserEnabled(e.target.checked)} />
              Enable Browser Control
            </label>
          </div>
          {browserEnabled && (
            <>
              <div className="form-group">
                <label className="checkbox-label">
                  <input type="checkbox" checked={browserHeadless} onChange={(e) => setBrowserHeadless(e.target.checked)} />
                  Headless Mode
                </label>
                <span className="help-text">Run browser without visible window</span>
              </div>
              <div className="form-group">
                <label>Timeout (seconds)</label>
                <input
                  type="number"
                  className="input"
                  value={browserTimeout}
                  onChange={(e) => setBrowserTimeout(Number(e.target.value))}
                  min={5}
                  max={120}
                />
              </div>
              <div className="form-group">
                <label>Max Tabs</label>
                <input
                  type="number"
                  className="input"
                  value={browserMaxTabs}
                  onChange={(e) => setBrowserMaxTabs(Number(e.target.value))}
                  min={1}
                  max={10}
                />
              </div>
              <div className="form-group">
                <label>Allowed Domains (comma-separated)</label>
                <input
                  type="text"
                  className="input"
                  value={browserAllowed}
                  onChange={(e) => setBrowserAllowed(e.target.value)}
                  placeholder="e.g. example.com, github.com"
                />
                <span className="help-text">Leave empty to allow all domains</span>
              </div>
              <div className="form-group">
                <label>Denied Domains (comma-separated)</label>
                <input
                  type="text"
                  className="input"
                  value={browserDenied}
                  onChange={(e) => setBrowserDenied(e.target.value)}
                  placeholder="e.g. malware.com"
                />
              </div>
            </>
          )}
          <div className="button-row">
            <button className="btn btn-primary" onClick={saveBrowser}>Save</button>
          </div>
        </section>

        <section className="settings-section">
          <h2>Skills & Plugins</h2>
          <div className="form-group">
            <label className="checkbox-label">
              <input type="checkbox" checked={pluginsEnabled} onChange={(e) => setPluginsEnabled(e.target.checked)} />
              Enable Skills
            </label>
          </div>
          {pluginsEnabled && (
            <>
              <div className="form-group">
                <label className="checkbox-label">
                  <input type="checkbox" checked={pluginsSandbox} onChange={(e) => setPluginsSandbox(e.target.checked)} />
                  Sandbox Enabled
                </label>
              </div>
              <div className="form-group">
                <label>Timeout (seconds)</label>
                <input
                  type="number"
                  className="input"
                  value={pluginsTimeout}
                  onChange={(e) => setPluginsTimeout(Number(e.target.value))}
                  min={5}
                  max={300}
                />
              </div>
              <div className="skills-list">
                <label>Installed Skills</label>
                {skills.length === 0 ? (
                  <p className="help-text">No skills installed. Add skills to ~/.opendan/skills/</p>
                ) : (
                  skills.map((s) => (
                    <div key={s.name} className="skill-item">
                      <label className="checkbox-label">
                        <input
                          type="checkbox"
                          checked={enabledSkills.includes(s.name)}
                          onChange={() => toggleSkill(s.name)}
                        />
                        <div className="skill-info">
                          <strong>{s.name}</strong> <span className="skill-version">v{s.version}</span>
                          {s.description && <p className="help-text">{s.description}</p>}
                        </div>
                      </label>
                    </div>
                  ))
                )}
              </div>
            </>
          )}
          <div className="button-row">
            <button className="btn btn-primary" onClick={savePlugins}>Save</button>
          </div>
        </section>
      </div>
    </div>
  );
}

export default Settings;
