import {
  Check,
  Copy,
  ExternalLink,
  Loader2,
  ShieldAlert,
  X,
} from "lucide-react";
import { useState } from "react";

interface LoginModalProps {
  open: boolean;
  authUrl: string | null;
  statusMessage: string;
  errorMessage: string | null;
  onOpenUrl: (url: string) => void;
  onCancel: () => void;
}

export function LoginModal({
  open,
  authUrl,
  statusMessage,
  errorMessage,
  onOpenUrl,
  onCancel,
}: LoginModalProps) {
  const [copied, setCopied] = useState(false);

  if (!open) return null;

  const copyAuthUrl = async () => {
    if (!authUrl) return;
    await navigator.clipboard.writeText(authUrl);
    setCopied(true);
    setTimeout(() => setCopied(false), 1800);
  };

  return (
    <div className="modal-backdrop" role="dialog" aria-modal="true">
      <div className="modal-card">
        <div className="modal-header">
          <div className="row-center">
            {errorMessage ? (
              <ShieldAlert size={19} className="err-icon" />
            ) : (
              <Loader2 size={19} className="spin accent-icon" />
            )}
            <h3>Google Calendar Authorization</h3>
          </div>
          <button
            type="button"
            className="icon-btn"
            onClick={onCancel}
            aria-label="Close authorization dialog"
          >
            <X size={16} />
          </button>
        </div>

        <p className="muted-text">{statusMessage}</p>

        {errorMessage && (
          <div className="inline-banner banner-danger">
            <span>{errorMessage}</span>
          </div>
        )}

        {authUrl && !errorMessage && (
          <div className="oauth-url-box">
            <span className="small-label">
              If your browser didn&apos;t open automatically, open or copy this
              link:
            </span>
            <code className="oauth-url-code">{authUrl}</code>
            <div className="row-center">
              <button
                type="button"
                className="btn btn-sm btn-primary"
                onClick={() => onOpenUrl(authUrl)}
              >
                Open in Browser
                <ExternalLink size={13} />
              </button>
              <button
                type="button"
                className="btn btn-sm btn-secondary"
                onClick={copyAuthUrl}
              >
                {copied ? <Check size={14} /> : <Copy size={14} />}
                {copied ? "Copied" : "Copy URL"}
              </button>
            </div>
          </div>
        )}

        <div className="modal-footer">
          <span className="muted-text small-text">
            Callback listener: <code>localhost:8085</code>
          </span>
          <button type="button" className="btn btn-secondary" onClick={onCancel}>
            {errorMessage ? "Close" : "Cancel Sign-In"}
          </button>
        </div>
      </div>
    </div>
  );
}
