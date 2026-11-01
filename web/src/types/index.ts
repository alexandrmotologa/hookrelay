export type EndpointStatus = 'ACTIVE' | 'SUSPENDED' | 'DISABLED';
export type DeliveryStatus = 'PENDING' | 'IN_FLIGHT' | 'SUCCESS' | 'RETRY' | 'FAILED';

export interface Endpoint {
  id: string;
  name: string;
  url: string;
  secret: string;
  event_types: string[];
  rate_limit: number;
  status: EndpointStatus;
  consecutive_failures: number;
  tripped_at?: string | null;
  created_at: string;
  updated_at: string;
}

export interface Delivery {
  id: string;
  message_id: string;
  endpoint_id: string;
  endpoint_name: string;
  endpoint_url: string;
  event_type: string;
  payload: string;
  status: DeliveryStatus;
  attempt_number: number;
  max_attempts: number;
  http_status?: number | null;
  duration_ms?: number | null;
  request_headers?: string;
  response_headers?: string;
  response_body?: string;
  error_message?: string;
  next_retry_at?: string | null;
  sent_at?: string | null;
  created_at: string;
}

export interface StatsSummary {
  total_messages: number;
  total_deliveries: number;
  successful_count: number;
  retrying_count: number;
  failed_count: number;
  pending_count: number;
  success_rate: number;
  avg_duration_ms: number;
  active_endpoints: number;
  suspended_endpoints: number;
}
