-- v10.4: propagate tenant ownership through the agent credential/config chain.

ALTER TABLE agent_enrollment_tokens ADD COLUMN IF NOT EXISTS organization_id UUID;
ALTER TABLE agent_certificates ADD COLUMN IF NOT EXISTS organization_id UUID;
ALTER TABLE agent_config_versions ADD COLUMN IF NOT EXISTS organization_id UUID;
ALTER TABLE agent_config_state ADD COLUMN IF NOT EXISTS organization_id UUID;
ALTER TABLE agent_config_acks ADD COLUMN IF NOT EXISTS organization_id UUID;
ALTER TABLE agent_keys ADD COLUMN IF NOT EXISTS organization_id UUID;

UPDATE agent_enrollment_tokens t SET organization_id=i.organization_id
FROM agent_identities i WHERE i.agent_id=t.agent_id AND t.organization_id IS NULL;
UPDATE agent_certificates c SET organization_id=i.organization_id
FROM agent_identities i WHERE i.agent_id=c.agent_id AND c.organization_id IS NULL;
UPDATE agent_config_versions c SET organization_id=i.organization_id
FROM agent_identities i WHERE i.agent_id=c.agent_id AND c.organization_id IS NULL;
UPDATE agent_config_state c SET organization_id=i.organization_id
FROM agent_identities i WHERE i.agent_id=c.agent_id AND c.organization_id IS NULL;
UPDATE agent_config_acks c SET organization_id=i.organization_id
FROM agent_identities i WHERE i.agent_id=c.agent_id AND c.organization_id IS NULL;
UPDATE agent_keys k SET organization_id=i.organization_id
FROM agent_identities i WHERE i.agent_id=k.agent_id AND k.organization_id IS NULL;

-- Legacy keys without a matching mTLS identity remain assigned to the default tenant.
UPDATE agent_keys SET organization_id='00000000-0000-0000-0000-000000000001'
WHERE organization_id IS NULL;

ALTER TABLE agent_enrollment_tokens ALTER COLUMN organization_id SET NOT NULL;
ALTER TABLE agent_certificates ALTER COLUMN organization_id SET NOT NULL;
ALTER TABLE agent_config_versions ALTER COLUMN organization_id SET NOT NULL;
ALTER TABLE agent_config_state ALTER COLUMN organization_id SET NOT NULL;
ALTER TABLE agent_config_acks ALTER COLUMN organization_id SET NOT NULL;
ALTER TABLE agent_keys ALTER COLUMN organization_id SET NOT NULL;

CREATE INDEX IF NOT EXISTS agent_enrollment_tokens_org_agent_idx ON agent_enrollment_tokens (organization_id, agent_id);
CREATE INDEX IF NOT EXISTS agent_certificates_org_agent_idx ON agent_certificates (organization_id, agent_id, status);
CREATE INDEX IF NOT EXISTS agent_config_versions_org_agent_idx ON agent_config_versions (organization_id, agent_id, version DESC);
CREATE INDEX IF NOT EXISTS agent_config_state_org_agent_idx ON agent_config_state (organization_id, agent_id);
CREATE INDEX IF NOT EXISTS agent_config_acks_org_agent_idx ON agent_config_acks (organization_id, agent_id, version DESC);
CREATE INDEX IF NOT EXISTS agent_keys_org_agent_idx ON agent_keys (organization_id, agent_id);
