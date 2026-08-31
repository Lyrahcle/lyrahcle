-- v14.6.5: fail-closed staged rollout acknowledgement deadlines.
ALTER TABLE agent_fleet_rollouts
    ADD COLUMN IF NOT EXISTS ack_timeout_seconds INTEGER NOT NULL DEFAULT 900;

ALTER TABLE agent_fleet_rollouts
    DROP CONSTRAINT IF EXISTS agent_fleet_rollouts_ack_timeout_seconds_check;
ALTER TABLE agent_fleet_rollouts
    ADD CONSTRAINT agent_fleet_rollouts_ack_timeout_seconds_check
    CHECK (ack_timeout_seconds BETWEEN 30 AND 86400);

CREATE INDEX IF NOT EXISTS agent_fleet_targets_issued_deadline_idx
    ON agent_fleet_rollout_targets (organization_id, rollout_id, issued_at)
    WHERE status = 'issued';
