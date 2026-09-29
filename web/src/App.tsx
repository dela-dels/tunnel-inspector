import React, { useEffect } from 'react';
import { useInspectorStore } from './stores/useInspectorStore';
import { useWebSocket } from './hooks/useWebSocket';
import { Header } from './components/Header';
import { TunnelBanner } from './components/TunnelBanner';
import { RequestToolbar } from './components/RequestToolbar';
import { RequestList } from './components/RequestList';
import { RequestDetail } from './components/RequestDetail';

export const App: React.FC = () => {
  const fetchRequests = useInspectorStore((s) => s.fetchRequests);
  const fetchStatus = useInspectorStore((s) => s.fetchStatus);

  // Initialize live WebSocket stream
  useWebSocket();

  // Load initial data on mount
  useEffect(() => {
    fetchStatus();
    fetchRequests();
  }, [fetchStatus, fetchRequests]);

  return (
    <div className="flex flex-col h-screen w-screen bg-slate-950 text-slate-100 overflow-hidden font-sans">
      <Header />
      <TunnelBanner />
      <RequestToolbar />

      {/* Main Split-View Workspace */}
      <main className="flex-1 flex flex-col md:flex-row overflow-hidden">
        {/* Left Master List */}
        <section className="w-full md:w-[420px] lg:w-[460px] border-b md:border-b-0 md:border-r border-slate-800 flex flex-col bg-slate-900/30 overflow-hidden shrink-0">
          <RequestList />
        </section>

        {/* Right Detail Pane */}
        <section className="flex-1 flex flex-col overflow-hidden bg-slate-950">
          <RequestDetail />
        </section>
      </main>
    </div>
  );
};

export default App;
