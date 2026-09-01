import { Component } from 'react'
import LogoMark from './LogoMark.jsx'

export default class ErrorBoundary extends Component {
  constructor(props) {
    super(props)
    this.state = { hasError: false }
  }

  static getDerivedStateFromError() {
    return { hasError: true }
  }

  componentDidCatch(error, info) {
    console.error('Lyrahcle dashboard crashed:', error, info)
  }

  handleReload = () => {
    window.location.reload()
  }

  render() {
    if (!this.state.hasError) return this.props.children

    return (
      <div className="flex min-h-screen items-center justify-center px-6">
        <div className="w-full max-w-sm text-center">
          <LogoMark className="mx-auto mb-5 h-12 w-12 opacity-70" />
          <p className="font-mono text-xs tracking-[0.3em] text-lyra-bad">
            Unexpected Error
          </p>
          <h1 className="mt-3 font-display text-lg font-medium text-lyra-text">
            Something went wrong
          </h1>
          <p className="mt-2 text-sm text-lyra-muted">
            This panel hit an unexpected error. Reloading usually fixes it — your
            session stays signed in.
          </p>
          <button
            type="button"
            onClick={this.handleReload}
            className="mt-6 rounded bg-lyra-gold px-4 py-2 text-sm font-medium text-lyra-bg outline-none transition-opacity hover:opacity-90 focus-visible:ring-2 focus-visible:ring-lyra-goldBright"
          >
            Reload
          </button>
        </div>
      </div>
    )
  }
}
