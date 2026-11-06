import {
  Check,
  Copy,
  ExternalLink,
  Loader2,
  ShieldAlert,
  X,
} from "lucide-react";
import { useEffect, useRef, useState } from "react";

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
  const dialogRef = useRef<HTMLDialogElement | null>(null);
  // Latest onCancel without making it an effect dependency (the parent passes a fresh
  // arrow every render, which would otherwise close and reopen the dialog each time).
  const onCancelRef = useRef(onCancel);
  useEffect(() => {
    onCancelRef.current = onCancel;
  });

  // A native <dialog> opened with showModal() provides real modal behaviour that
  // `aria-modal` alone does not: focus moves into the dialog, Tab cannot reach the
  // (inert) background, Esc closes it, and focus returns to the opener on close.
  useEffect(() => {
    const dialog = dialogRef.current;
    if (!open || !dialog) return;
    if (!dialog.open) {
      dialog.showModal();
    }
    // Esc (or any other native close) must also abort the CLI login.
    const handleNativeClose = () => onCancelRef.current();
    dialog.addEventListener("close", handleNativeClose);
    return () => {
      dialog.removeEventListener("close", handleNativeClose);
      // Closing explicitly (rather than just unmounting) restores focus to the opener.
      if (dialog.open) {
        dialog.close();
      }
    };
  }, [open]);

  if (!open) return null;

  const copyAuthUrl = async () => {
    if (!authUrl) return;
    await navigator.clipboard.writeText(authUrl);
    setCopied(true);
    setTimeout(() => setCopied(false), 1800);
  };

  return (
    <dialog
      ref={dialogRef}
      className="modal-dialog"
      aria-labelledby="login-modal-title"
    >
      <div className="modal-card">
        <div className="modal-header">
          <div className="row-center">
            {errorMessage ? (
              <ShieldAlert size={19} className="err-icon" />
            ) : (
              <Loader2 size={19} className="spin accent-icon" />
            )}
            <h3 id="login-modal-title">Google Calendar Authorization</h3>
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
    </dialog>
  );
}
