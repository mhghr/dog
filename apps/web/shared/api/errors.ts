export class ApiError extends Error {
  constructor(
    message: string,
    public status: number,
    public code?: string,
    public fields?: Record<string, string[]>,
    public requestId?: string,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

// Thrown for network-level failures (no response from the server).
export function networkError(): ApiError {
  return new ApiError("network_error", 0, "network_error");
}
