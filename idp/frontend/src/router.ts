import { useEffect, useState } from 'react';

// A minimal History API router for the three UC-01 pages under /ui/.

export type Route =
  | { name: 'list' }
  | { name: 'new' }
  | { name: 'edit'; applicationId: string }
  | { name: 'not-found' };

export interface NavigationState {
  /** Version number just saved, shown as a success message. */
  savedVersion?: number;
  /** Increments on every navigation so pages re-initialize. */
  seq?: number;
}

export interface Location {
  pathname: string;
  state: NavigationState;
  key: number;
}

export function parseRoute(pathname: string): Route {
  const path = pathname.replace(/\/+$/, '');
  if (path === '/ui' || path === '/ui/applications') return { name: 'list' };
  if (path === '/ui/applications/new') return { name: 'new' };
  const match = /^\/ui\/applications\/([^/]+)$/.exec(path);
  if (match?.[1]) return { name: 'edit', applicationId: decodeURIComponent(match[1]) };
  return { name: 'not-found' };
}

let seq = 0;
const listeners = new Set<() => void>();

function current(): Location {
  const state = (window.history.state ?? {}) as NavigationState;
  return { pathname: window.location.pathname, state, key: state.seq ?? 0 };
}

export function navigate(path: string, state: NavigationState = {}): void {
  seq += 1;
  window.history.pushState({ ...state, seq }, '', path);
  listeners.forEach((l) => l());
}

export function useLocation(): Location {
  const [location, setLocation] = useState(current);
  useEffect(() => {
    const update = () => setLocation(current());
    listeners.add(update);
    window.addEventListener('popstate', update);
    return () => {
      listeners.delete(update);
      window.removeEventListener('popstate', update);
    };
  }, []);
  return location;
}

/** Click handler for in-app links that keeps normal browser behaviour for modified clicks. */
export function linkHandler(path: string) {
  return (event: React.MouseEvent<HTMLAnchorElement>) => {
    if (event.defaultPrevented || event.button !== 0 || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) return;
    event.preventDefault();
    navigate(path);
  };
}
