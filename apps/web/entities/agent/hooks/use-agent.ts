"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { agentApi } from "@/entities/agent/api/agent.api";
import type {
  AgentListResponse,
  AgentStatus,
  CreateTokenInput,
  EnrollmentToken,
  TokenListResponse,
} from "@/entities/agent/model/types";

export function useAgents(status?: AgentStatus) {
  const params = status ? `?status=${status}` : "";

  return useQuery({
    queryKey: ["probe-agents", status],
    queryFn: () => agentApi.list(params),
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
    mutationFn: (agentId: string) => agentApi.approve(agentId),
    onSuccess: invalidateAgents,
  });

  const reject = useMutation({
    mutationFn: (agentId: string) => agentApi.reject(agentId),
    onSuccess: invalidateAgents,
  });

  const disable = useMutation({
    mutationFn: (agentId: string) => agentApi.disable(agentId),
    onSuccess: invalidateAgents,
  });

  const enable = useMutation({
    mutationFn: (agentId: string) => agentApi.enable(agentId),
    onSuccess: invalidateAgents,
  });

  const revoke = useMutation({
    mutationFn: (agentId: string) => agentApi.revoke(agentId),
    onSuccess: invalidateAgents,
  });

  const drain = useMutation({
    mutationFn: (agentId: string) => agentApi.drain(agentId),
    onSuccess: invalidateAgents,
  });

  const updatePublicIP = useMutation({
    mutationFn: ({ agentId, publicIP }: { agentId: string; publicIP: string }) =>
      agentApi.publicIp(agentId, publicIP),
    onSuccess: invalidateAgents,
  });

  const updateLocation = useMutation({
    mutationFn: ({ agentId, city, country }: { agentId: string; city: string; country: string }) =>
      agentApi.location(agentId, city, country),
    onSuccess: invalidateAgents,
  });

  const deleteAgent = useMutation({
    mutationFn: (agentId: string) => agentApi.delete(agentId),
    onSuccess: invalidateAgents,
  });

  return { approve, reject, disable, enable, revoke, drain, deleteAgent, updatePublicIP, updateLocation };
}

export function useCreateEnrollmentToken() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: CreateTokenInput) => agentApi.createEnrollmentToken(input),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["probe-agents"] });
      void queryClient.invalidateQueries({ queryKey: ["probe-tokens"] });
    },
  });
}

export function useUnusedTokens() {
  return useQuery({
    queryKey: ["probe-tokens"],
    queryFn: () => agentApi.listEnrollmentTokens(),
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
