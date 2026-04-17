CREATE TABLE IF NOT EXISTS agencies (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    legal_entity VARCHAR(255),
    contact_email VARCHAR(128),
    white_label_domain VARCHAR(255),
    white_label_logo_url TEXT,
    white_label_primary_color VARCHAR(20) DEFAULT '#1677ff',
    commission_rate DECIMAL(5,4) DEFAULT 0.10,
    status VARCHAR(20) DEFAULT 'active',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS agency_users (
    id BIGSERIAL PRIMARY KEY,
    agency_id BIGINT NOT NULL REFERENCES agencies(id),
    email VARCHAR(128) NOT NULL,
    name VARCHAR(255),
    role VARCHAR(30) NOT NULL,
    password_hash VARCHAR(128),
    status VARCHAR(20) DEFAULT 'active',
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (agency_id, email)
);

CREATE TABLE IF NOT EXISTS agency_advertisers (
    id BIGSERIAL PRIMARY KEY,
    agency_id BIGINT NOT NULL REFERENCES agencies(id),
    advertiser_id BIGINT NOT NULL,
    role VARCHAR(30) DEFAULT 'manage',
    added_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (agency_id, advertiser_id)
);

CREATE TABLE IF NOT EXISTS agency_commissions (
    id BIGSERIAL PRIMARY KEY,
    agency_id BIGINT NOT NULL,
    period_month DATE NOT NULL,
    total_advertiser_spend DECIMAL(15,4),
    commission_earned DECIMAL(15,4),
    status VARCHAR(20) DEFAULT 'pending',
    UNIQUE (agency_id, period_month)
);

CREATE TABLE IF NOT EXISTS user_permissions (
    id BIGSERIAL PRIMARY KEY,
    agency_user_id BIGINT NOT NULL REFERENCES agency_users(id),
    advertiser_id BIGINT,
    permission VARCHAR(30) NOT NULL,
    granted_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_agency_users_agency ON agency_users(agency_id);
CREATE INDEX IF NOT EXISTS idx_agency_advertisers_agency ON agency_advertisers(agency_id);
CREATE INDEX IF NOT EXISTS idx_agency_commissions_agency ON agency_commissions(agency_id);
CREATE INDEX IF NOT EXISTS idx_user_permissions_user ON user_permissions(agency_user_id);
