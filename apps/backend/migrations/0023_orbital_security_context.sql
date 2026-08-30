ALTER TABLE agent_identities ADD COLUMN IF NOT EXISTS station_latitude_deg DOUBLE PRECISION;
ALTER TABLE agent_identities ADD COLUMN IF NOT EXISTS station_longitude_deg DOUBLE PRECISION;
ALTER TABLE agent_identities ADD COLUMN IF NOT EXISTS station_altitude_m DOUBLE PRECISION;

ALTER TABLE agent_identities DROP CONSTRAINT IF EXISTS agent_identities_station_latitude_check;
ALTER TABLE agent_identities ADD CONSTRAINT agent_identities_station_latitude_check CHECK (station_latitude_deg IS NULL OR station_latitude_deg BETWEEN -90 AND 90);
ALTER TABLE agent_identities DROP CONSTRAINT IF EXISTS agent_identities_station_longitude_check;
ALTER TABLE agent_identities ADD CONSTRAINT agent_identities_station_longitude_check CHECK (station_longitude_deg IS NULL OR station_longitude_deg BETWEEN -180 AND 180);
ALTER TABLE agent_identities DROP CONSTRAINT IF EXISTS agent_identities_station_altitude_check;
ALTER TABLE agent_identities ADD CONSTRAINT agent_identities_station_altitude_check CHECK (station_altitude_m IS NULL OR station_altitude_m BETWEEN -500 AND 10000);
ALTER TABLE agent_identities DROP CONSTRAINT IF EXISTS agent_identities_station_complete_check;
ALTER TABLE agent_identities ADD CONSTRAINT agent_identities_station_complete_check CHECK (
    (station_latitude_deg IS NULL AND station_longitude_deg IS NULL AND station_altitude_m IS NULL)
    OR
    (station_latitude_deg IS NOT NULL AND station_longitude_deg IS NOT NULL AND station_altitude_m IS NOT NULL)
);

ALTER TABLE rf_readings ADD COLUMN IF NOT EXISTS satellite_id UUID REFERENCES satellites(id) ON DELETE SET NULL;
ALTER TABLE rf_anomalies ADD COLUMN IF NOT EXISTS satellite_id UUID REFERENCES satellites(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_rf_readings_satellite_captured ON rf_readings(satellite_id, captured_at DESC) WHERE satellite_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_rf_anomalies_satellite_detected ON rf_anomalies(satellite_id, detected_at DESC) WHERE satellite_id IS NOT NULL;
ALTER TABLE agent_config_versions ADD COLUMN IF NOT EXISTS satellite_id UUID REFERENCES satellites(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_agent_config_versions_satellite ON agent_config_versions(satellite_id, created_at DESC) WHERE satellite_id IS NOT NULL;
