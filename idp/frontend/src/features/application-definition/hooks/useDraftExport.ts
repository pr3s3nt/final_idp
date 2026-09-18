import { useState } from 'react';
import { draftToDto, type ApplicationDraft } from '../draft/model';

function draftJson(draft: ApplicationDraft): string {
  return JSON.stringify(draftToDto(draft), null, 2);
}

export interface DraftExport {
  /** Result of the last copy attempt, shown next to the conflict actions. */
  copyStatus: string;
  clearStatus: () => void;
  copyDraft: () => Promise<void>;
  downloadDraft: () => void;
  canDownload: boolean;
}

/**
 * Lets a Developer keep a copy of the draft when Save reports a conflict and
 * the alternative is losing the unsaved work.
 */
export function useDraftExport(draft: ApplicationDraft): DraftExport {
  const [copyStatus, setCopyStatus] = useState('');

  const copyDraft = async () => {
    try {
      if (!navigator.clipboard?.writeText) throw new Error('Clipboard unavailable');
      await navigator.clipboard.writeText(draftJson(draft));
      setCopyStatus('Draft JSON copied.');
    } catch {
      setCopyStatus('Copy was not available. Download the draft instead.');
    }
  };

  const downloadDraft = () => {
    const url = URL.createObjectURL(new Blob([draftJson(draft)], { type: 'application/json' }));
    const anchor = document.createElement('a');
    anchor.href = url;
    anchor.download = `${draft.name || 'application'}-draft.json`;
    anchor.click();
    URL.revokeObjectURL(url);
  };

  return {
    copyStatus,
    clearStatus: () => setCopyStatus(''),
    copyDraft,
    downloadDraft,
    canDownload: typeof URL.createObjectURL === 'function',
  };
}
