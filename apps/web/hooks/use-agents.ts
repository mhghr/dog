"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { apiRequest } from "@/lib/api-client";
import type {
  AgentListResponse,
  AgentStatus,
  CreateTokenInput,
  EnrollmentToken,
  TokenListResponse,
} from "@/types/agent";

export function useAgents(status?: AgentStatus) {
  const params = status ? `?status=${status}` : "";

  return useQuery({
    queryKey: ["probe-agents", status],
    queryFn: () =>
      apiRequest<AgentListResponse>(`/api/v1/admin/probe-agents${params}`),
    refetchInterval: 15_000,
    placeholderData: (previous) => previous,
  });
}

export function useAgentMutation() {
  const queryClient = useQueryClient();

  const invalidateAgents = () => {
    void queryClient.invalidateQueries({ queryKey: ["probe-agents"] });
  };

  const approve = useMutation({
    mutationFn: (agentId: string) =>
      apiRequest<{ id: string; status: AgentStatus }>(
        `/api/v1/admin/probe-agents/${agentId}/approve`,
        { method: "POST" },
      ),
    onSuccess: invalidateAgents,
  });

  const reject = useMutation({
    mutationFn: (agentId: string) =>
      apiRequest<{ id: string; status: AgentStatus }>(
        `/api/v1/admin/probe-agents/${agentId}/reject`,
        { method: "POST" },
      ),
    onSuccess: invalidateAgents,
  });

  const disable = useMutation({
    mutationFn: (agentId: string) =>
      apiRequest<{ id: string; status: AgentStatus }>(
        `/api/v1/admin/probe-agents/${agentId}/disable`,
        { method: "POST" },
      ),
    onSuccess: invalidateAgents,
  });

  const enable = useMutation({
    mutationFn: (agentId: string) =>
      apiRequest<{ id: string; status: AgentStatus }>(
        `/api/v1/admin/probe-agents/${agentId}/enable`,
        { method: "POST" },
      ),
    onSuccess: invalidateAgents,
  });

  const revoke = useMutation({
    mutationFn: (agentId: string) =>
      apiRequest<{ id: string; status: AgentStatus }>(
        `/api/v1/admin/probe-agents/${agentId}/revoke`,
        { method: "POST" },
      ),
    onSuccess: invalidateAgents,
  });

  const drain = useMutation({
    mutationFn: (agentId: string) =>
      apiRequest<{ id: string; status: AgentStatus }>(
        `/api/v1/admin/probe-agents/${agentId}/drain`,
        { method: "POST" },
      ),
    onSuccess: invalidateAgents,
  });

  const updatePublicIP = useMutation({
    mutationFn: ({ agentId, publicIP }: { agentId: string; publicIP: string }) =>
      apiRequest<{ status: string }>(
        `/api/v1/admin/probe-agents/${agentId}/public-ip`,
        { method: "PUT", body: JSON.stringify({ public_ip: publicIP }) },
      ),
    onSuccess: invalidateAgents,
  });

  const updateLocation = useMutation({
    mutationFn: ({ agentId, city, country }: { agentId: string; city: string; country: string }) =>
      apiRequest<{ status: string }>(
        `/api/v1/admin/probe-agents/${agentId}/location`,
        { method: "PUT", body: JSON.stringify({ city, country }) },
      ),
    onSuccess: invalidateAgents,
  });

  return { approve, reject, disable, enable, revoke, drain, updatePublicIP, updateLocation };
}

export function useCreateEnrollmentToken() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: CreateTokenInput) =>
      apiRequest<EnrollmentToken>("/api/v1/admin/probe-agent-enrollment-tokens", {
        method: "POST",
        body: JSON.stringify(input),
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["probe-agents"] });
      void queryClient.invalidateQueries({ queryKey: ["probe-tokens"] });
    },
  });
}

export function useUnusedTokens() {
  return useQuery({
    queryKey: ["probe-tokens"],
    queryFn: () =>
      apiRequest<TokenListResponse>("/api/v1/admin/probe-agent-enrollment-tokens"),
    refetchInterval: 30_000,
    placeholderData: (previous) => previous,
  });
}

export function useAgentStatusTransition(status: AgentStatus) {
  const actions: string[] = [];

  if (status === "pending") {
    actions.push("approve", "reject");
  } else if (status === "approved") {
    actions.push("disable", "revoke");
  } else if (status === "active") {
    actions.push("disable", "revoke", "drain");
  } else if (status === "offline") {
    actions.push("disable", "revoke");
  } else if (status === "disabled") {
    actions.push("enable", "revoke");
  } else if (status === "draining" || status === "updating") {
    actions.push("revoke");
  }

  return actions;
}
