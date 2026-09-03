import React from 'react'
import ReactDOM from 'react-dom/client'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import App from './App.tsx'
import ErrorBoundary from './components/ErrorBoundary.jsx'
import { I18nProvider } from './lib/I18nContext'
import { ToastProvider } from './components/ToastProvider.tsx'
import './index.css'

const queryClient = new QueryClient({ defaultOptions: { queries: { staleTime: 4000, retry: 1, refetchOnWindowFocus: true } } })

ReactDOM.createRoot(document.getElementById('root')).render(
  <React.StrictMode>
    <ErrorBoundary>
      <QueryClientProvider client={queryClient}>
        <I18nProvider>
          <ToastProvider>
            <App />
          </ToastProvider>
        </I18nProvider>
      </QueryClientProvider>
    </ErrorBoundary>
  </React.StrictMode>,
)
