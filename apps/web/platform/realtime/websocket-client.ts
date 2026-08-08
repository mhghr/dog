// Thin WebSocket client with automatic reconnect and typed event dispatch.
// Used for high-frequency realtime feeds (live metrics, streams). The SSE
// client remains the default for low-frequency domain events.
type WsHandler = (data: unknown) => void;

const RETRY_BASE_MS = 1000;
const RETRY_MAX_MS = 30_000;

export class WebSocketClient {
  private socket: WebSocket | null = null;
  private handlers = new Map<string, Set<WsHandler>>();
  private retryTimer: ReturnType<typeof setTimeout> | null = null;
  private retryCount = 0;
  private closed = false;

  constructor(private url: string) {}

  connect() {
    if (this.socket || this.closed) {
      return;
    }
    try {
      this.socket = new WebSocket(this.url);
    } catch {
      return;
    }
    this.socket.addEventListener("open", () => {
      this.retryCount = 0;
    });
    this.socket.addEventListener("message", (event) => {
      let parsed: { type?: string; data?: unknown } = {};
      try {
        parsed = JSON.parse(String(event.data)) as { type?: string; data?: unknown };
      } catch {
        return;
      }
      if (!parsed.type) {
        return;
      }
      const set = this.handlers.get(parsed.type);
      if (!set) {
        return;
      }
      for (const handler of set) {
        handler(parsed.data);
      }
    });
    this.socket.addEventListener("close", () => {
      this.socket = null;
      if (!this.closed) {
        this.scheduleReconnect();
      }
    });
  }

  on(type: string, handler: WsHandler) {
    if (!this.handlers.has(type)) {
      this.handlers.set(type, new Set());
    }
    this.handlers.get(type)!.add(handler);
    return () => this.off(type, handler);
  }

  off(type: string, handler: WsHandler) {
    this.handlers.get(type)?.delete(handler);
  }

  close() {
    this.closed = true;
    if (this.retryTimer) {
      clearTimeout(this.retryTimer);
      this.retryTimer = null;
    }
    this.socket?.close();
    this.socket = null;
  }

  private scheduleReconnect() {
    const delay = Math.min(
      RETRY_BASE_MS * 2 ** this.retryCount,
      RETRY_MAX_MS,
    );
    this.retryCount += 1;
    this.retryTimer = setTimeout(() => this.connect(), delay);
  }
}
