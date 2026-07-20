CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    google_id TEXT,
    email TEXT,
    phone VARCHAR(20),
    name VARCHAR(200) NOT NULL DEFAULT '',
    avatar_url TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_login_at TIMESTAMPTZ,
    CHECK (google_id IS NOT NULL OR email IS NOT NULL OR phone IS NOT NULL)
);

CREATE UNIQUE INDEX users_google_id_idx ON users(google_id) WHERE google_id IS NOT NULL;
CREATE UNIQUE INDEX users_email_idx ON users(email) WHERE email IS NOT NULL;
CREATE UNIQUE INDEX users_phone_idx ON users(phone) WHERE phone IS NOT NULL;

CREATE TABLE auth_refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash CHAR(64) NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMPTZ,
    replaced_by UUID
);

CREATE INDEX auth_refresh_tokens_user_idx ON auth_refresh_tokens(user_id);

CREATE TABLE auth_otp_codes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    phone VARCHAR(20) NOT NULL,
    code_hash CHAR(64) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    attempts INTEGER NOT NULL DEFAULT 0,
    consumed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX auth_otp_codes_phone_idx ON auth_otp_codes(phone, created_at DESC);
