import { Component, type ErrorInfo, type ReactNode } from 'react';

/** Shows the failure instead of an empty page when a page throws. */
export class ErrorBoundary extends Component<{ children: ReactNode }, { error: Error | null }> {
  state: { error: Error | null } = { error: null };

  static getDerivedStateFromError(error: Error) {
    return { error };
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error('UI error', error, info.componentStack);
  }

  render() {
    if (!this.state.error) return this.props.children;
    return (
      <div className="banner banner-error" role="alert">
        <h2>The page stopped working</h2>
        <p>{this.state.error.message}</p>
        <p>Your draft is still stored in this browser tab. Reload the page to continue.</p>
        <button type="button" onClick={() => window.location.reload()}>
          Reload the page
        </button>
      </div>
    );
  }
}
