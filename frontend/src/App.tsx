import { useState, useEffect } from 'react';
import './App.css';
import { IsSetupCompleted } from '../wailsjs/go/main/App';
import SetupWizard from './pages/SetupWizard';
import Dashboard from './pages/Dashboard';
import Settings from './pages/Settings';

type Page = 'loading' | 'wizard' | 'dashboard' | 'settings';

function App() {
  const [page, setPage] = useState<Page>('loading');

  useEffect(() => {
    IsSetupCompleted().then((completed: boolean) => {
      setPage(completed ? 'dashboard' : 'wizard');
    }).catch(() => {
      setPage('wizard');
    });
  }, []);

  if (page === 'loading') {
    return (
      <div className="app-loading">
        <div className="spinner" />
        <p>Loading OpenDan...</p>
      </div>
    );
  }

  return (
    <div className="app">
      {page === 'wizard' && (
        <SetupWizard onComplete={() => setPage('dashboard')} />
      )}
      {page === 'dashboard' && (
        <Dashboard onNavigate={setPage} />
      )}
      {page === 'settings' && (
        <Settings onBack={() => setPage('dashboard')} />
      )}
    </div>
  );
}

export default App;
