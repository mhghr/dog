import { apiRequest } from "@/shared/api";
import { endpoints } from "@/shared/api/endpoints";
import type {
  AgentListResponse,
  AgentStatus,
  CreateTokenInput,
  EnrollmentToken,
  ProbeAgent,
  TokenListResponse,
} from "@/entities/agent/model/types";

export const agentApi = {
  list(params: string) {
    return apiRequest<AgentListResponse>(endpoints.agent.list(params));
  },

  byId(id: string) {
    return apiRequest<ProbeAgent>(endpoints.agent.byId(id));
  },

  approve(id: string) {
    return apiRequest<{ id: string; status: AgentStatus }>(endpoints.agent.approve(id), {
      method: "POST",
    });
  },

  reject(id: string) {
    return apiRequest<{ id: string; status: AgentStatus }>(endpoints.agent.reject(id), {
      method: "POST",
    });
  },

  disable(id: string) {
    return apiRequest<{ id: string; status: AgentStatus }>(endpoints.agent.disable(id), {
      method: "POST",
    });
  },

  enable(id: string) {
    return apiRequest<{ id: string; status: AgentStatus }>(endpoints.agent.enable(id), {
      method: "POST",
    });
  },

  revoke(id: string) {
    return apiRequest<{ id: string; status: AgentStatus }>(endpoints.agent.revoke(id), {
      method: "POST",
    });
  },

  drain(id: string) {
    return apiRequest<{ id: string; status: AgentStatus }>(endpoints.agent.drain(id), {
      method: "POST",
    });
  },

  publicIp(id: string, publicIP: string) {
    return apiRequest<{ status: string }>(endpoints.agent.publicIp(id), {
      method: "PUT",
      body: JSON.stringify({ public_ip: publicIP }),
    });
  },

  location(id: string, city: string, country: string) {
    return apiRequest<{ status: string }>(endpoints.agent.location(id), {
      method: "PUT",
      body: JSON.stringify({ city, country }),
    });
  },

  delete(id: string) {
    return apiRequest<{ status: string }>(endpoints.agent.byId(id), { method: "DELETE" });
  },

  createEnrollmentToken(input: CreateTokenInput) {
    return apiRequest<EnrollmentToken>(endpoints.agent.enrollmentTokens, {
      method: "POST",
      body: JSON.stringify(input),
    });
  },

  listEnrollmentTokens() {
    return apiRequest<TokenListResponse>(endpoints.agent.enrollmentTokens);
  },
};
