// Thin SSE client. Authenticated via the HttpOnly session cookie. Provides
// automatic reconnect with backoff and typed event subscription.
type SseHandler = (event: MessageEvent<string>) => void;

const RETRY_BASE_MS = 1000;
const RETRY_MAX_MS = 30_000;

export class SseClient {
  private source: EventSource | null = null;
  private handlers = new Map<string, Set<SseHandler>>();
  private retryTimer: ReturnType<typeof setTimeout> | null = null;
  private retryCount = 0;

  constructor(private url: string) {}

  connect() {
    if (this.source) {
      return;
    }
    try {
      this.source = new EventSource(this.url, { withCredentials: true });
    } catch {
      return;
    }
    this.source.addEventListener("open", () => {
      this.retryCount = 0;
    });
    this.source.addEventListener("error", () => {
      this.handleDisconnect();
    });
    for (const [event, set] of this.handlers) {
      for (const handler of set) {
        this.source.addEventListener(event, handler as EventListener);
      }
    }
  }

  on(event: string, handler: SseHandler) {
    if (!this.handlers.has(event)) {
      this.handlers.set(event, new Set());
    }
    this.handlers.get(event)!.add(handler);
    this.source?.addEventListener(event, handler as EventListener);
    return () => this.off(event, handler);
  }

  off(event: string, handler: SseHandler) {
    this.handlers.get(event)?.delete(handler);
    this.source?.removeEventListener(event, handler as EventListener);
  }

  close() {
    if (this.retryTimer) {
      clearTimeout(this.retryTimer);
      this.retryTimer = null;
    }
    this.source?.close();
    this.source = null;
  }

  private handleDisconnect() {
    this.source?.close();
    this.source = null;
    const delay = Math.min(
      RETRY_BASE_MS * 2 ** this.retryCount,
      RETRY_MAX_MS,
    );
    this.retryCount += 1;
    this.retryTimer = setTimeout(() => this.connect(), delay);
  }
}
